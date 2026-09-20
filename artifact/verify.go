package artifact

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type marker struct {
	SchemaVersion      int    `json:"schemaVersion"`
	ModelRevision      string `json:"modelRevision"`
	ONNXRuntimeVersion string `json:"onnxRuntimeVersion"`
	Platform           string `json:"platform"`
}

func complete(root string, runtimeInfo runtimeSpec) bool {
	for _, spec := range modelFiles {
		if verify(filepath.Join(root, spec.path), spec.bytes, spec.sha256) != nil {
			return false
		}
	}
	return verify(filepath.Join(root, "onnxruntime", runtimeInfo.libraryName), runtimeInfo.libraryBytes, runtimeInfo.librarySHA256) == nil
}

func looksComplete(root string, runtimeInfo runtimeSpec, platform string) bool {
	raw, err := os.ReadFile(filepath.Join(root, ".complete.json"))
	if err != nil {
		return false
	}
	var value marker
	if json.Unmarshal(raw, &value) != nil || value != (marker{1, ModelRevision, ONNXRuntimeVersion, platform}) {
		return false
	}
	for _, spec := range modelFiles {
		if !hasSize(filepath.Join(root, spec.path), spec.bytes) {
			return false
		}
	}
	return hasSize(filepath.Join(root, "onnxruntime", runtimeInfo.libraryName), runtimeInfo.libraryBytes)
}

func hasSize(path string, size int64) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() == size
}
