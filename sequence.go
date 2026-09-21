package layaonnx

import (
	"errors"
	"fmt"
	"strings"
)

const (
	maxSequenceTokens = 512
	maxHeadTokens     = 192
	maxOptionTokens   = 48
)

type preparedQuestion struct {
	id       string
	typeName QuestionType
	typeID   int64
	options  []string
	keys     []string
	legend   []string
	inputIDs []int64
	markers  []int64
}

func (e *Engine) prepareQuestion(id, state string, question Question) (preparedQuestion, error) {
	if err := validateQuestion(question); err != nil {
		return preparedQuestion{}, err
	}
	options, keys, typeID := renderQuestion(question)
	inputIDs, markers, err := e.buildSequence(question.typeName, question.instructions, options, state)
	if err != nil {
		return preparedQuestion{}, err
	}
	return preparedQuestion{
		id: id, typeName: question.typeName, typeID: typeID,
		options: options, keys: keys, legend: question.Levels(), inputIDs: inputIDs, markers: markers,
	}, nil
}

func validateQuestion(question Question) error {
	switch question.typeName {
	case QuestionChoice:
		_, err := NewChoice(question.instructions, question.options...)
		return err
	case QuestionScore:
		_, err := NewScore(question.instructions, question.levels...)
		return err
	case QuestionNoul:
		_, err := NewNoul(question.instructions, question.boolean)
		return err
	default:
		return errors.New("question type is missing")
	}
}

func renderQuestion(question Question) (options, keys []string, typeID int64) {
	switch question.typeName {
	case QuestionChoice:
		for _, option := range question.options {
			keys = append(keys, option.Name)
			if option.Description == "" {
				options = append(options, option.Name)
			} else {
				options = append(options, option.Name+": "+option.Description)
			}
		}
		return options, keys, 0
	case QuestionScore:
		for index, level := range question.levels {
			keys = append(keys, fmt.Sprint(index))
			options = append(options, fmt.Sprintf("level %d: %s", index, level))
		}
		return options, keys, 1
	default:
		falseMeaning := question.boolean.False
		trueMeaning := question.boolean.True
		if falseMeaning == "" {
			falseMeaning = "no, the statement does not hold"
		}
		if trueMeaning == "" {
			trueMeaning = "yes, the statement holds"
		}
		return []string{"false: " + falseMeaning, "true: " + trueMeaning}, []string{"false", "true"}, 2
	}
}

func (e *Engine) buildSequence(questionType QuestionType, instructions string, options []string, state string) ([]int64, []int64, error) {
	head := e.encode(string(questionType) + " question: " + sanitizeMask(instructions))
	optionIDs := e.encodeOptions(options)
	budget := headBudget(optionIDs)
	if budget < 16 {
		optionIDs = trimOptions(optionIDs)
		budget = headBudget(optionIDs)
	}
	if budget < 8 {
		return nil, nil, errors.New("question options are too large")
	}
	if len(head) > budget {
		head = head[:budget]
	}
	ids := append([]int64{e.tokens.classification}, head...)
	ids = append(ids, e.tokens.separator)
	markers := make([]int64, 0, len(optionIDs))
	for _, option := range optionIDs {
		markers = append(markers, int64(len(ids)))
		ids = append(ids, option...)
	}
	ids = append(ids, e.tokens.separator)
	room := maxSequenceTokens - len(ids) - 1
	if room < 0 {
		return nil, nil, errors.New("question is too large")
	}
	stateIDs := e.encode(sanitizeMask(state))
	if len(stateIDs) > room {
		stateIDs = stateIDs[:room]
	}
	ids = append(ids, stateIDs...)
	ids = append(ids, e.tokens.separator)
	return ids, markers, nil
}

func (e *Engine) encodeOptions(options []string) [][]int64 {
	result := make([][]int64, len(options))
	for index, option := range options {
		encoded := e.encode(" " + sanitizeMask(option))
		if len(encoded) > maxOptionTokens {
			encoded = encoded[:maxOptionTokens]
		}
		result[index] = append([]int64{e.tokens.mask}, encoded...)
	}
	return result
}

func headBudget(options [][]int64) int {
	total := 0
	for _, option := range options {
		total += len(option)
	}
	return maxHeadTokens - total
}

func trimOptions(options [][]int64) [][]int64 {
	perOption := max(4, (maxHeadTokens-16)/len(options))
	for index := range options {
		if len(options[index]) > perOption {
			options[index] = options[index][:perOption]
		}
	}
	return options
}

func (e *Engine) encode(text string) []int64 {
	encoded := e.tokenizer.Encode(text)
	result := make([]int64, len(encoded))
	for index, token := range encoded {
		result[index] = int64(token)
	}
	return result
}

func sanitizeMask(value string) string {
	return strings.ReplaceAll(value, "[MASK]", " ")
}
