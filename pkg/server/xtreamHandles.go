/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamesnetherton/m3u"
	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/config"
	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/utils"
	xtreamapi "github.com/pierre-emmanuelJ/iptv-proxy/pkg/xtream-proxy"
	uuid "github.com/satori/go.uuid"
	xtream "github.com/tellytv/go.xtream-codes"
)

type cacheMeta struct {
	string
	time.Time
}

var hlsChannelsRedirectURL map[string]url.URL = map[string]url.URL{}
var hlsChannelsRedirectURLLock = sync.RWMutex{}

// XXX Use key/value storage e.g: etcd, redis...
// and remove that dirty globals
var xtreamM3uCache map[string]cacheMeta = map[string]cacheMeta{}
var xtreamM3uCacheLock = sync.RWMutex{}

func (c *Config) cacheXtreamM3u(playlist *m3u.Playlist, cacheName string) error {
	xtreamM3uCacheLock.Lock()
	defer xtreamM3uCacheLock.Unlock()

	tmp := *c
	tmp.playlist = playlist

	path := filepath.Join(os.TempDir(), uuid.NewV4().String()+".iptv-proxy.m3u")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmp.marshallInto(f, true); err != nil {
		return err
	}
	xtreamM3uCache[cacheName] = cacheMeta{path, time.Now()}

	return nil
}

func (c *Config) xtreamGenerateM3u(ctx *gin.Context, extension string) (*m3u.Playlist, error) {
	client, err := xtreamapi.New(c.XtreamUser.String(), c.XtreamPassword.String(), c.XtreamBaseURL, ctx.Request.UserAgent())
	if err != nil {
		utils.DebugLog("-> ** xtreamGenerateM3u: xtreamapi.New - error: %s", err.Error())
		return nil, err
	}

	cat, err := client.GetLiveCategories()
	if err != nil {
		utils.DebugLog("-> ** xtreamGenerateM3u: client.GetLiveCategories - error: %s", err.Error())
		return nil, err
	}

	// this is specific to xtream API,
	// prefix with "live" if there is an extension.
	var prefix string
	if extension != "" {
		extension = "." + extension
		prefix = "live/"
	}

	var playlist = new(m3u.Playlist)
	playlist.Tracks = make([]m3u.Track, 0)

	for _, category := range cat {
		live, err := client.GetLiveStreams(fmt.Sprint(category.ID))
		if err != nil {
			utils.DebugLog("-> ** xtreamGenerateM3u: client.GetLiveStreams - error: %s", err.Error())
			return nil, err
		}

		for _, stream := range live {
			track := m3u.Track{Name: stream.Name, Length: -1, URI: "", Tags: nil}

			//TODO: Add more tag if needed.
			if stream.EPGChannelID != "" {
				track.Tags = append(track.Tags, m3u.Tag{Name: "tvg-id", Value: stream.EPGChannelID})
			}
			if stream.Name != "" {
				track.Tags = append(track.Tags, m3u.Tag{Name: "tvg-name", Value: stream.Name})
			}
			if stream.Icon != "" {
				track.Tags = append(track.Tags, m3u.Tag{Name: "tvg-logo", Value: stream.Icon})
			}
			if category.Name != "" {
				track.Tags = append(track.Tags, m3u.Tag{Name: "group-title", Value: category.Name})
			}

			track.URI = fmt.Sprintf("%s/%s%s/%s/%s%s", c.XtreamBaseURL, prefix, c.XtreamUser, c.XtreamPassword, fmt.Sprint(stream.ID), extension)
			playlist.Tracks = append(playlist.Tracks, track)
		}
	}

	return playlist, nil
}

func (c *Config) xtreamGetAuto(ctx *gin.Context) {
	newQuery := ctx.Request.URL.Query()
	q := c.RemoteURL.Query()
	for k, v := range q {
		if k == "username" || k == "password" {
			continue
		}

		newQuery.Add(k, strings.Join(v, ","))
	}
	ctx.Request.URL.RawQuery = newQuery.Encode()

	c.xtreamGet(ctx)
}

