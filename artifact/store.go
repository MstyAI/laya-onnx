package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// Paths identifies a prepared model directory and native runtime library.
type Paths struct {
	// ModelDir contains the verified graph, weights, tokenizer, and config.
	ModelDir string
	// RuntimeLibrary is the verified ONNX Runtime shared library.
	RuntimeLibrary string
}

// Store manages one verified model and runtime cache.
type Store struct {
	// Root overrides the cache root. Empty uses the user cache directory.
	Root string
	// ModelBaseURL overrides the pinned release asset base URL.
	ModelBaseURL string
	// Client overrides the HTTP client used for downloads.
	Client *http.Client
	// Platform overrides the target in GOOS/GOARCH form.
	Platform string

	mu    sync.Mutex
	ready *Paths
}

// DefaultStore returns a Store rooted in the current user's cache directory.
func DefaultStore() (*Store, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve cache directory: %w", err)
	}
	return &Store{Root: filepath.Join(cache, "laya-onnx"), ModelBaseURL: DefaultModelURL}, nil
}

// Ensure returns verified local artifact paths, downloading them when needed.
// Calls sharing this Store or its cache directory are serialized.
func (s *Store) Ensure(ctx context.Context) (Paths, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready != nil {
		return *s.ready, nil
	}
	if err := s.applyDefaults(); err != nil {
		return Paths{}, err
	}
	runtimeInfo, ok := runtimes[s.Platform]
	if !ok {
		return Paths{}, fmt.Errorf("unsupported platform %s", s.Platform)
	}
	root := s.modelRoot()
	if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
		return Paths{}, fmt.Errorf("create cache directory: %w", err)
	}
	installLock := flock.New(root + ".lock")
	locked, err := installLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return Paths{}, fmt.Errorf("lock model cache: %w", err)
	}
	if !locked {
		if err := ctx.Err(); err != nil {
			return Paths{}, err
		}
		return Paths{}, errors.New("model cache lock was not acquired")
	}
	defer installLock.Unlock()
	if complete(root, runtimeInfo) {
		result := paths(root, runtimeInfo)
		s.ready = &result
		return result, nil
	}
	if err := s.install(ctx, root, runtimeInfo); err != nil {
		return Paths{}, err
	}
	result := paths(root, runtimeInfo)
	s.ready = &result
	return result, nil
}

// Installed checks the completion marker and expected file sizes. Ensure also
// verifies cryptographic hashes before returning a cache for the first time.
func (s *Store) Installed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.applyDefaults(); err != nil {
		return false
	}
	runtimeInfo, ok := runtimes[s.Platform]
	return ok && looksComplete(s.modelRoot(), runtimeInfo, s.Platform)
}

func (s *Store) applyDefaults() error {
	if s.Root == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("resolve cache directory: %w", err)
		}
		s.Root = filepath.Join(cache, "laya-onnx")
	}
	if s.ModelBaseURL == "" {
		s.ModelBaseURL = DefaultModelURL
	}
	if s.Client == nil {
		s.Client = http.DefaultClient
	}
	if s.Platform == "" {
		s.Platform = runtime.GOOS + "/" + runtime.GOARCH
	}
	return nil
}

func (s *Store) modelRoot() string {
	return filepath.Join(s.Root, ModelRevision, strings.ReplaceAll(s.Platform, "/", "-"))
}

func paths(root string, runtimeInfo runtimeSpec) Paths {
	return Paths{ModelDir: root, RuntimeLibrary: filepath.Join(root, "onnxruntime", runtimeInfo.libraryName)}
}

func (s *Store) install(ctx context.Context, root string, runtimeInfo runtimeSpec) error {
	baseURL, err := validateModelURL(s.ModelBaseURL)
	if err != nil {
		return err
	}
	stage := root + ".download"
	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("clear staging directory: %w", err)
	}
	if err := os.MkdirAll(stage, 0o700); err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stage)
		}
	}()
	for _, spec := range modelFiles {
		assetURL := *baseURL
		assetURL.Path = strings.TrimRight(baseURL.Path, "/") + "/" + spec.path
		if err := download(ctx, s.Client, assetURL.String(), spec, filepath.Join(stage, spec.path)); err != nil {
			return err
		}
	}
	if err := installRuntime(ctx, s.Client, stage, runtimeInfo); err != nil {
		return err
	}
	marker := marker{1, ModelRevision, ONNXRuntimeVersion, s.Platform}
	encoded, _ := json.Marshal(marker)
	if err := os.WriteFile(filepath.Join(stage, ".complete.json"), encoded, 0o600); err != nil {
		return fmt.Errorf("write completion marker: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("replace previous install: %w", err)
	}
	if err := os.Rename(stage, root); err != nil {
		return fmt.Errorf("activate install: %w", err)
	}
	committed = true
	return nil
}

func validateModelURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimRight(value, "/"))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("model base URL must be an HTTPS path without credentials, query, or fragment")
	}
	return parsed, nil
}
