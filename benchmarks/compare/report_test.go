package main

import (
	"testing"
	"time"
)

func TestMetricsSeparatesAccuracyStabilityAndLatency(t *testing.T) {
	cases := []benchmarkCase{
		{Expected: labels{Department: "billing", Refund: true, Urgency: 2}},
		{Expected: labels{Department: "sales", Refund: false, Urgency: 0}},
	}
	samples := []sample{
		{caseIndex: 0, prediction: prediction{Model: "test", Labels: labels{Department: "billing", Refund: true, Urgency: 1}}, latency: 10 * time.Millisecond},
		{caseIndex: 1, prediction: prediction{Model: "test", Labels: labels{Department: "sales", Refund: false, Urgency: 0}}, latency: 20 * time.Millisecond},
		{caseIndex: 0, prediction: prediction{Model: "test", Labels: labels{Department: "billing", Refund: true, Urgency: 1}}, latency: 30 * time.Millisecond},
		{caseIndex: 1, prediction: prediction{Model: "test", Labels: labels{Department: "sales", Refund: true, Urgency: 0}}, latency: 40 * time.Millisecond},
	}
	result := metrics(cases, samples)
	if result.CorrectQuestions != 5 || result.ExactCases != 1 {
		t.Fatalf("accuracy = %+v", result)
	}
	if result.StableRepeats != 1 || result.TotalRepeats != 2 {
		t.Fatalf("stability = %+v", result)
	}
	if result.LatencyP50MS != 30 || result.LatencyP95MS != 40 {
		t.Fatalf("latency = %+v", result)
	}
}
