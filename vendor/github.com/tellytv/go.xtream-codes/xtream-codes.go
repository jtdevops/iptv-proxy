// Package xtreamcodes provides a Golang interface to the Xtream-Codes IPTV Server API.
package xtreamcodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/gin-gonic/gin"
	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/utils"
)

var defaultUserAgent = "go.xstream-codes (Go-http-client/1.1)"

var useAdvancedParsing bool

func init() {
	useAdvancedParsing = os.Getenv("USE_XTREAM_ADVANCED_PARSING") == "true"
}

// XtreamClient is the client used to communicate with a Xtream-Codes server.
type XtreamClient struct {
	Username  string
	Password  string
	BaseURL   string
	UserAgent string

	ServerInfo ServerInfo
	UserInfo   UserInfo

	// Our HTTP client to communicate with Xtream
	HTTP    *http.Client
	Context context.Context

	// We store an internal map of Streams for use with GetStreamURL
	streams map[int]Stream
}

// NewClient returns an initialized XtreamClient with the given values.
func NewClient(username, password, baseURL string) (*XtreamClient, error) {

	_, parseURLErr := url.Parse(baseURL)
	if parseURLErr != nil {
		return nil, fmt.Errorf("error parsing url: %s", parseURLErr.Error())
	}

	client := &XtreamClient{
		Username:  username,
		Password:  password,
		BaseURL:   baseURL,
		UserAgent: defaultUserAgent,

		HTTP:    http.DefaultClient,
		Context: context.Background(),

		streams: make(map[int]Stream),
	}

	authData, authErr := client.sendRequest("", nil)
	if authErr != nil {
		return nil, fmt.Errorf("error sending authentication request: %s", authErr.Error())
	}

	a := &AuthenticationResponse{}

	if jsonErr := json.Unmarshal(authData, &a); jsonErr != nil {
		debugLog("-> NewClient unmarshalling error for AuthenticationResponse - error: %s", jsonErr.Error())
		return nil, fmt.Errorf("error unmarshaling json: %s", jsonErr.Error())
	}

	client.ServerInfo = a.ServerInfo
	client.UserInfo = a.UserInfo

	return client, nil
}

// NewClientWithContext returns an initialized XtreamClient with the given values.
func NewClientWithContext(ctx context.Context, username, password, baseURL string) (*XtreamClient, error) {
	c, err := NewClient(username, password, baseURL)
	if err != nil {
		return nil, err
	}
	c.Context = ctx

	return c, nil
}

// NewClientWithUserAgent returns an initialized XtreamClient with the given values.
func NewClientWithUserAgent(ctx context.Context, username, password, baseURL, userAgent string) (*XtreamClient, error) {
	c, err := NewClient(username, password, baseURL)
	if err != nil {
		return nil, err
	}
	c.UserAgent = userAgent
	c.Context = ctx

	return c, nil
}

// GetStreamURL will return a stream URL string for the given streamID and wantedFormat.
func (c *XtreamClient) GetStreamURL(streamID int, wantedFormat string) (string, error) {

	// For Live Streams the main format is
	// http(s)://domain:port/live/username/password/streamID.ext ( In allowed_output_formats element you have the available ext )
	// For VOD Streams the format is:
	// http(s)://domain:port/movie/username/password/streamID.ext ( In target_container element you have the available ext )
	// For Series Streams the format is
	// http(s)://domain:port/series/username/password/streamID.ext ( In target_container element you have the available ext )

	validFormat := false

	for _, allowedFormat := range c.UserInfo.AllowedOutputFormats {
		if wantedFormat == allowedFormat {
			validFormat = true
		}
	}

	if !validFormat {
		return "", fmt.Errorf("%s is not an allowed output format", wantedFormat)
	}

	if _, ok := c.streams[streamID]; !ok {
		return "", fmt.Errorf("%d is not a valid stream id", streamID)
	}

	stream := c.streams[streamID]

	return fmt.Sprintf("%s/%s/%s/%s/%d.%s", c.BaseURL, stream.Type, c.Username, c.Password, stream.ID, wantedFormat), nil
}

// GetLiveCategories will return a slice of categories for live streams.
func (c *XtreamClient) GetLiveCategories() ([]Category, error) {
	return c.GetCategories("live")
}

// GetVideoOnDemandCategories will return a slice of categories for VOD streams.
func (c *XtreamClient) GetVideoOnDemandCategories() ([]Category, error) {
	return c.GetCategories("vod")
}

// GetSeriesCategories will return a slice of categories for series streams.
func (c *XtreamClient) GetSeriesCategories() ([]Category, error) {
	return c.GetCategories("series")
}

