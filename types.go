package layaonnx

// QuestionType identifies one of Laya's typed decision primitives.
type QuestionType string

const (
	// QuestionChoice selects one named option.
	QuestionChoice QuestionType = "choice"
	// QuestionScore evaluates an ordered set of levels.
	QuestionScore QuestionType = "score"
	// QuestionNoul estimates whether a statement is true.
	QuestionNoul QuestionType = "noul"
)

// Option is one allowed answer to a Choice question.
type Option struct {
	// Name is returned when this option is selected.
	Name string
	// Description tells the model when this option applies.
	Description string
}

// BooleanCriteria describes the meaning of false and true for a Noul question.
type BooleanCriteria struct {
	// False describes when the statement does not hold.
	False string
	// True describes when the statement holds.
	True string
}

// Question is created with NewChoice, NewScore, or NewNoul.
// Its fields are intentionally private so invalid combinations cannot be built.
type Question struct {
	typeName     QuestionType
	instructions string
	options      []Option
	levels       []string
	boolean      BooleanCriteria
}

// Type returns the question primitive.
func (q Question) Type() QuestionType { return q.typeName }

// Instructions returns the question sent to Laya.
func (q Question) Instructions() string { return q.instructions }

// Options returns a copy of a Choice question's options.
func (q Question) Options() []Option { return append([]Option(nil), q.options...) }

// Levels returns a copy of a Score question's ordered levels.
func (q Question) Levels() []string { return append([]string(nil), q.levels...) }

// BooleanCriteria returns a Noul question's false and true meanings.
func (q Question) BooleanCriteria() BooleanCriteria { return q.boolean }

// Request evaluates every named question against the same state.
// State may be a string or any value accepted by encoding/json.
type Request struct {
	// State is the shared input evaluated by every question.
	State any
	// Questions maps stable caller-defined identifiers to questions.
	Questions map[string]Question
}

// Usage reports the model input size. Laya produces no generated tokens.
type Usage struct {
	// InputTokens is the total number of tokens across every question row.
	InputTokens int64 `json:"input_tokens"`
	// OutputTokens is always zero because Laya classifies rather than generates.
	OutputTokens int64 `json:"output_tokens"`
}

// Answer contains the result for one question. Fields that do not apply to
// the question type are omitted from JSON.
type Answer struct {
	// Type identifies which answer fields are populated.
	Type QuestionType `json:"type"`
	// Choice is the selected option name for a Choice question.
	Choice string `json:"choice,omitempty"`
	// Noul is the probability of true for a Noul question.
	Noul *float64 `json:"noul,omitempty"`
	// Score is the expected level index for a Score question.
	Score *float64 `json:"score,omitempty"`
	// Probabilities maps Choice names or Score indices to probabilities.
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	// Confidence is the stronger side for Noul and one minus normalized entropy
	// for Choice and Score.
	Confidence *float64 `json:"confidence,omitempty"`
	// Legend maps Score indices to their level descriptions.
	Legend map[string]string `json:"legend,omitempty"`
}

// Result contains every answer from one batched evaluation.
type Result struct {
	// Model identifies the inference implementation.
	Model string `json:"model"`
	// Answers maps each request question identifier to its answer.
	Answers map[string]Answer `json:"answers"`
	// Usage reports the batch input size.
	Usage Usage `json:"usage"`
}
