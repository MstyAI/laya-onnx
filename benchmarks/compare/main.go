package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
)

type predictor interface {
	predict(context.Context, string) (prediction, error)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "compare:", err)
		os.Exit(1)
	}
}

func run() error {
	casesPath := flag.String("cases", "benchmarks/support_cases.json", "labeled benchmark cases")
	outputPath := flag.String("output", "", "write JSON report to this path")
	runs := flag.Int("runs", 5, "measured passes over all cases")
	machine := flag.String("machine", "unspecified", "machine description for the report")
	jevModel := flag.String("jev-model", "jev-latest", "TypeSafe model alias")
	jevEndpoint := flag.String("jev-endpoint", "https://api.typesafe.ai/v1/systemone", "TypeSafe System One endpoint")
	localCPU := flag.Bool("local-cpu", false, "disable Core ML for the local model")
	flag.Parse()
	if *runs < 2 {
		return errors.New("runs must be at least 2")
	}
	apiKey := os.Getenv("TYPESAFE_API_KEY")
	if apiKey == "" {
		return errors.New("TYPESAFE_API_KEY is required")
	}
	cases, err := loadCases(*casesPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	local, err := openLocal(ctx, *localCPU)
	if err != nil {
		return err
	}
	defer local.close()
	jev := &jevRunner{
		apiKey: apiKey, endpoint: *jevEndpoint, model: *jevModel,
		client: &http.Client{Timeout: 30 * time.Second},
	}
	started := time.Now()
	if _, err := local.predict(ctx, cases[0].State); err != nil {
		return fmt.Errorf("local warmup: %w", err)
	}
	localWarmup := time.Since(started)
	local.memoryMiB = residentMemoryMiB()
	started = time.Now()
	if _, err := jev.predict(ctx, cases[0].State); err != nil {
		return fmt.Errorf("jev warmup: %w", err)
	}
	jevWarmup := time.Since(started)
	localSamples, err := measure(ctx, local, cases, *runs)
	if err != nil {
		return fmt.Errorf("local benchmark: %w", err)
	}
	jevSamples, err := measure(ctx, jev, cases, *runs)
	if err != nil {
		return fmt.Errorf("jev benchmark: %w", err)
	}
	localMetrics := metrics(cases, localSamples)
	localMetrics.ModelLoadMS = durationMS(local.loadTime)
	localMetrics.WarmupMS = durationMS(localWarmup)
	localMetrics.ResidentMemoryMiB = local.memoryMiB
	jevMetrics := metrics(cases, jevSamples)
	jevMetrics.WarmupMS = durationMS(jevWarmup)
	agreement := agreement(localSamples, jevSamples, len(cases))
	result := report{
		SchemaVersion: 1, GeneratedAt: time.Now().UTC(), Machine: *machine,
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Cases:    len(cases), QuestionsPerCase: 3, MeasuredRuns: *runs,
		JevEndpoint: *jevEndpoint, Local: localMetrics, Jev: jevMetrics,
		Agreement: ratio(agreement.all, agreement.total), AgreedQuestions: agreement.all, TotalComparisons: agreement.total,
		DepartmentAgreement: ratio(agreement.department, len(cases)), RefundAgreement: ratio(agreement.refund, len(cases)),
		UrgencyAgreement: ratio(agreement.urgency, len(cases)), CaseResults: comparisons(cases, localSamples, jevSamples),
		LocalLatencySamplesMS: latencySamples(localSamples), JevLatencySamplesMS: latencySamples(jevSamples),
	}
	result.LocalExecutionProvider = "CoreML + CPU fallback"
	if *localCPU {
		result.LocalExecutionProvider = "CPU"
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, raw, 0o600); err != nil {
			return err
		}
	}
	_, err = os.Stdout.Write(raw)
	return err
}

func loadCases(path string) ([]benchmarkCase, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cases []benchmarkCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, errors.New("benchmark has no cases")
	}
	seen := make(map[string]struct{}, len(cases))
	for _, current := range cases {
		if current.ID == "" || current.State == "" {
			return nil, errors.New("benchmark case id and state are required")
		}
		if _, exists := seen[current.ID]; exists {
			return nil, fmt.Errorf("duplicate benchmark case %q", current.ID)
		}
		seen[current.ID] = struct{}{}
		if current.Expected.Urgency < 0 || current.Expected.Urgency > 3 {
			return nil, fmt.Errorf("case %q has invalid urgency", current.ID)
		}
		switch current.Expected.Department {
		case "account", "billing", "sales", "shipping", "technical":
		default:
			return nil, fmt.Errorf("case %q has invalid department", current.ID)
		}
	}
	return cases, nil
}

func measure(ctx context.Context, runner predictor, cases []benchmarkCase, runs int) ([]sample, error) {
	samples := make([]sample, 0, len(cases)*runs)
	for run := 0; run < runs; run++ {
		for index, current := range cases {
			started := time.Now()
			result, err := runner.predict(ctx, current.State)
			if err != nil {
				return nil, fmt.Errorf("case %s run %d: %w", current.ID, run+1, err)
			}
			samples = append(samples, sample{caseIndex: index, prediction: result, latency: time.Since(started)})
		}
	}
	return samples, nil
}
