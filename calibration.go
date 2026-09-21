package layaonnx

import "math"

func (e *Engine) answer(question preparedQuestion, logits []float32) Answer {
	probabilities := softmax(logits, e.temperature(question))
	answer := Answer{Type: question.typeName}
	switch question.typeName {
	case QuestionNoul:
		value := probabilities[1]
		answer.Noul = &value
		confidence := math.Max(value, 1-value)
		answer.Confidence = &confidence
	case QuestionChoice:
		answer.Probabilities = probabilityMap(question.keys, probabilities)
		answer.Choice = question.keys[argmax(probabilities)]
		confidence := entropyConfidence(probabilities)
		answer.Confidence = &confidence
	case QuestionScore:
		answer.Probabilities = probabilityMap(question.keys, probabilities)
		answer.Legend = make(map[string]string, len(question.options))
		score := 0.0
		for index, level := range question.legend {
			answer.Legend[question.keys[index]] = level
			score += float64(index) * probabilities[index]
		}
		answer.Score = &score
		confidence := entropyConfidence(probabilities)
		answer.Confidence = &confidence
	}
	return answer
}

func (e *Engine) temperature(question preparedQuestion) float64 {
	count := len(question.options)
	bucket := string(question.typeName) + ":11+"
	if count == 2 {
		bucket = string(question.typeName) + ":2"
	} else if count <= 5 {
		bucket = string(question.typeName) + ":3-5"
	} else if count <= 10 {
		bucket = string(question.typeName) + ":6-10"
	}
	if value := e.config.TemperatureByOptions[bucket]; value > 0 {
		return value
	}
	return e.config.Temperature[question.typeID]
}

func softmax(values []float32, temperature float64) []float64 {
	maximum := float64(values[0]) / temperature
	for _, value := range values[1:] {
		maximum = math.Max(maximum, float64(value)/temperature)
	}
	result := make([]float64, len(values))
	total := 0.0
	for index, value := range values {
		result[index] = math.Exp(float64(value)/temperature - maximum)
		total += result[index]
	}
	for index := range result {
		result[index] /= total
	}
	return result
}

func entropyConfidence(values []float64) float64 {
	entropy := 0.0
	for _, value := range values {
		if value > 0 {
			entropy -= value * math.Log(value)
		}
	}
	return 1 - entropy/math.Log(float64(len(values)))
}

func argmax(values []float64) int {
	best := 0
	for index := 1; index < len(values); index++ {
		if values[index] > values[best] {
			best = index
		}
	}
	return best
}

func probabilityMap(keys []string, values []float64) map[string]float64 {
	result := make(map[string]float64, len(keys))
	for index, key := range keys {
		result[key] = values[index]
	}
	return result
}