// replaceQueryParams replaces username and password values in the query string while preserving order
func replaceQueryParams(rawQuery string, username string, password string) string {
	rawParams := strings.Split(rawQuery, "&")
	var pairs []string

	for _, param := range rawParams {
		if param == "" {
			continue
		}
		parts := strings.SplitN(param, "=", 2)
		key := parts[0]
		var value string
		if len(parts) > 1 {
			value = parts[1]
		}

		// URL-unescape the value for processing
		unescapedValue, err := url.QueryUnescape(value)
		if err != nil {
			continue
		}

		// Replace username/password values if they match
		switch key {
		case "username":
			value = url.QueryEscape(username)
		case "password":
			value = url.QueryEscape(password)
		default:
			value = url.QueryEscape(unescapedValue)
		}

		pairs = append(pairs, key+"="+value)
	}

	return strings.Join(pairs, "&")
}

func (c *Config) xtreamGet(ctx *gin.Context) {
	// Create expURL with modified hostname and credentials
	expURL := *ctx.Request.URL

	// Parse the base URL to get scheme and host
	baseURL, err := url.Parse(c.XtreamBaseURL)
	if err != nil {
		log.Printf("-> ** xtreamGet: url.Parse baseURL - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	// Set scheme and host from the parsed URL
	expURL.Scheme = baseURL.Scheme
	expURL.Host = baseURL.Host

	// Replace username and password in query string while preserving order
	expURL.RawQuery = replaceQueryParams(expURL.RawQuery, c.XtreamUser.String(), c.XtreamPassword.String())

	rawURL := fmt.Sprintf("%s/get.php?username=%s&password=%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword)

	q := ctx.Request.URL.Query()

	for k, v := range q {
		if k == "username" || k == "password" {
			continue
		}

		rawURL = fmt.Sprintf("%s&%s=%s", rawURL, k, strings.Join(v, ","))
	}

	utils.DebugLog("|- xtreamGet expURL: %s", expURL.String())
	utils.DebugLog("|- xtreamGet rawURL: %s", rawURL)

	var urlToParse string
	if useExp := strings.ToLower(os.Getenv("USE_EXPERIMENTAL_URL")); useExp == "true" || useExp == "1" || useExp == "yes" {
		utils.DebugLog("Using experimental URL: %s", expURL.String())
		urlToParse = expURL.String()
	} else {
		urlToParse = rawURL
	}

	m3uURL, err := url.Parse(urlToParse)
	if err != nil {
		log.Printf("-> ** xtreamGet: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	xtreamM3uCacheLock.RLock()
	meta, ok := xtreamM3uCache[m3uURL.String()]
	d := time.Since(meta.Time)
	if !ok || d.Hours() >= float64(c.M3UCacheExpiration) {
		log.Printf("[iptv-proxy] %v | %s | xtream cache m3u file\n", time.Now().Format("2006/01/02 - 15:04:05"), ctx.ClientIP())
		xtreamM3uCacheLock.RUnlock()
		playlist, err := m3u.Parse(m3uURL.String())
		if err != nil {
			log.Printf("-> ** xtreamGet: m3u.Parse - error: %s", err.Error())
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
		if err := c.cacheXtreamM3u(&playlist, m3uURL.String()); err != nil {
			log.Printf("-> ** xtreamGet: c.cacheXtreamM3u - error: %s", err.Error())
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
	} else {
		xtreamM3uCacheLock.RUnlock()
	}

	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, c.M3UFileName))
	xtreamM3uCacheLock.RLock()
	path := xtreamM3uCache[m3uURL.String()].string
	xtreamM3uCacheLock.RUnlock()
	ctx.Header("Content-Type", "application/octet-stream")

	ctx.File(path)
}

func (c *Config) xtreamApiGet(ctx *gin.Context) {
	const (
		apiGet = "apiget"
	)

	var (
		extension = ctx.Query("output")
		cacheName = apiGet + extension
	)

	xtreamM3uCacheLock.RLock()
	meta, ok := xtreamM3uCache[cacheName]
	d := time.Since(meta.Time)
	if !ok || d.Hours() >= float64(c.M3UCacheExpiration) {
		log.Printf("[iptv-proxy] %v | %s | xtream cache API m3u file\n", time.Now().Format("2006/01/02 - 15:04:05"), ctx.ClientIP())
		xtreamM3uCacheLock.RUnlock()
		playlist, err := c.xtreamGenerateM3u(ctx, extension)
		if err != nil {
			log.Printf("-> ** xtreamApiGet: c.xtreamGenerateM3u - error: %s", err.Error())
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
		if err := c.cacheXtreamM3u(playlist, cacheName); err != nil {
			log.Printf("-> ** xtreamApiGet: c.cacheXtreamM3u - error: %s", err.Error())
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
	} else {
		xtreamM3uCacheLock.RUnlock()
	}

	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, c.M3UFileName))
	xtreamM3uCacheLock.RLock()
	path := xtreamM3uCache[cacheName].string
	xtreamM3uCacheLock.RUnlock()
	ctx.Header("Content-Type", "application/octet-stream")

	ctx.File(path)

}

func (c *Config) xtreamPlayerAPIGET(ctx *gin.Context) {
	c.xtreamPlayerAPI(ctx, ctx.Request.URL.Query())
}

func (c *Config) xtreamPlayerAPIPOST(ctx *gin.Context) {
	contents, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Printf("-> ** xtreamPlayerAPIPOST: ioutil.ReadAll - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	q, err := url.ParseQuery(string(contents))
	if err != nil {
		log.Printf("-> ** xtreamPlayerAPIPOST: url.ParseQuery - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamPlayerAPI(ctx, q)
}

func (c *Config) xtreamPlayerAPI(ctx *gin.Context, q url.Values) {
	var action string
	if len(q["action"]) > 0 {
		action = q["action"][0]
	}

	client, err := xtreamapi.New(c.XtreamUser.String(), c.XtreamPassword.String(), c.XtreamBaseURL, ctx.Request.UserAgent())
	if err != nil {
		utils.DebugLog("-> ** xtreamPlayerAPI: xtreamapi.New - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	resp, httpcode, err := client.Action(c.ProxyConfig, action, q)
	if err != nil {
		utils.DebugLog("-> ** xtreamPlayerAPI: client.Action - error: %s", err.Error())
		ctx.AbortWithError(httpcode, err) // nolint: errcheck
		return
	}

	log.Printf("[iptv-proxy] %v | %s |Action\t%s\n", time.Now().Format("2006/01/02 - 15:04:05"), ctx.ClientIP(), action)

	processedResp := ProcessResponse(resp)

	if config.CacheFolder != "" {
		readableJSON, _ := json.Marshal(processedResp)
		utils.WriteResponseToFile(ctx, readableJSON)
	}

	ctx.JSON(http.StatusOK, processedResp)
}

// ProcessResponse processes various types of xtream-codes responses
func ProcessResponse(resp interface{}) interface{} {

	respType := reflect.TypeOf(resp)

	switch {
	case respType == nil:
		return resp
	case strings.Contains(respType.String(), "[]xtreamcodes."):
		return processXtreamArray(resp)
	case strings.Contains(respType.String(), "xtreamcodes."):
		return processXtreamStruct(resp)
	default:
	}
	return resp
}

func processXtreamArray(arr interface{}) interface{} {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		return arr
	}

	if v.Len() == 0 {
		return arr
	}

	// Check if the first item is an xtreamcodes struct having a Fields field
	if !isXtreamCodesStruct(v.Index(0).Interface()) {
		return arr
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = processXtreamStruct(v.Index(i).Interface())
	}

	return result
}

// Define a helper function to check if fields exist
func hasFieldsField(item interface{}) bool {
	respValue := reflect.ValueOf(item)
	if respValue.Kind() == reflect.Ptr {
		respValue = respValue.Elem()
	}

	// Check for specific fields, e.g., "Fields"
	fieldValue := respValue.FieldByName(xtream.StructFields)
	return fieldValue.IsValid() && !fieldValue.IsNil()
}

func isXtreamCodesStruct(item interface{}) bool {
	respType := reflect.TypeOf(item)
	return respType != nil && strings.Contains(respType.String(), "xtreamcodes.") && hasFieldsField(item)
}

func processXtreamStruct(item interface{}) interface{} {
	if isXtreamCodesStruct(item) {
		respValue := reflect.ValueOf(item)
		if respValue.Kind() == reflect.Ptr {
			respValue = respValue.Elem()
		}

		fieldValue := respValue.FieldByName(xtream.StructFields)
		if fieldValue.IsValid() && !fieldValue.IsNil() {

			if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
				var unmarshaledValue interface{}
				err := json.Unmarshal(fieldValue.Interface().([]byte), &unmarshaledValue)
				if err != nil {
					utils.DebugLog("-- processXtreamStruct: JSON unmarshal error: %v", err)
					return fieldValue.Interface()
				}
				return unmarshaledValue
			}

			return fieldValue.Interface()
		}
	}
	return item
}

func (c *Config) xtreamXMLTV(ctx *gin.Context) {
	client, err := xtreamapi.New(c.XtreamUser.String(), c.XtreamPassword.String(), c.XtreamBaseURL, ctx.Request.UserAgent())
	if err != nil {
		utils.DebugLog("-> ** xtreamXMLTV: xtreamapi.New - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	resp, err := client.GetXMLTV()
	if err != nil {
		utils.DebugLog("-> ** xtreamXMLTV: client.GetXMLTV - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	// Write response to file
	// utils.WriteResponseToFile(ctx, resp)

	ctx.Data(http.StatusOK, "application/xml", resp)
}

func (c *Config) xtreamStreamHandler(ctx *gin.Context) {
	id := ctx.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/%s/%s/%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword, id))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamHandler: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, rpURL)
}

func (c *Config) xtreamStreamLive(ctx *gin.Context) {
	id := ctx.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/live/%s/%s/%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword, id))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamLive: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, rpURL)
}

func (c *Config) xtreamStreamPlay(ctx *gin.Context) {
	token := ctx.Param("token")
	t := ctx.Param("type")
	rpURL, err := url.Parse(fmt.Sprintf("%s/play/%s/%s", c.XtreamBaseURL, token, t))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamPlay: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, rpURL)
}

func (c *Config) xtreamStreamTimeshift(ctx *gin.Context) {
	duration := ctx.Param("duration")
	start := ctx.Param("start")
	id := ctx.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/timeshift/%s/%s/%s/%s/%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword, duration, start, id))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamTimeshift: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.stream(ctx, rpURL)
}

func (c *Config) xtreamStreamMovie(ctx *gin.Context) {
	id := ctx.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/movie/%s/%s/%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword, id))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamMovie: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, rpURL)
}

