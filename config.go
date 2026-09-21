package layaonnx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type modelConfig struct {
	Temperature          []float64          `json:"temperature"`
	TemperatureByOptions map[string]float64 `json:"temperature_by_options"`
}

func readModelConfig(path string) (modelConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return modelConfig{}, fmt.Errorf("read model config: %w", err)
	}
	var config modelConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return modelConfig{}, fmt.Errorf("decode model config: %w", err)
	}
	if err := validateModelConfig(config); err != nil {
		return modelConfig{}, err
	}
	return config, nil
}

func validateModelConfig(config modelConfig) error {
	if len(config.Temperature) != 3 {
		return errors.New("invalid model config: expected three temperatures")
	}
	for _, value := range config.Temperature {
		if value <= 0 {
			return errors.New("invalid model config: temperatures must be positive")
		}
	}
	for _, value := range config.TemperatureByOptions {
		if value <= 0 {
			return errors.New("invalid model config: option temperatures must be positive")
		}
	}
	return nil
}
