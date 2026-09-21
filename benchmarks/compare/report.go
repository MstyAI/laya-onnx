package main

import (
	"cmp"
	"slices"
	"time"
)

func metrics(cases []benchmarkCase, samples []sample) modelMetrics {
	first := make(map[int]prediction, len(cases))
	latencies := make([]time.Duration, 0, len(samples))
	stable, repeats := 0, 0
	var inputTokens, outputTokens int64
	for _, current := range samples {
		latencies = append(latencies, current.latency)
		inputTokens += current.prediction.Usage.InputTokens
		outputTokens += current.prediction.Usage.OutputTokens
		baseline, exists := first[current.caseIndex]
		if !exists {
			first[current.caseIndex] = current.prediction
			continue
		}
		repeats++
		if current.prediction.Labels == baseline.Labels {
			stable++
		}
	}
	correctQuestions, exactCases := 0, 0
	departmentCorrect, refundCorrect, urgencyCorrect := 0, 0, 0
	refundPositiveCorrect, refundPositiveTotal := 0, 0
	refundNegativeCorrect, refundNegativeTotal := 0, 0
	model := ""
	for index, current := range cases {
		prediction := first[index]
		model = prediction.Model
		correct := 0
		if prediction.Labels.Department == current.Expected.Department {
			correct++
			departmentCorrect++
		}
		if prediction.Labels.Refund == current.Expected.Refund {
			correct++
			refundCorrect++
			if current.Expected.Refund {
				refundPositiveCorrect++
			} else {
				refundNegativeCorrect++
			}
		}
		if current.Expected.Refund {
			refundPositiveTotal++
		} else {
			refundNegativeTotal++
		}
		if prediction.Labels.Urgency == current.Expected.Urgency {
			correct++
			urgencyCorrect++
		}
		correctQuestions += correct
		if correct == 3 {
			exactCases++
		}
	}
	positiveRecall := ratio(refundPositiveCorrect, refundPositiveTotal)
	negativeRecall := ratio(refundNegativeCorrect, refundNegativeTotal)
	return modelMetrics{
		Model:            model,
		CorrectQuestions: correctQuestions, TotalQuestions: len(cases) * 3,
		QuestionAccuracy:  ratio(correctQuestions, len(cases)*3),
		DepartmentCorrect: departmentCorrect, DepartmentAccuracy: ratio(departmentCorrect, len(cases)),
		RefundCorrect: refundCorrect, RefundAccuracy: ratio(refundCorrect, len(cases)),
		RefundPositiveRecall: positiveRecall, RefundNegativeRecall: negativeRecall,
		RefundBalancedAccuracy: (positiveRecall + negativeRecall) / 2,
		UrgencyCorrect:         urgencyCorrect, UrgencyAccuracy: ratio(urgencyCorrect, len(cases)),
		ExactCases: exactCases, TotalCases: len(cases), ExactCaseAccuracy: ratio(exactCases, len(cases)),
		StableRepeats: stable, TotalRepeats: repeats, RepeatStability: ratio(stable, repeats),
		LatencyMeanMS: durationMS(meanDuration(latencies)),
		LatencyP50MS:  durationMS(percentile(latencies, 0.50)),
		LatencyP95MS:  durationMS(percentile(latencies, 0.95)),
		InputTokens:   inputTokens, OutputTokens: outputTokens,
	}
}

func agreement(local, jev []sample, cases int) agreementMetrics {
	localFirst := firstPredictions(local, cases)
	jevFirst := firstPredictions(jev, cases)
	result := agreementMetrics{total: cases * 3}
	for index := 0; index < cases; index++ {
		if localFirst[index].Department == jevFirst[index].Department {
			result.all++
			result.department++
		}
		if localFirst[index].Refund == jevFirst[index].Refund {
			result.all++
			result.refund++
		}
		if localFirst[index].Urgency == jevFirst[index].Urgency {
			result.all++
			result.urgency++
		}
	}
	return result
}

func comparisons(cases []benchmarkCase, local, jev []sample) []caseResult {
	localFirst := firstPredictions(local, len(cases))
	jevFirst := firstPredictions(jev, len(cases))
	result := make([]caseResult, len(cases))
	for index, current := range cases {
		result[index] = caseResult{ID: current.ID, Expected: current.Expected, Local: localFirst[index], Jev: jevFirst[index]}
	}
	return result
}

func latencySamples(samples []sample) []float64 {
	result := make([]float64, len(samples))
	for index, current := range samples {
		result[index] = durationMS(current.latency)
	}
	return result
}

func firstPredictions(samples []sample, count int) []labels {
	result := make([]labels, count)
	seen := make([]bool, count)
	for _, current := range samples {
		if !seen[current.caseIndex] {
			result[current.caseIndex] = current.prediction.Labels
			seen[current.caseIndex] = true
		}
	}
	return result
}

func percentile(values []time.Duration, fraction float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	ordered := slices.Clone(values)
	slices.SortFunc(ordered, cmp.Compare[time.Duration])
	index := int(float64(len(ordered)-1)*fraction + 0.5)
	return ordered[index]
}

func meanDuration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
