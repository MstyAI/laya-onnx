package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestVersionAndUnknownCommand(t *testing.T) {
	originalVersion := version
	version = "v0.1.0-test"
	t.Cleanup(func() { version = originalVersion })
	var output bytes.Buffer
	if err := run(context.Background(), []string{"version"}, strings.NewReader(""), &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != "v0.1.0-test" {
		t.Fatalf("version output = %q", output.String())
	}
	if err := run(context.Background(), []string{"unknown"}, strings.NewReader(""), &output, io.Discard); err == nil {
		t.Fatal("unknown command was accepted")
	}
}

func TestEvalRequiresBothCustomPaths(t *testing.T) {
	err := run(context.Background(), []string{"eval", "--model-dir", "/tmp/model"}, strings.NewReader(""), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "must be provided together") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestReaderUsesStdin(t *testing.T) {
	input := strings.NewReader("request")
	reader, closeReader, err := requestReader("-", input)
	if err != nil || reader != input {
		t.Fatalf("reader=%v err=%v", reader, err)
	}
	closeReader()
}
