package layaonnx

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"

	ort "github.com/shota3506/onnxruntime-purego/onnxruntime"
)

type inferenceBackend interface {
	Infer(context.Context, modelBatch) ([]float32, []int64, error)
	Close() error
}

type onnxBackend struct {
	runtime *ort.Runtime
	env     *ort.Env
	session *ort.Session
}

// DefaultExecutionProviders returns the accelerated provider available in the
// official ONNX Runtime package for the current platform. ONNX Runtime handles
// unsupported graph operations with its CPU provider.
func DefaultExecutionProviders() []string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return []string{"CoreMLExecutionProvider"}
	}
	return nil
}

func openONNXBackend(options Options) (inferenceBackend, error) {
	ortRuntime, err := ort.NewRuntime(options.RuntimeLibrary, 23)
	if err != nil {
		return nil, fmt.Errorf("load ONNX Runtime: %w", err)
	}
	environment, err := ortRuntime.NewEnv("laya-onnx", ort.LoggingLevelError)
	if err != nil {
		_ = ortRuntime.Close()
		return nil, fmt.Errorf("create ONNX environment: %w", err)
	}
	threads := options.Threads
	if threads <= 0 {
		threads = max(1, runtime.NumCPU()/2)
	}
	session, err := ortRuntime.NewSession(environment, filepath.Join(options.ModelDir, "model.onnx"), &ort.SessionOptions{
		IntraOpNumThreads:  threads,
		ExecutionProviders: append([]string(nil), options.ExecutionProviders...),
	})
	if err != nil {
		environment.Close()
		_ = ortRuntime.Close()
		return nil, fmt.Errorf("open ONNX model: %w", err)
	}
	wantInputs := []string{"input_ids", "attention_mask", "marker_pos", "marker_mask", "qtype"}
	if !slices.Equal(session.InputNames(), wantInputs) || !slices.Equal(session.OutputNames(), []string{"logits"}) {
		session.Close()
		environment.Close()
		_ = ortRuntime.Close()
		return nil, ErrModelContract
	}
	return &onnxBackend{runtime: ortRuntime, env: environment, session: session}, nil
}

func (b *onnxBackend) Infer(ctx context.Context, batch modelBatch) ([]float32, []int64, error) {
	// The binding passes Go-backed tensor memory to ONNX Runtime. Keep the
	// source slices live until the native call returns.
	defer runtime.KeepAlive(batch)
	sequenceShape := []int64{int64(batch.rows), int64(batch.sequence)}
	optionShape := []int64{int64(batch.rows), int64(batch.options)}
	typeShape := []int64{int64(batch.rows)}
	inputIDs, err := ort.NewTensorValue(b.runtime, batch.inputIDs, sequenceShape)
	if err != nil {
		return nil, nil, err
	}
	defer inputIDs.Close()
	attention, err := ort.NewTensorValue(b.runtime, batch.attentionMask, sequenceShape)
	if err != nil {
		return nil, nil, err
	}
	defer attention.Close()
	markers, err := ort.NewTensorValue(b.runtime, batch.markerPos, optionShape)
	if err != nil {
		return nil, nil, err
	}
	defer markers.Close()
	markerMask, err := ort.NewTensorValue(b.runtime, batch.markerMask, optionShape)
	if err != nil {
		return nil, nil, err
	}
	defer markerMask.Close()
	questionTypes, err := ort.NewTensorValue(b.runtime, batch.questionTypes, typeShape)
	if err != nil {
		return nil, nil, err
	}
	defer questionTypes.Close()
	outputs, err := b.session.Run(ctx, map[string]*ort.Value{
		"input_ids":      inputIDs,
		"attention_mask": attention,
		"marker_pos":     markers,
		"marker_mask":    markerMask,
		"qtype":          questionTypes,
	}, ort.WithOutputNames("logits"))
	if err != nil {
		return nil, nil, err
	}
	logits := outputs["logits"]
	if logits == nil {
		return nil, nil, ErrModelContract
	}
	defer logits.Close()
	return ort.GetTensorData[float32](logits)
}

func (b *onnxBackend) Close() error {
	if b.session != nil {
		b.session.Close()
		b.session = nil
	}
	if b.env != nil {
		b.env.Close()
		b.env = nil
	}
	if b.runtime == nil {
		return nil
	}
	err := b.runtime.Close()
	b.runtime = nil
	return err
}
