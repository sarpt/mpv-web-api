package mpv

import (
	"encoding/json"
	"errors"
)

type PlaylistFormatNodeMap struct {
	Filename string `json:"filename"`
	Current  bool   `json:"current"`
	Playing  bool   `json:"playing"`
	Title    string `json:"title"`
	ID       int64  `json:"id"`
}

type PlaylistFormatNodeArray = []PlaylistFormatNodeMap

// TODO: when generics land, converters should be rewritten to be generic on type of format node returned.
type FormatNodeMapConverter = func(data interface{}) (interface{}, error)

var (
	// ErrFormatNodeConversionDataNotString occurs when format node data is not a string containing node format.
	ErrFormatNodeConversionDataNotString = errors.New("format node data is not a string")
)

var (
	// FormatNodeConverters is a map of property names to their converter.
	// Converter transforms string received from MPV IPC, which stores MPV_FORMAT_NODE structure, into a type.
	// This list is needed for additional unmarshall of response payload data by command-dispatcher in
	// order to convert into a dedicated type instead of sending string data to be unmarshaled by client.
	FormatNodeConverters = map[string]FormatNodeMapConverter{
		PlaylistProperty: func(data interface{}) (interface{}, error) {
			var result PlaylistFormatNodeArray

			resultStr, ok := data.(string)
			if !ok {
				return result, ErrFormatNodeConversionDataNotString
			}

			err := json.Unmarshal([]byte(resultStr), &result)
			return result, err
		},
	}
)
