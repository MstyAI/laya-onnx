package artifact

import (
	_ "embed"
	"encoding/json"
)

type fileSpec struct {
	path   string
	sha256 string
	bytes  int64
}

type modelManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Model         string `json:"model"`
	Revision      string `json:"revision"`
	Precision     string `json:"precision"`
	Files         []struct {
		Path   string `json:"path"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

//go:embed model_manifest.json
var modelManifestJSON []byte

var modelFiles = loadModelFiles()

func loadModelFiles() []fileSpec {
	var manifest modelManifest
	if json.Unmarshal(modelManifestJSON, &manifest) != nil || manifest.SchemaVersion != 1 || manifest.Model != UpstreamModel || manifest.Revision != ModelRevision || manifest.Precision != "fp16" || len(manifest.Files) == 0 {
		panic("invalid embedded model manifest")
	}
	files := make([]fileSpec, len(manifest.Files))
	for index, file := range manifest.Files {
		files[index] = fileSpec{path: file.Path, bytes: file.Bytes, sha256: file.SHA256}
	}
	return files
}
