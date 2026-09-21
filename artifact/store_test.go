package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestStoreInstallsAndReusesVerifiedFiles(t *testing.T) {
	originalFiles, originalRuntimes := modelFiles, runtimes
	t.Cleanup(func() { modelFiles, runtimes = originalFiles, originalRuntimes })

	model := []byte("model")
	library := []byte("runtime")
	archive := tarGzip(t, "package/lib/runtime.so", library)
	modelFiles = []fileSpec{{path: "model.onnx", sha256: checksum(model), bytes: int64(len(model))}}
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		switch request.URL.Path {
		case "/model/model.onnx":
			_, _ = w.Write(model)
		case "/runtime":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	runtimes = map[string]runtimeSpec{"test/arch": {
		url: server.URL + "/runtime", sha256: checksum(archive), bytes: int64(len(archive)),
		member:      "package/lib/runtime.so",
		libraryName: "runtime.so", librarySHA256: checksum(library), libraryBytes: int64(len(library)),
	}}
	var progress Progress
	store := Store{
		Root: t.TempDir(), ModelBaseURL: server.URL + "/model", Client: server.Client(), Platform: "test/arch",
		OnProgress: func(update Progress) { progress = update },
	}
	paths, err := store.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !store.Installed() {
		t.Fatal("completed install was not detected")
	}
	if raw, err := os.ReadFile(filepath.Join(paths.ModelDir, "model.onnx")); err != nil || string(raw) != string(model) {
		t.Fatalf("model=%q err=%v", raw, err)
	}
	if raw, err := os.ReadFile(paths.RuntimeLibrary); err != nil || string(raw) != string(library) {
		t.Fatalf("runtime=%q err=%v", raw, err)
	}
	if progress.DownloadedBytes != int64(len(model)+len(archive)) || progress.TotalBytes != progress.DownloadedBytes {
		t.Fatalf("progress = %+v", progress)
	}
	before := requests.Load()
	if _, err := store.Ensure(context.Background()); err != nil || requests.Load() != before {
		t.Fatalf("cached ensure err=%v requests=%d->%d", err, before, requests.Load())
	}
	if err := os.WriteFile(filepath.Join(paths.ModelDir, "model.onnx"), []byte("modzl"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondStore := Store{Root: store.Root, ModelBaseURL: store.ModelBaseURL, Client: store.Client, Platform: store.Platform}
	if _, err := secondStore.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() == before {
		t.Fatal("corrupted cache was reused")
	}
}

func TestResolveNeverDownloadsMissingArtifacts(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	store := Store{Root: t.TempDir(), ModelBaseURL: server.URL, Client: server.Client(), Platform: "darwin/arm64"}
	if _, err := store.Resolve(context.Background()); !errors.Is(err, ErrNotPrepared) {
		t.Fatalf("resolve error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("resolve made %d requests", requests.Load())
	}
}

func TestStoreRejectsUnsafeModelURL(t *testing.T) {
	store := Store{Root: t.TempDir(), ModelBaseURL: "http://example.com/model", Platform: "darwin/arm64"}
	if _, err := store.Ensure(context.Background()); err == nil {
		t.Fatal("insecure model URL was accepted")
	}
}

func tarGzip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	compressed := gzip.NewWriter(&output)
	archive := tar.NewWriter(compressed)
	if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func checksum(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
