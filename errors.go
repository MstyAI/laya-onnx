package layaonnx

import "errors"

var (
	// ErrInvalidRequest identifies invalid state or question input.
	ErrInvalidRequest = errors.New("invalid decision request")
	// ErrModelContract identifies a model whose inputs or outputs do not match
	// the pinned Laya graph contract.
	ErrModelContract = errors.New("unexpected ONNX model contract")
)
