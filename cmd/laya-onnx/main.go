package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	layaonnx "github.com/MstyAI/laya-onnx"
	"github.com/MstyAI/laya-onnx/artifact"
)

var version = "dev"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "laya-onnx:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: laya-onnx <setup|eval|version>")
	}
	switch args[0] {
	case "setup":
		return setup(ctx, stdout)
	case "eval":
		return evaluate(ctx, args[1:], stdin, stdout)
	case "version":
		_, err := fmt.Fprintln(stdout, version)
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func setup(ctx context.Context, output io.Writer) error {
	store, err := artifact.DefaultStore()
	if err != nil {
		return err
	}
	paths, err := store.Ensure(ctx)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(paths)
}

func evaluate(ctx context.Context, args []string, stdin io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("eval", flag.ContinueOnError)
	requestPath := flags.String("request", "-", "JSON request file, or - for stdin")
	modelDir := flags.String("model-dir", "", "prepared model directory")
	runtimeLibrary := flags.String("runtime", "", "ONNX Runtime shared library")
	cpu := flags.Bool("cpu", false, "disable platform acceleration")
	if err := flags.Parse(args); err != nil {
		return err
	}
	reader, closeReader, err := requestReader(*requestPath, stdin)
	if err != nil {
		return err
	}
	defer closeReader()
	request, err := layaonnx.DecodeRequest(reader)
	if err != nil {
		return err
	}
	if *modelDir == "" || *runtimeLibrary == "" {
		store, err := artifact.DefaultStore()
		if err != nil {
			return err
		}
		paths, err := store.Ensure(ctx)
		if err != nil {
			return err
		}
		*modelDir, *runtimeLibrary = paths.ModelDir, paths.RuntimeLibrary
	}
	providers := layaonnx.DefaultExecutionProviders()
	if *cpu {
		providers = nil
	}
	engine, err := layaonnx.Open(layaonnx.Options{
		ModelDir: *modelDir, RuntimeLibrary: *runtimeLibrary, ExecutionProviders: providers,
	})
	if err != nil {
		return err
	}
	defer engine.Close()
	result, err := engine.Evaluate(ctx, request)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func requestReader(path string, stdin io.Reader) (io.Reader, func(), error) {
	if path == "-" {
		return stdin, func() {}, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	return file, func() { _ = file.Close() }, nil
}
