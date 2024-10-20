package xtreamcodes

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
)

// TODO: Add more flex types on IDs if needed
// for future potential provider issues.

// ServerInfo describes the state of the Xtream-Codes server.
type ServerInfo struct {
	HTTPSPort    FlexInt   `json:"https_port"`
	Port         FlexInt   `json:"port"`
	Process      bool      `json:"process"`
	RTMPPort     FlexInt   `json:"rtmp_port"`
	Protocol     string    `json:"server_protocol"`
	TimeNow      string    `json:"time_now"`
	TimestampNow Timestamp `json:"timestamp_now"`
	Timezone     string    `json:"timezone"`
	URL          string    `json:"url"`
}

// UserInfo is the current state of the user as it relates to the Xtream-Codes server.
type UserInfo struct {
	ActiveConnections    FlexInt            `json:"active_cons"`
	AllowedOutputFormats []string           `json:"allowed_output_formats"`
	Auth                 ConvertibleBoolean `json:"auth"`
	CreatedAt            Timestamp          `json:"created_at"`
	ExpDate              *Timestamp         `json:"exp_date"`
	IsTrial              ConvertibleBoolean `json:"is_trial"`
	MaxConnections       FlexInt            `json:"max_connections"`
	Message              string             `json:"message"`
	Password             string             `json:"password"`
	Status               string             `json:"status"`
	Username             string             `json:"username"`
}

// AuthenticationResponse is a container for what the server returns after the initial authentication.
type AuthenticationResponse struct {
	ServerInfo ServerInfo `json:"server_info"`
	UserInfo   UserInfo   `json:"user_info"`
}

// Category describes a grouping of Stream.
type Category struct {
	ID     FlexInt `json:"category_id"`
	Name   string  `json:"category_name"`
	Parent FlexInt `json:"parent_id"`

	// Set by us, not Xtream.
	Type string `json:"-"`
}

// Stream is a streamble video source.
type Stream struct {
	Added              *Timestamp `json:"added"`
	CategoryID         FlexInt    `json:"category_id"`
	CategoryName       string     `json:"category_name"`
	ContainerExtension string     `json:"container_extension"`
	CustomSid          string     `json:"custom_sid"`
	DirectSource       string     `json:"direct_source,omitempty"`
	EPGChannelID       string     `json:"epg_channel_id"`
	Icon               string     `json:"stream_icon"`
	ID                 FlexInt    `json:"stream_id"`
	Name               string     `json:"name"`
	Number             FlexInt    `json:"num"`
	Rating             FlexFloat  `json:"rating"`
	Rating5based       FlexFloat  `json:"rating_5based"`
	TVArchive          FlexInt    `json:"tv_archive"`
	TVArchiveDuration  *FlexInt   `json:"tv_archive_duration"`
	Type               string     `json:"stream_type"`
}

// SeriesInfo contains information about a TV series.
type SeriesInfo struct {
	BackdropPath   *JSONStringSlice `json:"backdrop_path,omitempty"`
	Cast           string           `json:"cast"`
	CategoryID     *FlexInt         `json:"category_id"`
	Cover          string           `json:"cover"`
	Director       string           `json:"director"`
	EpisodeRunTime string           `json:"episode_run_time"`
	Genre          string           `json:"genre"`
	LastModified   *Timestamp       `json:"last_modified,omitempty"`
	Name           string           `json:"name"`
	Num            FlexInt          `json:"num"`
	Plot           string           `json:"plot"`
	Rating         FlexInt          `json:"rating"`
	Rating5        FlexFloat        `json:"rating_5based"`
	ReleaseDate    string           `json:"releaseDate"`
	SeriesID       FlexInt          `json:"series_id"`
	StreamType     string           `json:"stream_type"`
	YoutubeTrailer string           `json:"youtube_trailer"`
}

type SeriesEpisode struct {
	Added              string       `json:"added"`
	ContainerExtension string       `json:"container_extension"`
	CustomSid          string       `json:"custom_sid"`
	DirectSource       string       `json:"direct_source"`
	EpisodeNum         FlexInt      `json:"episode_num"`
	ID                 string       `json:"id"`
	Info               *EpisodeInfo `json:"info,omitempty"`
	Season             FlexInt      `json:"season"`
	Title              string       `json:"title"`
}

type Series struct {
	Episodes map[string][]SeriesEpisode `json:"episodes"`
	Info     SeriesInfo                 `json:"info"`
	Seasons  []interface{}              `json:"seasons"`
}

