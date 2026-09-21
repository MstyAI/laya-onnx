package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type jevRunner struct {
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
}

func (r *jevRunner) predict(ctx context.Context, state string) (prediction, error) {
	body := map[string]any{
		"model": r.model,
		"state": state,
		"questions": map[string]any{
			"department": map[string]any{
				"type": "choice", "instructions": "Which department should handle this customer request?",
				"criteria": map[string]string{
					"billing":   "Charges, invoices, payments, and refunds",
					"technical": "Bugs, outages, errors, and product malfunctions",
					"sales":     "Pricing, plans, purchasing, and product evaluation",
					"account":   "Sign-in, identity, access, and account settings",
					"shipping":  "Delivery, tracking, damaged packages, and returns in transit",
				},
			},
			"refund": map[string]any{"type": "noul", "instructions": "Does the customer explicitly request a refund or their money back?"},
			"urgency": map[string]any{
				"type": "score", "instructions": "How urgent is the request?",
				"criteria": []string{
					"routine; no time pressure", "soon; should be handled within a few days",
					"urgent; blocking work or needs action today", "critical; severe security, safety, or widespread outage",
				},
			},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return prediction{}, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		result, retry, err := r.request(ctx, encoded)
		if err == nil {
			return result, nil
		}
		if retry == 0 {
			return prediction{}, err
		}
		select {
		case <-ctx.Done():
			return prediction{}, ctx.Err()
		case <-time.After(retry):
		}
	}
	return prediction{}, errors.New("jev retry limit reached")
}

func (r *jevRunner) request(ctx context.Context, encoded []byte) (prediction, time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return prediction{}, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+r.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		return prediction{}, 0, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return prediction{}, 0, err
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		delay := time.Second
		if seconds, parseErr := strconv.Atoi(response.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
			delay = time.Duration(seconds) * time.Second
		}
		return prediction{}, delay, fmt.Errorf("jev HTTP %d", response.StatusCode)
	}
	if response.StatusCode != http.StatusOK {
		return prediction{}, 0, fmt.Errorf("jev HTTP %d: %s", response.StatusCode, raw)
	}
	var result struct {
		Model   string `json:"model"`
		Answers map[string]struct {
			Choice        string             `json:"choice"`
			Noul          *float64           `json:"noul"`
			Probabilities map[string]float64 `json:"probabilities"`
		} `json:"answers"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return prediction{}, 0, err
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
	}, 0, nil
}
