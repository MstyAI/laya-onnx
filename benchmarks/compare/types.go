package main

import "time"

type benchmarkCase struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	Expected labels `json:"expected"`
}

type labels struct {
	Department string `json:"department"`
	Refund     bool   `json:"refund"`
	Urgency    int    `json:"urgency"`
}

type prediction struct {
	Labels labels
	Model  string
	Usage  usage
}

type usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

type modelMetrics struct {
	Model                  string  `json:"model"`
	CorrectQuestions       int     `json:"correctQuestions"`
	TotalQuestions         int     `json:"totalQuestions"`
	QuestionAccuracy       float64 `json:"questionAccuracy"`
	DepartmentCorrect      int     `json:"departmentCorrect"`
	DepartmentAccuracy     float64 `json:"departmentAccuracy"`
	RefundCorrect          int     `json:"refundCorrect"`
	RefundAccuracy         float64 `json:"refundAccuracy"`
	RefundPositiveRecall   float64 `json:"refundPositiveRecall"`
	RefundNegativeRecall   float64 `json:"refundNegativeRecall"`
	RefundBalancedAccuracy float64 `json:"refundBalancedAccuracy"`
	UrgencyCorrect         int     `json:"urgencyCorrect"`
	UrgencyAccuracy        float64 `json:"urgencyAccuracy"`
	ExactCases             int     `json:"exactCases"`
	TotalCases             int     `json:"totalCases"`
	ExactCaseAccuracy      float64 `json:"exactCaseAccuracy"`
	StableRepeats          int     `json:"stableRepeats"`
	TotalRepeats           int     `json:"totalRepeats"`
	RepeatStability        float64 `json:"repeatStability"`
	LatencyMeanMS          float64 `json:"latencyMeanMs"`
	LatencyP50MS           float64 `json:"latencyP50Ms"`
	LatencyP95MS           float64 `json:"latencyP95Ms"`
	InputTokens            int64   `json:"inputTokens,omitempty"`
	OutputTokens           int64   `json:"outputTokens,omitempty"`
	ResidentMemoryMiB      float64 `json:"residentMemoryMiB,omitempty"`
	ModelLoadMS            float64 `json:"modelLoadMs,omitempty"`
	WarmupMS               float64 `json:"warmupMs,omitempty"`
}

type report struct {
	SchemaVersion          int          `json:"schemaVersion"`
	GeneratedAt            time.Time    `json:"generatedAt"`
	Machine                string       `json:"machine"`
	Platform               string       `json:"platform"`
	Cases                  int          `json:"cases"`
	QuestionsPerCase       int          `json:"questionsPerCase"`
	MeasuredRuns           int          `json:"measuredRuns"`
	JevEndpoint            string       `json:"jevEndpoint"`
	LocalExecutionProvider string       `json:"localExecutionProvider"`
	Local                  modelMetrics `json:"local"`
	Jev                    modelMetrics `json:"jev"`
	Agreement              float64      `json:"agreement"`
	AgreedQuestions        int          `json:"agreedQuestions"`
	TotalComparisons       int          `json:"totalComparisons"`
	DepartmentAgreement    float64      `json:"departmentAgreement"`
	RefundAgreement        float64      `json:"refundAgreement"`
	UrgencyAgreement       float64      `json:"urgencyAgreement"`
	CaseResults            []caseResult `json:"caseResults"`
	LocalLatencySamplesMS  []float64    `json:"localLatencySamplesMs"`
	JevLatencySamplesMS    []float64    `json:"jevLatencySamplesMs"`
}

type caseResult struct {
	ID       string `json:"id"`
	Expected labels `json:"expected"`
	Local    labels `json:"local"`
	Jev      labels `json:"jev"`
}

type agreementMetrics struct {
	all        int
	department int
	refund     int
	urgency    int
	total      int
}

type sample struct {
	caseIndex  int
	prediction prediction
	latency    time.Duration
}
