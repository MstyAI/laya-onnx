# laya-onnx

`laya-onnx` runs [Laya](https://github.com/NandhaKishorM/laya) typed decision models through ONNX Runtime without Python or PyTorch at inference time.

The first release supports macOS and Linux on ARM64 and x86-64. Linux uses the official glibc ONNX Runtime build; Alpine and other musl-based systems are not supported yet. Windows is not part of the first release.

It accepts one state and several named questions. Each question is evaluated independently in one batched model call:

- **Choice** selects from named options and returns a probability for each.
- **Score** evaluates ordered levels and returns their distribution and expected score.
- **Noul** returns the probability that a statement is true.

This project is an independent runtime. Laya is not Jev, and calibrated probabilities do not guarantee that an individual answer is correct.

## Install the CLI

```sh
go install github.com/MstyAI/laya-onnx/cmd/laya-onnx@latest
```

Evaluate a JSON request from stdin:

```sh
laya-onnx eval <<'JSON'
{
  "state": "I was charged twice. Please refund the duplicate.",
  "questions": {
    "department": {
      "type": "choice",
      "instructions": "Which team should handle this?",
      "criteria": {
        "billing": "Payments and refunds",
        "sales": "Purchases",
        "technical": "Bugs and outages"
      }
    },
    "refund": {
      "type": "noul",
      "instructions": "Does the customer ask for money back?"
    }
  }
}
JSON
```

The first run downloads the pinned model and ONNX Runtime release. Later runs use the local cache. The model artifact is about 810 MiB because the upstream checkpoint contains 421 million FP16 parameters.

Run `laya-onnx setup` to prepare the cache separately. Use `laya-onnx eval --request request.json` to read a file.
The repository also includes a complete request at [`examples/request.json`](examples/request.json).

## Go library

```go
import (
    "context"
    "log"

    layaonnx "github.com/MstyAI/laya-onnx"
    "github.com/MstyAI/laya-onnx/artifact"
)

func main() {
	ctx := context.Background()
	store, err := artifact.DefaultStore()
	if err != nil {
		log.Fatal(err)
	}
	paths, err := store.Ensure(ctx)
	if err != nil {
		log.Fatal(err)
	}

	engine, err := layaonnx.Open(layaonnx.Options{
		ModelDir:          paths.ModelDir,
		RuntimeLibrary:    paths.RuntimeLibrary,
		ExecutionProviders: layaonnx.DefaultExecutionProviders(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	department, err := layaonnx.NewChoice(
		"Which team should handle this?",
		layaonnx.Option{Name: "billing", Description: "Payments and refunds"},
		layaonnx.Option{Name: "technical", Description: "Bugs and outages"},
	)
	if err != nil {
		log.Fatal(err)
	}
	refund, err := layaonnx.NewNoul("Does the customer ask for money back?")
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine.Evaluate(ctx, layaonnx.Request{
		State: "I was charged twice. Please refund the duplicate.",
		Questions: map[string]layaonnx.Question{
			"department": department,
			"refund":     refund,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(result.Answers)
}
```

Applications that already manage model files can omit the `artifact` package and pass their own model directory and ONNX Runtime library to `Open`.

The library opens no sockets. Products decide whether to embed it, invoke the CLI over stdin/stdout, or place their own API in front of it.

## Model artifact

The release workflow:

1. Downloads the immutable upstream Laya checkpoint.
2. Exports a dynamic FP16 ONNX graph.
3. Excludes Laya's experimental action/escalation head, which is not used for typed answers.
4. Compares ONNX results with upstream Laya across Choice, Score, Noul, high-cardinality, and full-context cases.
5. Publishes the graph, weights, tokenizer, configuration, validation report, and checksums as immutable release assets.

The accepted export does not retrain or quantize the model. Selected answers matched upstream in the checked cases; the largest probability difference was below 0.001. The checked results are committed in [`evidence/model-c5d78730.json`](evidence/model-c5d78730.json). A smaller INT8 experiment was rejected after it changed a full-context decision.

Reproduce an export:

```sh
python -m venv .venv
. .venv/bin/activate
pip install -r scripts/requirements-release.txt
python scripts/export_model.py --output build/model
python scripts/validate_model.py \
  --onnx build/model/model.onnx \
  --output build/model/validation.json
```

## Limits

- The English checkpoint has a 512-token budget per question, including instructions, options, and state.
- Text beyond that budget is truncated from the end, matching upstream Laya.
- Arithmetic, counting, date comparison, and multi-hop lookup belong in deterministic code.
- Confidence thresholds must be evaluated on the application's own data.
- Core ML acceleration is used on Apple Silicon when supported by the graph; ONNX Runtime falls back to CPU for unsupported operations.

## Security

Model and runtime downloads use pinned versions, expected byte counts, and SHA-256 checksums. Downloads require HTTPS and install through a staging directory before activation.

Report vulnerabilities according to [SECURITY.md](SECURITY.md).

## License

The runtime is Apache-2.0. Laya code and model weights retain their upstream Apache-2.0 license and attribution. See [NOTICE](NOTICE).
