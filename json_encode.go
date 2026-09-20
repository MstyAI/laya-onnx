package layaonnx

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON writes the System One wire shape while preserving Choice option
// order.
func (q Question) MarshalJSON() ([]byte, error) {
	var criteria any
	switch q.typeName {
	case QuestionChoice:
		value, err := marshalOptions(q.options)
		if err != nil {
			return nil, err
		}
		criteria = value
	case QuestionScore:
		criteria = q.levels
	case QuestionNoul:
		if q.boolean.False != "" || q.boolean.True != "" {
			criteria = map[string]string{"false": q.boolean.False, "true": q.boolean.True}
		}
	default:
		return nil, fmt.Errorf("%w: question type is missing", ErrInvalidRequest)
	}
	return json.Marshal(struct {
		Type         QuestionType `json:"type"`
		Instructions string       `json:"instructions"`
		Criteria     any          `json:"criteria,omitempty"`
	}{q.typeName, q.instructions, criteria})
}

func marshalOptions(options []Option) (json.RawMessage, error) {
	var value bytes.Buffer
	value.WriteByte('{')
	for index, option := range options {
		if index > 0 {
			value.WriteByte(',')
		}
		name, err := json.Marshal(option.Name)
		if err != nil {
			return nil, err
		}
		value.Write(name)
		value.WriteByte(':')
		if option.Description == "" {
			value.WriteString("null")
			continue
		}
		description, err := json.Marshal(option.Description)
		if err != nil {
			return nil, err
		}
		value.Write(description)
	}
	value.WriteByte('}')
	return value.Bytes(), nil
}

// MarshalJSON writes questions in stable identifier order.
func (r Request) MarshalJSON() ([]byte, error) {
	ids := sortedQuestionIDs(r.Questions)
	var questions bytes.Buffer
	questions.WriteByte('{')
	for index, id := range ids {
		if index > 0 {
			questions.WriteByte(',')
		}
		name, err := json.Marshal(id)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(r.Questions[id])
		if err != nil {
			return nil, err
		}
		questions.Write(name)
		questions.WriteByte(':')
		questions.Write(value)
	}
	questions.WriteByte('}')
	return json.Marshal(struct {
		State     any             `json:"state"`
		Questions json.RawMessage `json:"questions"`
	}{r.State, questions.Bytes()})
}