// GetCategories is a helper function used by GetLiveCategories, GetVideoOnDemandCategories and
// GetSeriesCategories to reduce duplicate code.
func (c *XtreamClient) GetCategories(catType string) ([]Category, error) {
	catData, catErr := c.sendRequest(fmt.Sprintf("get_%s_categories", catType), nil)
	if catErr != nil {
		return nil, catErr
	}

	cats := make([]Category, 0)

	if useAdvancedParsing {
		debugLog("- GetCategories using Advanced Parsing for: []Category")

		_, jsonErr := jsonparser.ArrayEach(catData, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil {
				debugLog(">> GetCategories - Error iterating through array: %s", err.Error())
				return
			}

			cat := Category{
				Fields: value, // Directly assign the raw JSON bytes
			}
			cats = append(cats, cat)
		})

		return cats, jsonErr
	} else {
		jsonErr := json.Unmarshal(catData, &cats)
		if jsonErr != nil {
			debugLog("-> GetCategories unmarshalling error for []Category - error: %s", jsonErr.Error())
		}

		for idx := range cats {
			cats[idx].Type = catType
		}

		return cats, jsonErr
	}

}

// GetLiveStreams will return a slice of live streams.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetLiveStreams(categoryID string) ([]Stream, error) {
	return c.GetStreams("live", categoryID)
}

// GetVideoOnDemandStreams will return a slice of VOD streams.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetVideoOnDemandStreams(categoryID string) ([]Stream, error) {
	return c.GetStreams("vod", categoryID)
}

// GetStreams is a helper function used by GetLiveStreams and GetVideoOnDemandStreams
// to reduce duplicate code.
func (c *XtreamClient) GetStreams(streamAction, categoryID string) ([]Stream, error) {
	var params url.Values
	if categoryID != "" {
		params = url.Values{}
		params.Add("category_id", categoryID)
	}

	// For whatever reason, unlike live and vod, series streams action doesn't have "_streams".
	if streamAction != "series" {
		streamAction = fmt.Sprintf("%s_streams", streamAction)
	}

	streamData, streamErr := c.sendRequest(fmt.Sprintf("get_%s", streamAction), params)
	if streamErr != nil {
		return nil, streamErr
	}

	streams := make([]Stream, 0)

	if useAdvancedParsing {
		debugLog("- GetStreams using Advanced Parsing for: []Stream")

		_, jsonErr := jsonparser.ArrayEach(streamData, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil {
				debugLog(">> GetStreams - Error iterating through array: %s", err.Error())
				return
			}

			stream := Stream{
				Fields: value,
			}

			streams = append(streams, stream)

			// The below code is to replicate the behavior of the old code:
			//   c.streams[int(stream.ID)] = stream

			streamID, _, _, err := jsonparser.Get(value, StreamFieldID)
			if err != nil {
				streamID = []byte("0")
			}
			streamType, _, _, err := jsonparser.Get(value, StreamFieldType)
			if err != nil {
				streamType = []byte("live")
			}

			// Now you can use streamID and streamType as []byte
			// To convert to string if needed:
			streamTypeStr := string(streamType)

			var flexID FlexInt
			flexID.UnmarshalJSON(streamID)

			s := Stream{
				ID:   flexID,
				Type: streamTypeStr,
			}
			c.streams[int(s.ID)] = s
		})

		return streams, jsonErr
	} else {
		if jsonErr := json.Unmarshal(streamData, &streams); jsonErr != nil {
			debugLog("-> GetStreams unmarshalling error for []Stream - error: %s", jsonErr.Error())
			return nil, jsonErr
		}

		for _, stream := range streams {
			c.streams[int(stream.ID)] = stream
		}

		return streams, nil
	}
}

// GetSeries will return a slice of all available Series.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetSeries(categoryID string) ([]SeriesInfo, error) {
	var params url.Values
	if categoryID != "" {
		params = url.Values{}
		params.Add("category_id", categoryID)
	}

	seriesData, seriesErr := c.sendRequest("get_series", params)
	if seriesErr != nil {
		return nil, seriesErr
	}

	seriesInfos := make([]SeriesInfo, 0)

	if useAdvancedParsing {
		debugLog("- GetSeries using Advanced Parsing for: []SeriesInfo")

		_, jsonErr := jsonparser.ArrayEach(seriesData, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil {
				debugLog(">> GetSeries - Error iterating through array: %s", err.Error())
				return
			}

			seriesInfo := SeriesInfo{
				Fields: value,
			}

			seriesInfos = append(seriesInfos, seriesInfo)
		})

		return seriesInfos, jsonErr
	} else {
		jsonErr := json.Unmarshal(seriesData, &seriesInfos)
		if jsonErr != nil {
			debugLog("-> GetSeries unmarshalling error for []SeriesInfo - error: %s", jsonErr.Error())
		}

		return seriesInfos, jsonErr
	}
}

// GetSeriesInfo will return a series info for the given seriesID.
func (c *XtreamClient) GetSeriesInfo(seriesID string) (*Series, error) {
	if seriesID == "" {
		return nil, fmt.Errorf("series ID can not be empty")
	}

	seriesData, seriesErr := c.sendRequest("get_series_info", url.Values{"series_id": []string{seriesID}})
	if seriesErr != nil {
		return nil, seriesErr
	}

	if useAdvancedParsing {
		debugLog("- GetSeriesInfo using Advanced Parsing for: Series")

		seriesInfo := &Series{
			Fields: seriesData,
		}

		return seriesInfo, nil
	} else {
		seriesInfo := &Series{}

		jsonErr := json.Unmarshal(seriesData, &seriesInfo)
		if jsonErr != nil {
			debugLog("-> GetSeriesInfo unmarshalling error for Series - error: %s", jsonErr.Error())
		}

		return seriesInfo, jsonErr
	}
}

