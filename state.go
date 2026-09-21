package layaonnx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func serializeState(state any) (string, error) {
	if text, ok := state.(string); ok {
		return text, nil
	}
	if raw, ok := state.(encodedState); ok {
		return formatStructuredState(raw)
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(state); err != nil {
		return "", fmt.Errorf("%w: state is not JSON-serializable", ErrInvalidRequest)
	}
	return formatStructuredState(encoded.Bytes())
}

func formatStructuredState(raw []byte) (string, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return "", fmt.Errorf("%w: state is not JSON-serializable", ErrInvalidRequest)
	}
	value := compact.Bytes()
	var formatted strings.Builder
	formatted.Grow(len(value) + bytes.Count(value, []byte{','}) + bytes.Count(value, []byte{':'}))
	inString, escaped := false, false
	for _, current := range value {
		formatted.WriteByte(current)
		if inString {
			if escaped {
				escaped = false
			} else if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		}
		if current == '"' {
			inString = true
		} else if current == ',' || current == ':' {
			formatted.WriteByte(' ')
		}
	}
	return formatted.String(), nil
}