func (c *Config) xtreamStreamSeries(ctx *gin.Context) {
	id := ctx.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/series/%s/%s/%s", c.XtreamBaseURL, c.XtreamUser, c.XtreamPassword, id))
	if err != nil {
		utils.DebugLog("-> ** xtreamStreamSeries: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, rpURL)
}

func (c *Config) xtreamHlsStream(ctx *gin.Context) {
	chunk := ctx.Param("chunk")
	s := strings.Split(chunk, "_")
	if len(s) != 2 {
		utils.DebugLog("-> ** xtreamHlsStream: chunks parts != 2")
		ctx.AbortWithError( // nolint: errcheck
			http.StatusInternalServerError,
			errors.New("HSL malformed chunk"),
		)
		return
	}
	channel := s[0]

	url, err := getHlsRedirectURL(channel)
	if err != nil {
		utils.DebugLog("-> ** xtreamHlsStream: getHlsRedirectURL - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	req, err := url.Parse(
		fmt.Sprintf(
			"%s://%s/hls/%s/%s",
			url.Scheme,
			url.Host,
			ctx.Param("token"),
			ctx.Param("chunk"),
		),
	)

	if err != nil {
		utils.DebugLog("-> ** xtreamHlsStream: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, req)
}

func (c *Config) xtreamHlsrStream(ctx *gin.Context) {
	channel := ctx.Param("channel")

	url, err := getHlsRedirectURL(channel)
	if err != nil {
		utils.DebugLog("-> ** xtreamHlsrStream: getHlsRedirectURL - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	req, err := url.Parse(
		fmt.Sprintf(
			"%s://%s/hlsr/%s/%s/%s/%s/%s/%s",
			url.Scheme,
			url.Host,
			ctx.Param("token"),
			c.XtreamUser,
			c.XtreamPassword,
			ctx.Param("channel"),
			ctx.Param("hash"),
			ctx.Param("chunk"),
		),
	)

	if err != nil {
		utils.DebugLog("-> ** xtreamHlsrStream: url.Parse - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.xtreamStream(ctx, req)
}

func getHlsRedirectURL(channel string) (*url.URL, error) {
	hlsChannelsRedirectURLLock.RLock()
	defer hlsChannelsRedirectURLLock.RUnlock()

	url, ok := hlsChannelsRedirectURL[channel+".m3u8"]
	if !ok {
		utils.DebugLog("-> ** getHlsRedirectURL: hlsChannelsRedirectURL")
		return nil, errors.New("HSL redirect url not found")
	}

	return &url, nil
}

func (c *Config) hlsXtreamStream(ctx *gin.Context, oriURL *url.URL) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", oriURL.String(), nil)
	if err != nil {
		utils.DebugLog("-> ** hlsXtreamStream: http.NewRequest - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	mergeHttpHeader(req.Header, ctx.Request.Header)

	resp, err := client.Do(req)
	if err != nil {
		utils.DebugLog("-> ** hlsXtreamStream: client.Do - error: %s", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound {
		location, err := resp.Location()
		if err != nil {
			utils.DebugLog("-> ** hlsXtreamStream: resp.Location - error: %s", err.Error())
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
		id := ctx.Param("id")
		if strings.Contains(location.String(), id) {
			hlsChannelsRedirectURLLock.Lock()
			hlsChannelsRedirectURL[id] = *location
			hlsChannelsRedirectURLLock.Unlock()

			hlsReq, err := http.NewRequest("GET", location.String(), nil)
			if err != nil {
				utils.DebugLog("-> ** hlsXtreamStream: http.NewRequest - error: %s", err.Error())
				ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
				return
			}

			mergeHttpHeader(hlsReq.Header, ctx.Request.Header)

			hlsResp, err := client.Do(hlsReq)
			if err != nil {
				utils.DebugLog("-> ** hlsXtreamStream: client.Do - error: %s", err.Error())
				ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
				return
			}
			defer hlsResp.Body.Close()

			b, err := ioutil.ReadAll(hlsResp.Body)
			if err != nil {
				utils.DebugLog("-> ** hlsXtreamStream: ioutil.ReadAll - error: %s", err.Error())
				ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
				return
			}
			body := string(b)
			body = strings.ReplaceAll(body, "/"+c.XtreamUser.String()+"/"+c.XtreamPassword.String()+"/", "/"+c.User.String()+"/"+c.Password.String()+"/")

			mergeHttpHeader(ctx.Writer.Header(), hlsResp.Header)

			ctx.Data(http.StatusOK, hlsResp.Header.Get("Content-Type"), []byte(body))
			return
		}
		utils.DebugLog("-> ** hlsXtreamStream: Unable to HLS stream")
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Unable to HLS stream")) // nolint: errcheck
		return
	}

	ctx.Status(resp.StatusCode)
}
