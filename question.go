package layaonnx

import (
	"errors"
	"fmt"
	"strings"
)

// NewChoice creates a question that selects one of at least two named options.
func NewChoice(instructions string, options ...Option) (Question, error) {
	if err := validateInstructions(instructions); err != nil {
		return Question{}, err
	}
	if len(options) < 2 {
		return Question{}, errors.New("choice requires at least two options")
	}
	cleaned := make([]Option, len(options))
	seen := make(map[string]struct{}, len(options))
	for index, option := range options {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			return Question{}, errors.New("choice option name is required")
		}
		if _, exists := seen[name]; exists {
			return Question{}, fmt.Errorf("duplicate choice option %q", name)
		}
		seen[name] = struct{}{}
		cleaned[index] = Option{Name: name, Description: option.Description}
	}
	return Question{typeName: QuestionChoice, instructions: instructions, options: cleaned}, nil
}

// NewScore creates a question over at least two ordered levels.
func NewScore(instructions string, levels ...string) (Question, error) {
	if err := validateInstructions(instructions); err != nil {
		return Question{}, err
	}
	if len(levels) < 2 {
		return Question{}, errors.New("score requires at least two levels")
	}
	for _, level := range levels {
		if strings.TrimSpace(level) == "" {
			return Question{}, errors.New("score level must not be empty")
		}
	}
	return Question{typeName: QuestionScore, instructions: instructions, levels: append([]string(nil), levels...)}, nil
}

// NewNoul creates a true-or-false probability question. Optional criteria
// describe what false and true mean for the question.
func NewNoul(instructions string, criteria ...BooleanCriteria) (Question, error) {
	if err := validateInstructions(instructions); err != nil {
		return Question{}, err
	}
	if len(criteria) > 1 {
		return Question{}, errors.New("noul accepts at most one criteria value")
	}
	var value BooleanCriteria
	if len(criteria) == 1 {
		value = criteria[0]
	}
	return Question{typeName: QuestionNoul, instructions: instructions, boolean: value}, nil
}

func validateInstructions(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("question instructions are required")
	}
	return nil
}
