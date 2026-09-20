package layaonnx

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"
)

const (
	// ModelName is returned in every Result.
	ModelName = "laya-onnx"

	maxBatchQuestions = 64
)

// Options identifies the model files, runtime library, and inference settings.
type Options struct {
	// ModelDir contains model.onnx, its external data, tokenizer, and config.
	ModelDir string
	// RuntimeLibrary is the ONNX Runtime shared library path.
	RuntimeLibrary string
	// Threads controls ONNX Runtime intra-operation parallelism. Zero chooses a
	// platform default.
	Threads int
	// ExecutionProviders lists providers in ONNX Runtime priority order.
	ExecutionProviders []string
}

// Engine owns one loaded Laya model and can evaluate repeated requests.
type Engine struct {
	tokenizer modelTokenizer
	backend   inferenceBackend
	config    modelConfig
	tokens    specialTokens
}

type specialTokens struct {
	classification int64
	separator      int64
	padding        int64
	mask           int64
}

type modelTokenizer interface {
	Encode(string) []int
	With(api.EncodeOptions) error
	TokenToID(string) (int, bool)
}

// Open loads the tokenizer, model configuration, and ONNX graph.
func Open(options Options) (*Engine, error) {
	if options.ModelDir == "" || options.RuntimeLibrary == "" {
		return nil, errors.New("model directory and ONNX Runtime library are required")
	}
	tokenizer, err := hftokenizer.NewFromFile(nil, filepath.Join(options.ModelDir, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}
	if err := tokenizer.With(api.EncodeOptions{AddSpecialTokens: false}); err != nil {
		return nil, fmt.Errorf("configure tokenizer: %w", err)
	}
	config, err := readModelConfig(filepath.Join(options.ModelDir, "rl_agent_config.json"))
	if err != nil {
		return nil, err
	}
	tokens, err := readSpecialTokens(tokenizer)
	if err != nil {
		return nil, err
	}
	backend, err := openONNXBackend(options)
	if err != nil {
		return nil, err
	}
	return &Engine{tokenizer: tokenizer, backend: backend, config: config, tokens: tokens}, nil
}

func readSpecialTokens(tokenizer modelTokenizer) (specialTokens, error) {
	var tokens specialTokens
	for token, target := range map[string]*int64{
		"[CLS]":  &tokens.classification,
		"[SEP]":  &tokens.separator,
		"[PAD]":  &tokens.padding,
		"[MASK]": &tokens.mask,
	} {
		id, found := tokenizer.TokenToID(token)
		if !found {
			return specialTokens{}, fmt.Errorf("tokenizer is missing %s", token)
		}
		*target = int64(id)
	}
	return tokens, nil
}

// Close releases the model and ONNX Runtime resources. It is safe to call more
// than once.
func (e *Engine) Close() error {
	if e == nil || e.backend == nil {
		return nil
	}
	err := e.backend.Close()
	e.backend = nil
	return err
}

// Evaluate answers all questions in one model batch.
func (e *Engine) Evaluate(ctx context.Context, request Request) (Result, error) {
	if e == nil || e.backend == nil {
		return Result{}, errors.New("engine is closed")
	}
	state, err := serializeState(request.State)
	if err != nil {
		return Result{}, err
	}
	ids := sortedQuestionIDs(request.Questions)
	if len(ids) == 0 {
		return Result{}, fmt.Errorf("%w: at least one question is required", ErrInvalidRequest)
	}
	if len(ids) > maxBatchQuestions {
		return Result{}, fmt.Errorf("%w: at most %d questions are allowed", ErrInvalidRequest, maxBatchQuestions)
	}
	rows := make([]preparedQuestion, 0, len(ids))
	var inputTokens int64
	for _, id := range ids {
		if id == "" {
			return Result{}, fmt.Errorf("%w: question id is required", ErrInvalidRequest)
		}
		row, err := e.prepareQuestion(id, state, request.Questions[id])
		if err != nil {
			return Result{}, fmt.Errorf("%w: question %q: %v", ErrInvalidRequest, id, err)
		}
		inputTokens += int64(len(row.inputIDs))
		rows = append(rows, row)
	}
	batch := e.makeBatch(rows)
	logits, shape, err := e.backend.Infer(ctx, batch)
	if err != nil {
		return Result{}, fmt.Errorf("run inference: %w", err)
	}
	if err := validateOutput(batch, logits, shape); err != nil {
		return Result{}, err
	}
	answers := make(map[string]Answer, len(rows))
	for index, row := range rows {
		values := logits[index*batch.options : index*batch.options+len(row.options)]
		answers[row.id] = e.answer(row, values)
	}
	return Result{Model: ModelName, Answers: answers, Usage: Usage{InputTokens: inputTokens}}, nil
}

func sortedQuestionIDs(questions map[string]Question) []string {
	ids := make([]string, 0, len(questions))
	for id := range questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
