package artifact_test

import (
	"context"
	"os"
	"testing"

	layaonnx "github.com/MstyAI/laya-onnx"
	"github.com/MstyAI/laya-onnx/artifact"
)

func TestPublishedArtifact(t *testing.T) {
	if os.Getenv("LAYA_ONNX_TEST_PUBLISHED") == "" {
		t.Skip("set LAYA_ONNX_TEST_PUBLISHED to test the public release")
	}
	store := artifact.Store{Root: t.TempDir()}
	paths, err := store.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	engine, err := layaonnx.Open(layaonnx.Options{
		ModelDir:           paths.ModelDir,
		RuntimeLibrary:     paths.RuntimeLibrary,
		ExecutionProviders: layaonnx.DefaultExecutionProviders(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	question, err := layaonnx.NewNoul("Does the customer ask for money back?")
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Evaluate(context.Background(), layaonnx.Request{
		State:     "I was charged twice. Please refund the duplicate.",
		Questions: map[string]layaonnx.Question{"refund": question},
	})
	if err != nil {
		t.Fatal(err)
	}
	answer := result.Answers["refund"]
	if answer.Noul == nil {
		t.Fatal("refund probability is missing")
	}
	if *answer.Noul < 0.8 {
		t.Fatalf("refund probability = %f", *answer.Noul)
	}
}
