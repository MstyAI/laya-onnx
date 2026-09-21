package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	layaonnx "github.com/MstyAI/laya-onnx"
	"github.com/MstyAI/laya-onnx/artifact"
)

var version = "dev"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "laya-onnx:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: laya-onnx <prepare|status|eval|version>")
	}
	switch args[0] {
	case "prepare":
		return prepare(ctx, args[1:], stdout, stderr)
	case "status":
		return status(stdout)
	case "setup":
		return prepare(ctx, []string{"--json"}, stdout, stderr)
	case "eval":
		return evaluate(ctx, args[1:], stdin, stdout)
	case "version":
		_, err := fmt.Fprintln(stdout, reportedVersion())
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func reportedVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return version
	}
	return info.Main.Version
}

func prepare(ctx context.Context, args []string, output, progressOutput io.Writer) error {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	jsonOutput := flags.Bool("json", false, "print prepared paths as JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	store, err := artifact.DefaultStore()
	if err != nil {
		return err
	}
	progressShown := false
	store.OnProgress = func(progress artifact.Progress) {
		progressShown = true
		percentage := 0.0
		if progress.TotalBytes > 0 {
			percentage = float64(progress.DownloadedBytes) / float64(progress.TotalBytes) * 100
		}
		_, _ = fmt.Fprintf(progressOutput, "\rPreparing Laya… %5.1f%%", percentage)
	}
	paths, err := store.Ensure(ctx)
	if err != nil {
		return err
	}
	if progressShown {
		_, _ = fmt.Fprintln(progressOutput)
	}
	if *jsonOutput {
		return json.NewEncoder(output).Encode(paths)
	}
	_, err = fmt.Fprintln(output, "Laya is ready.")
	return err
}

func status(output io.Writer) error {
	store, err := artifact.DefaultStore()
	if err != nil {
		return err
	}
	if store.Installed() {
		_, err = fmt.Fprintln(output, "ready")
		return err
	}
	size, err := store.DownloadSize()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "not prepared (%.0f MiB download)\n", float64(size)/(1024*1024))
	return err
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
	if (*modelDir == "") != (*runtimeLibrary == "") {
		return errors.New("--model-dir and --runtime must be provided together")
	}
	if *modelDir == "" {
		store, err := artifact.DefaultStore()
		if err != nil {
			return err
		}
		paths, err := store.Resolve(ctx)
		if errors.Is(err, artifact.ErrNotPrepared) {
			size, sizeErr := store.DownloadSize()
			if sizeErr != nil {
				return sizeErr
			}
			return fmt.Errorf("laya is not prepared; run %q first (%.0f MiB download)", "laya-onnx prepare", float64(size)/(1024*1024))
		}
		if err != nil {
			return err
		}
		*modelDir, *runtimeLibrary = paths.ModelDir, paths.RuntimeLibrary
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
