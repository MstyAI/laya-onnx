package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionAndUnknownCommand(t *testing.T) {
	var output bytes.Buffer
	if err := run(context.Background(), []string{"version"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != version {
		t.Fatalf("version output = %q", output.String())
	}
	if err := run(context.Background(), []string{"unknown"}, strings.NewReader(""), &output); err == nil {
		t.Fatal("unknown command was accepted")
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