// VideoOnDemandInfo contains information about a video on demand stream.
type VideoOnDemandInfo struct {
	Info      *VODInfo `json:"info,omitempty"`
	MovieData struct {
		Added              Timestamp `json:"added"`
		CategoryID         FlexInt   `json:"category_id"`
		ContainerExtension string    `json:"container_extension"`
		CustomSid          string    `json:"custom_sid"`
		DirectSource       string    `json:"direct_source"`
		Name               string    `json:"name"`
		StreamID           FlexInt   `json:"stream_id"`
	} `json:"movie_data"`
}

type VODInfo struct {
	Audio          *FFMPEGStreamInfo `json:"audio,omitempty"`
	BackdropPath   []string          `json:"backdrop_path"`
	Bitrate        FlexInt           `json:"bitrate"`
	Cast           string            `json:"cast"`
	Director       string            `json:"director"`
	Duration       string            `json:"duration"`
	DurationSecs   FlexInt           `json:"duration_secs"`
	Genre          string            `json:"genre"`
	MovieImage     string            `json:"movie_image"`
	Plot           string            `json:"plot"`
	Rating         FlexFloat         `json:"rating"`
	ReleaseDate    string            `json:"releasedate"`
	TmdbID         FlexInt           `json:"tmdb_id"`
	Video          *FFMPEGStreamInfo `json:"video,omitempty"`
	YoutubeTrailer string            `json:"youtube_trailer"`
}

type epgContainer struct {
	EPGListings []EPGInfo `json:"epg_listings"`
}

// EPGInfo describes electronic programming guide information of a stream.
type EPGInfo struct {
	ChannelID      string             `json:"channel_id"`
	Description    Base64Value        `json:"description"`
	End            string             `json:"end"`
	EPGID          FlexInt            `json:"epg_id"`
	HasArchive     ConvertibleBoolean `json:"has_archive"`
	ID             FlexInt            `json:"id"`
	Lang           string             `json:"lang"`
	NowPlaying     ConvertibleBoolean `json:"now_playing"`
	Start          string             `json:"start"`
	StartTimestamp Timestamp          `json:"start_timestamp"`
	StopTimestamp  Timestamp          `json:"stop_timestamp"`
	Title          Base64Value        `json:"title"`
}

type EpisodeInfo struct {
	Audio        *FFMPEGStreamInfo `json:"audio,omitempty"`
	Bitrate      FlexInt           `json:"bitrate"`
	Duration     string            `json:"duration"`
	DurationSecs FlexInt           `json:"duration_secs"`
	MovieImage   string            `json:"movie_image"`
	Name         string            `json:"name"`
	Plot         string            `json:"plot"`
	Rating       FlexFloat         `json:"rating"`
	ReleaseDate  string            `json:"releasedate"`
	Video        *FFMPEGStreamInfo `json:"video,omitempty"`
}

type EpisodesMapper struct {
	Episodes [][]SeriesEpisode	`json:"episodes"`
	Info	 SeriesInfo			`json:"info"`
	Seasons	 []interface{}		`json:"seasons"`		
}

