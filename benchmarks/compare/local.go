package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	layaonnx "github.com/MstyAI/laya-onnx"
	"github.com/MstyAI/laya-onnx/artifact"
)

type localRunner struct {
	engine    *layaonnx.Engine
	questions map[string]layaonnx.Question
	loadTime  time.Duration
	memoryMiB float64
}

func openLocal(ctx context.Context, cpuOnly bool) (*localRunner, error) {
	store, err := artifact.DefaultStore()
	if err != nil {
		return nil, err
	}
	paths, err := store.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve local model: %w", err)
	}
	questions, err := benchmarkQuestions()
	if err != nil {
		return nil, err
	}
	started := time.Now()
	providers := layaonnx.DefaultExecutionProviders()
	if cpuOnly {
		providers = nil
	}
	engine, err := layaonnx.Open(layaonnx.Options{
		ModelDir: paths.ModelDir, RuntimeLibrary: paths.RuntimeLibrary,
		ExecutionProviders: providers,
	})
	if err != nil {
		return nil, err
	}
	return &localRunner{
		engine: engine, questions: questions,
		loadTime: time.Since(started),
	}, nil
}

func (r *localRunner) close() error { return r.engine.Close() }

func (r *localRunner) predict(ctx context.Context, state string) (prediction, error) {
	result, err := r.engine.Evaluate(ctx, layaonnx.Request{State: state, Questions: r.questions})
	if err != nil {
		return prediction{}, err
	}
	urgency := 0
	for index := 1; index < 4; index++ {
		if result.Answers["urgency"].Probabilities[strconv.Itoa(index)] > result.Answers["urgency"].Probabilities[strconv.Itoa(urgency)] {
			urgency = index
		}
	}
	refund := result.Answers["refund"].Noul != nil && *result.Answers["refund"].Noul >= 0.5
	return prediction{
		Labels: labels{Department: result.Answers["department"].Choice, Refund: refund, Urgency: urgency},
		Model:  result.Model,
		Usage:  usage{InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens},
	}, nil
}

func residentMemoryMiB() float64 {
	output, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return 0
	}
	kib, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0
	}
	return kib / 1024
}
