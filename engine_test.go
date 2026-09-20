package layaonnx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/gomlx/go-huggingface/tokenizers/api"
)

func TestEvaluateRejectsInvalidBatch(t *testing.T) {
	question, _ := NewNoul("Check")
	engine := testEngine(&fakeBackend{})
	if _, err := engine.Evaluate(context.Background(), Request{Questions: map[string]Question{"": question}}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("empty question id returned %v", err)
	}

	questions := make(map[string]Question, maxBatchQuestions+1)
	for index := 0; index <= maxBatchQuestions; index++ {
		questions[fmt.Sprint(index)] = question
	}
	if _, err := engine.Evaluate(context.Background(), Request{Questions: questions}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("large batch returned %v", err)
	}
}

type fakeTokenizer struct{}

func (fakeTokenizer) Encode(text string) []int {
	result := make([]int, len(text))
	for index := range text {
		result[index] = int(text[index]) + 100
	}
	return result
}
func (fakeTokenizer) With(api.EncodeOptions) error { return nil }
func (fakeTokenizer) TokenToID(token string) (int, bool) {
	values := map[string]int{"[CLS]": 1, "[SEP]": 2, "[PAD]": 3, "[MASK]": 4}
	value, ok := values[token]
	return value, ok
}

type fakeBackend struct {
	batch  modelBatch
	closed bool
}

func (b *fakeBackend) Infer(_ context.Context, batch modelBatch) ([]float32, []int64, error) {
	b.batch = batch
	logits := make([]float32, batch.rows*batch.options)
	for row := 0; row < batch.rows; row++ {
		for option := 0; option < batch.options; option++ {
			logits[row*batch.options+option] = float32(option)
		}
	}
	return logits, []int64{int64(batch.rows), int64(batch.options)}, nil
}
func (b *fakeBackend) Close() error { b.closed = true; return nil }

func TestEvaluateBatchesMixedQuestions(t *testing.T) {
	choice, _ := NewChoice("Pick", Option{Name: "a"}, Option{Name: "b"}, Option{Name: "c"})
	score, _ := NewScore("Rate", "low", "medium", "high", "critical")
	noul, _ := NewNoul("Check")
	backend := &fakeBackend{}
	engine := testEngine(backend)
	result, err := engine.Evaluate(context.Background(), Request{
		State:     map[string]any{"message": "hello"},
		Questions: map[string]Question{"choice": choice, "score": score, "noul": noul},
	})
	if err != nil {
		t.Fatal(err)
	}
	if backend.batch.rows != 3 || backend.batch.options != 4 {
		t.Fatalf("batch = %+v", backend.batch)
	}
	if result.Answers["choice"].Choice != "c" || result.Answers["noul"].Noul == nil || *result.Answers["noul"].Noul <= 0.5 {
		t.Fatalf("answers = %+v", result.Answers)
	}
	if result.Answers["noul"].Confidence == nil || *result.Answers["noul"].Confidence <= 0.5 {
		t.Fatalf("noul confidence = %+v", result.Answers["noul"])
	}
	if result.Answers["score"].Legend["3"] != "critical" {
		t.Fatalf("score = %+v", result.Answers["score"])
	}
	if result.Usage.InputTokens <= 0 || result.Usage.OutputTokens != 0 {
		t.Fatalf("usage = %+v", result.Usage)
	}
}

func TestEvaluateBoundsStateAndUsesCardinalityCalibration(t *testing.T) {
	question, _ := NewChoice("Pick", Option{Name: "a"}, Option{Name: "b"})
	backend := &fakeBackend{}
	engine := testEngine(backend)
	result, err := engine.Evaluate(context.Background(), Request{State: string(make([]byte, 2000)), Questions: map[string]Question{"route": question}})
	if err != nil {
		t.Fatal(err)
	}
	if backend.batch.sequence != maxSequenceTokens {
		t.Fatalf("sequence = %d", backend.batch.sequence)
	}
	want := math.Exp(0.5) / (1 + math.Exp(0.5))
	if got := result.Answers["route"].Probabilities["b"]; math.Abs(got-want) > 1e-6 {
		t.Fatalf("probability = %f, want %f", got, want)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	backend := &fakeBackend{}
	engine := testEngine(backend)
	if err := engine.Close(); err != nil || !backend.closed {
		t.Fatalf("close err=%v closed=%t", err, backend.closed)
	}
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
}

func testEngine(backend inferenceBackend) *Engine {
	return &Engine{
		tokenizer: fakeTokenizer{}, backend: backend,
		config: modelConfig{Temperature: []float64{1, 1, 1}, TemperatureByOptions: map[string]float64{"choice:2": 2}},
		tokens: specialTokens{classification: 1, separator: 2, padding: 3, mask: 4},
	}
}
