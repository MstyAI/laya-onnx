package layaonnx

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

func TestPinnedModelParity(t *testing.T) {
	modelDir := os.Getenv("LAYA_ONNX_TEST_MODEL_DIR")
	runtimeLibrary := os.Getenv("LAYA_ONNX_TEST_RUNTIME")
	if modelDir == "" || runtimeLibrary == "" {
		t.Skip("set LAYA_ONNX_TEST_MODEL_DIR and LAYA_ONNX_TEST_RUNTIME")
	}
	engine, err := Open(Options{
		ModelDir: modelDir, RuntimeLibrary: runtimeLibrary,
		ExecutionProviders: DefaultExecutionProviders(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	department, _ := NewChoice("Which team should handle this?",
		Option{Name: "billing", Description: "payments and refunds"},
		Option{Name: "sales", Description: "purchases"},
		Option{Name: "technical", Description: "bugs"},
	)
	refund, _ := NewNoul("Does the customer ask for money back?")
	urgency, _ := NewScore("How urgent is this?", "not urgent", "soon", "urgent", "critical")
	started := time.Now()
	result, err := engine.Evaluate(context.Background(), Request{
		State: "I was charged twice and need the duplicate refunded today.",
		Questions: map[string]Question{
			"department": department,
			"refund":     refund,
			"urgency":    urgency,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answers["department"].Choice != "billing" {
		t.Fatalf("department = %+v", result.Answers["department"])
	}
	assertNear(t, result.Answers["department"].Probabilities["billing"], 0.9745, 0.005)
	assertNear(t, *result.Answers["refund"].Noul, 0.9421, 0.005)
	assertNear(t, *result.Answers["urgency"].Score, 1.5297, 0.01)
	t.Logf("three questions completed in %s", time.Since(started))

	difficultyOptions := make([]Option, 12)
	for index := range difficultyOptions {
		difficultyOptions[index] = Option{
			Name:        fmt.Sprintf("level_%d", index),
			Description: fmt.Sprintf("difficulty level %d", index),
		}
	}
	difficulty, err := NewChoice("How difficult is the requested work?", difficultyOptions...)
	if err != nil {
		t.Fatal(err)
	}
	structured, err := engine.Evaluate(context.Background(), Request{
		State: map[string]any{"request": "Refactor this service and investigate its deadlock."},
		Questions: map[string]Question{
			"difficulty": difficulty,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if structured.Answers["difficulty"].Choice != "level_1" {
		t.Fatalf("difficulty = %+v", structured.Answers["difficulty"])
	}
}

func assertNear(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("got %f, want %f ± %f", got, want, tolerance)
	}
}
