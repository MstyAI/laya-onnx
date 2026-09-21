package layaonnx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// DecodeRequest reads the same state/questions shape used by System One APIs.
func DecodeRequest(reader io.Reader) (Request, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var wire struct {
		State     json.RawMessage            `json:"state"`
		Questions map[string]json.RawMessage `json:"questions"`
	}
	if err := decoder.Decode(&wire); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if len(wire.State) == 0 || len(wire.Questions) == 0 {
		return Request{}, fmt.Errorf("%w: state and questions are required", ErrInvalidRequest)
	}
	if err := requireEOF(decoder); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	var state any
	stateDecoder := json.NewDecoder(bytes.NewReader(wire.State))
	stateDecoder.UseNumber()
	if err := stateDecoder.Decode(&state); err != nil {
		return Request{}, fmt.Errorf("%w: invalid state", ErrInvalidRequest)
	}
	questions := make(map[string]Question, len(wire.Questions))
	for id, raw := range wire.Questions {
		if id == "" {
			return Request{}, fmt.Errorf("%w: question id is required", ErrInvalidRequest)
		}
		question, err := decodeQuestion(raw)
		if err != nil {
			return Request{}, fmt.Errorf("%w: question %q: %v", ErrInvalidRequest, id, err)
		}
		questions[id] = question
	}
	requestState := state
	if _, isText := state.(string); !isText {
		requestState = encodedState(append([]byte(nil), wire.State...))
	}
	return Request{State: requestState, Questions: questions}, nil
}

type encodedState []byte

func (state encodedState) MarshalJSON() ([]byte, error) {
	if !json.Valid(state) {
		return nil, errors.New("invalid encoded state")
	}
	return append([]byte(nil), state...), nil
}

func decodeQuestion(raw json.RawMessage) (Question, error) {
	var value struct {
		Type         QuestionType    `json:"type"`
		Instructions string          `json:"instructions"`
		Criteria     json.RawMessage `json:"criteria"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return Question{}, err
	}
	switch value.Type {
	case QuestionChoice:
		options, err := decodeOptions(value.Criteria)
		if err != nil {
			return Question{}, err
		}
		return NewChoice(value.Instructions, options...)
	case QuestionScore:
		var levels []string
		if err := json.Unmarshal(value.Criteria, &levels); err != nil {
			return Question{}, errors.New("score criteria must be an array of strings")
		}
		return NewScore(value.Instructions, levels...)
	case QuestionNoul:
		if len(value.Criteria) == 0 || bytes.Equal(bytes.TrimSpace(value.Criteria), []byte("null")) {
			return NewNoul(value.Instructions)
		}
		var criteria struct {
			False string `json:"false"`
			True  string `json:"true"`
		}
		criteriaDecoder := json.NewDecoder(bytes.NewReader(value.Criteria))
		criteriaDecoder.DisallowUnknownFields()
		if err := criteriaDecoder.Decode(&criteria); err != nil {
			return Question{}, errors.New("noul criteria must describe false and true")
		}
		return NewNoul(value.Instructions, BooleanCriteria(criteria))
	default:
		return Question{}, fmt.Errorf("unsupported question type %q", value.Type)
	}
}

func decodeOptions(raw json.RawMessage) ([]Option, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("choice criteria must be an object")
	}
	var options []Option
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := key.(string)
		if !ok {
			return nil, errors.New("choice option name must be a string")
		}
		var description *string
		if err := decoder.Decode(&description); err != nil {
			return nil, errors.New("choice option description must be a string or null")
		}
		option := Option{Name: name}
		if description != nil {
			option.Description = *description
		}
		options = append(options, option)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return options, nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}