// GetVideoOnDemandInfo will return VOD info for the given vodID.
func (c *XtreamClient) GetVideoOnDemandInfo(vodID string) (*VideoOnDemandInfo, error) {
	if vodID == "" {
		return nil, fmt.Errorf("vod ID can not be empty")
	}

	vodData, vodErr := c.sendRequest("get_vod_info", url.Values{"vod_id": []string{vodID}})
	if vodErr != nil {
		return nil, vodErr
	}

	if useAdvancedParsing {
		debugLog("- GetVideoOnDemandInfo using Advanced Parsing for: VideoOnDemandInfo")

		vodInfo := &VideoOnDemandInfo{
			Fields: vodData,
		}

		return vodInfo, nil
	} else {
		vodInfo := &VideoOnDemandInfo{}

		jsonErr := json.Unmarshal(vodData, &vodInfo)
		if jsonErr != nil {
			debugLog("- GetVideoOnDemandInfo unmarshalling error for VideoOnDemandInfo - error: %s", jsonErr.Error())
		}
		return vodInfo, jsonErr
	}
}

// GetShortEPG returns a short version of the EPG for the given streamID. If no limit is provided, the next 4 items in the EPG will be returned.
func (c *XtreamClient) GetShortEPG(streamID string, limit int) ([]EPGInfo, error) {
	return c.getEPG("get_short_epg", streamID, limit)
}

// GetEPG returns the full EPG for the given streamID.
func (c *XtreamClient) GetEPG(streamID string) ([]EPGInfo, error) {
	return c.getEPG("get_simple_data_table", streamID, 0)
}

// GetXMLTV will return a slice of bytes for the XMLTV EPG file available from the provider.
func (c *XtreamClient) GetXMLTV() ([]byte, error) {
	xmlTVData, xmlTVErr := c.sendRequest("xmltv.php", nil)
	if xmlTVErr != nil {
		return nil, xmlTVErr
	}

	return xmlTVData, xmlTVErr
}

func (c *XtreamClient) getEPG(action, streamID string, limit int) ([]EPGInfo, error) {
	if streamID == "" {
		return nil, fmt.Errorf("stream ID can not be empty")
	}

	params := url.Values{"stream_id": []string{streamID}}
	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}

	epgData, epgErr := c.sendRequest(action, params)
	if epgErr != nil {
		return nil, epgErr
	}

	epgContainer := &epgContainer{}

	jsonErr := json.Unmarshal(epgData, &epgContainer)
	if jsonErr != nil {
		debugLog("-> getEPG unmarshalling error for epgContainer - error: %s", jsonErr.Error())
	}

	return epgContainer.EPGListings, jsonErr
}

func (c *XtreamClient) sendRequest(action string, parameters url.Values) ([]byte, error) {
	data, _, err := c.sendRequestWithURL(action, parameters)
	return data, err
}

func (c *XtreamClient) sendRequestWithURL(action string, parameters url.Values) ([]byte, string, error) {
	file := "player_api.php"
	if action == "xmltv.php" {
		file = action
		action = ""
	}
	url := fmt.Sprintf("%s/%s?username=%s&password=%s", c.BaseURL, file, c.Username, c.Password)
	if action != "" {
		url = fmt.Sprintf("%s&action=%s", url, action)
	}
	if parameters != nil {
		url = fmt.Sprintf("%s&%s", url, parameters.Encode())
	}

	debugLog("<- Outgoing URL: %s", url)

	request, httpErr := http.NewRequest("GET", url, nil)
	if httpErr != nil {
		return nil, url, httpErr
	}

	request.Header.Set("User-Agent", c.UserAgent)

	request = request.WithContext(c.Context)

	response, httpErr := c.HTTP.Do(request)
	if httpErr != nil {
		return nil, url, fmt.Errorf("cannot reach server. %v", httpErr)
	}

	if response.StatusCode > 399 {
		return nil, url, fmt.Errorf("status code was %d, expected 2XX-3XX", response.StatusCode)
	}

	buf := &bytes.Buffer{}
	if _, copyErr := io.Copy(buf, response.Body); copyErr != nil {
		return nil, url, copyErr
	}

	if closeErr := response.Body.Close(); closeErr != nil {
		return nil, url, fmt.Errorf("cannot read response. %v", closeErr)
	}

	// Create a mock gin.Context for the WriteResponseToFile function
	mockCtx := &gin.Context{
		Request: &http.Request{
			URL: request.URL,
		},
	}

	// Write response to file
	utils.WriteResponseToFile(mockCtx, buf.Bytes())

	return buf.Bytes(), url, nil
}