// UnmarshalJSON implements custom unmarshaling for VideoOnDemandInfo
func (vod *VideoOnDemandInfo) UnmarshalJSON(data []byte) error {
	type Alias VideoOnDemandInfo
	aux := &struct {
		*Alias
		Info json.RawMessage `json:"info"`
	}{
		Alias: (*Alias)(vod),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for VideoOnDemandInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for VideoOnDemandInfo: %s", string(data))

	// Handle Info field
	if len(aux.Info) > 0 && string(aux.Info) != "\"\"" && string(aux.Info) != "[]" && string(aux.Info) != "[null]" {
		var info VODInfo
		if err := json.Unmarshal(aux.Info, &info); err != nil {
			log.Printf("Warning: Failed to unmarshal Info field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Info, &info, "Info"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		vod.Info = &info
	}

	// Log initial error and data only if subsequent unmarshalling fails
	if logInitialError && initialErr != nil {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for VODInfo
func (vi *VODInfo) UnmarshalJSON(data []byte) error {
	type Alias VODInfo
	aux := &struct {
		*Alias
		Audio json.RawMessage `json:"audio"`
		Video json.RawMessage `json:"video"`
	}{
		Alias: (*Alias)(vi),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for VODInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for VODInfo: %s", string(data))

	// Handle Audio field
	if len(aux.Audio) > 0 && string(aux.Audio) != "\"\"" && string(aux.Audio) != "[]" && string(aux.Audio) != "[null]" {
		var audio FFMPEGStreamInfo
		if err := json.Unmarshal(aux.Audio, &audio); err != nil {
			log.Printf("Warning: Failed to unmarshal Audio field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Audio, &audio, "Audio"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		vi.Audio = &audio
	}

	// Handle Video field
	if len(aux.Video) > 0 && string(aux.Video) != "\"\"" && string(aux.Video) != "[]" && string(aux.Video) != "[null]" {
		var video FFMPEGStreamInfo
		if err := json.Unmarshal(aux.Video, &video); err != nil {
			log.Printf("Warning: Failed to unmarshal Video field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Video, &video, "Video"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		vi.Video = &video
	}

	// Unmarshal remaining fields using reflective unmarshalling
	if err := unmarshalReflectiveFields(data, vi, "VODInfo"); err != nil {
		log.Printf("Warning: Error during reflective unmarshalling of VODInfo: %v", err)
		logInitialError = true
	}

	// Log initial error and data only if subsequent unmarshalling fails
	if logInitialError && initialErr != nil {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for SeriesEpisode
func (se *SeriesEpisode) UnmarshalJSON(data []byte) error {
	type Alias SeriesEpisode
	aux := &struct {
		*Alias
		Info json.RawMessage `json:"info"`
	}{
		Alias: (*Alias)(se),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for SeriesEpisode: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for SeriesEpisode: %s", string(data))

	// Handle Info field
	if len(aux.Info) > 0 && string(aux.Info) != "\"\"" && string(aux.Info) != "[]" && string(aux.Info) != "[null]" {
		var info EpisodeInfo
		if err := json.Unmarshal(aux.Info, &info); err != nil {
			log.Printf("Warning: Failed to unmarshal Info field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Info, &info, "Info"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		se.Info = &info
	}

	// Log initial error and data only if subsequent unmarshalling fails
	if logInitialError && initialErr != nil {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for EpisodeInfo
func (ei *EpisodeInfo) UnmarshalJSON(data []byte) error {
	type Alias EpisodeInfo
	aux := &struct {
		*Alias
		Video json.RawMessage `json:"video"`
		Audio json.RawMessage `json:"audio"`
	}{
		Alias: (*Alias)(ei),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for EpisodeInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for EpisodeInfo: %s", string(data))

	// Handle Video field
	if len(aux.Video) > 0 && string(aux.Video) != "\"\"" && string(aux.Video) != "[]" && string(aux.Video) != "[null]" {
		var video FFMPEGStreamInfo
		if err := json.Unmarshal(aux.Video, &video); err != nil {
			log.Printf("Warning: Failed to unmarshal Video field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Video, &video, "Video"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		ei.Video = &video
	}

	// Handle Audio field
	if len(aux.Audio) > 0 && string(aux.Audio) != "\"\"" && string(aux.Audio) != "[]" && string(aux.Audio) != "[null]" {
		var audio FFMPEGStreamInfo
		if err := json.Unmarshal(aux.Audio, &audio); err != nil {
			log.Printf("Warning: Failed to unmarshal Audio field. Using reflective unmarshalling.")

			if unmarshalErr := unmarshalReflectiveFields(aux.Audio, &audio, "Audio"); unmarshalErr != nil {
				logInitialError = true
			}
		}
		ei.Audio = &audio
	}

	// Log initial error and data only if subsequent unmarshalling fails
	if logInitialError && initialErr != nil {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for ServerInfo
func (si *ServerInfo) UnmarshalJSON(data []byte) error {
	type Alias ServerInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(si),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for ServerInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for ServerInfo: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal ServerInfo. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, si, "ServerInfo"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for UserInfo
func (ui *UserInfo) UnmarshalJSON(data []byte) error {
	type Alias UserInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ui),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for UserInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for UserInfo: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal UserInfo. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, ui, "UserInfo"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for AuthenticationResponse
func (ar *AuthenticationResponse) UnmarshalJSON(data []byte) error {
	type Alias AuthenticationResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ar),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for AuthenticationResponse: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for AuthenticationResponse: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal AuthenticationResponse. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, ar, "AuthenticationResponse"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for Category
func (c *Category) UnmarshalJSON(data []byte) error {
	type Alias Category
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for Category: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for Category: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal Category. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, c, "Category"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for Stream
func (s *Stream) UnmarshalJSON(data []byte) error {
	type Alias Stream
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for Stream: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for Stream: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal Stream. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, s, "Stream"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for SeriesInfo
func (si *SeriesInfo) UnmarshalJSON(data []byte) error {
	type Alias SeriesInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(si),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for SeriesInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for SeriesInfo: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal SeriesInfo. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, si, "SeriesInfo"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for Series
func (s *Series) UnmarshalJSON(data []byte) error {
	type Alias Series
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for Series: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for Series: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal Series. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, s, "Series"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// Mapper adds the mapped numbering of the series.episodes if it is not existing
func (ep *EpisodesMapper) Mapper (data []byte) []byte {
	if jsonErr := json.Unmarshal(data, &ep); jsonErr != nil {
		return data
	}

	episodesListUnmapped := ep.Episodes
	episodesListMapped := make(map[string][]SeriesEpisode)

	for i := range episodesListUnmapped {
		index := fmt.Sprintf("%d", i+1)
		episodesListMapped[index] = episodesListUnmapped[i]
	}

	result := map[string]interface{}{
		"seasons":  ep.Seasons,
		"info":     ep.Info,
		"episodes": episodesListMapped,
	}

	marshaledData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Warning: Failed to marshal remapped Season.Episodes.")
		return data
	}

	return marshaledData
}

// UnmarshalJSON implements custom unmarshaling for EPGInfo
func (ei *EPGInfo) UnmarshalJSON(data []byte) error {
	type Alias EPGInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ei),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for EPGInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for EPGInfo: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal EPGInfo. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, ei, "EPGInfo"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling for FFMPEGStreamInfo
func (fsi *FFMPEGStreamInfo) UnmarshalJSON(data []byte) error {
	type Alias FFMPEGStreamInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(fsi),
	}

	logInitialError := false
	initialErr := json.Unmarshal(data, &aux)
	errMsg := fmt.Sprintf("UnmarshalJSON error for FFMPEGStreamInfo: %v", initialErr)
	dataMsg := fmt.Sprintf("Problematic JSON data for FFMPEGStreamInfo: %s", string(data))

	if initialErr != nil {
		log.Printf("Warning: Failed to unmarshal FFMPEGStreamInfo. Using reflective unmarshalling.")
		if unmarshalErr := unmarshalReflectiveFields(data, fsi, "FFMPEGStreamInfo"); unmarshalErr != nil {
			logInitialError = true
		}
	}

	if logInitialError {
		log.Println(errMsg)
		log.Println(dataMsg)
	}

	return nil
}

func unmarshalReflectiveFields(data []byte, v interface{}, fieldName string) error {
	var objMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &objMap); err != nil {
		return fmt.Errorf("error unmarshaling %s: %v", fieldName, err)
	}

	valuePtr := reflect.ValueOf(v)
	if valuePtr.Kind() != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer", fieldName)
	}
	value := valuePtr.Elem()

	// Create a map to track which fields have been processed
	processedFields := make(map[string]bool)

	// Create a slice to store errors
	var errors []string

	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}
		if jsonTag == "-" {
			continue
		}
		jsonTag = strings.Split(jsonTag, ",")[0]

		processedFields[jsonTag] = true

		if rawValue, ok := objMap[jsonTag]; ok {
			// Check if the value is empty or an empty array
			if len(rawValue) == 0 || string(rawValue) == "\"\"" || string(rawValue) == "[]" || string(rawValue) == "[null]" {
				continue
			}

			fieldValue := value.Field(i)
			if fieldValue.CanSet() {
				err := json.Unmarshal(rawValue, fieldValue.Addr().Interface())
				if err != nil {
					errMsg := fmt.Sprintf("Error unmarshaling field %s.%s (value: %s): %v", fieldName, field.Name, string(rawValue), err)
					log.Printf("Warning: %s", errMsg)
					errors = append(errors, errMsg)
					// Continue with other fields instead of returning an error
				}
			}
		}
	}

	/*
	   // Log fields in the JSON that are not in the struct
	   for jsonField, rawValue := range objMap {
	       if !processedFields[jsonField] {
	           var value interface{}
	           err := json.Unmarshal(rawValue, &value)
	           if err != nil {
	               log.Printf("Warning: Error unmarshaling extra field %s.%s: %v", fieldName, jsonField, err)
	               // } else {
	               //  log.Printf("Extra field in %s: %s = %v", fieldName, jsonField, value)
	           }
	       }
	   }
	*/

	// If there were any errors during the process, return an error
	if len(errors) > 0 {
		return fmt.Errorf("unmarshalReflectiveFields encountered %d error(s) for %s: %s", len(errors), fieldName, strings.Join(errors, "; "))
	}

	return nil
}
