package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func download(ctx context.Context, client *http.Client, source string, spec fileSpec, destination string, progress func(int64)) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", spec.path, err)
	}
	defer response.Body.Close()
	if response.Request == nil || response.Request.URL == nil || response.Request.URL.Scheme != "https" || response.Request.URL.User != nil {
		return fmt.Errorf("download %s redirected to an unsafe destination", spec.path)
	}
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("download %s: HTTP %d", spec.path, response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength != spec.bytes {
		return fmt.Errorf("download %s: unexpected size", spec.path)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create downloaded file: %w", err)
	}
	hash := sha256.New()
	progressOutput := &progressWriter{report: progress}
	written, copyErr := io.Copy(io.MultiWriter(file, hash, progressOutput), io.LimitReader(response.Body, spec.bytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("write %s: %w", spec.path, errors.Join(copyErr, closeErr))
	}
	if written != spec.bytes || hex.EncodeToString(hash.Sum(nil)) != spec.sha256 {
		return fmt.Errorf("verify %s: checksum or size mismatch", spec.path)
	}
	return nil
}

type progressWriter struct {
	written int64
	report  func(int64)
}

func (w *progressWriter) Write(value []byte) (int, error) {
	w.written += int64(len(value))
	if w.report != nil {
		w.report(w.written)
	}
	return len(value), nil
}

func verify(path string, size int64, checksum string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(file, size+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return errors.Join(copyErr, closeErr)
	}
	if written != size || hex.EncodeToString(hash.Sum(nil)) != checksum {
		return errors.New("checksum or size mismatch")
	}
	return nil
}

func installRuntime(ctx context.Context, client *http.Client, root string, spec runtimeSpec, progress func(int64)) error {
	archive := filepath.Join(root, ".onnxruntime-download")
	if err := download(ctx, client, spec.url, fileSpec{"onnxruntime", spec.sha256, spec.bytes}, archive, progress); err != nil {
		return err
	}
	destination := filepath.Join(root, "onnxruntime", spec.libraryName)
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return fmt.Errorf("create ONNX Runtime directory: %w", err)
	}
	if err := extract(archive, spec, destination); err != nil {
		return err
	}
	if err := verify(destination, spec.libraryBytes, spec.librarySHA256); err != nil {
		return fmt.Errorf("verify ONNX Runtime: %w", err)
	}
	if err := os.Remove(archive); err != nil {
		return fmt.Errorf("remove runtime archive: %w", err)
	}
	return nil
}
