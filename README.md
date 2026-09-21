# laya-onnx

Run [Laya](https://github.com/NandhaKishorM/laya) decision models locally with ONNX Runtime.

`laya-onnx` is a Go library and a small CLI. It needs no Python or PyTorch at runtime, opens no ports, and sends no data off the machine.

Supported platforms:

- macOS on Apple Silicon and Intel
- Linux on ARM64 and x86-64 with glibc

Windows and musl-based Linux distributions such as Alpine are not supported yet.

## Laya or Jev?

Use **[Jev](https://typesafe.ai)** when cloud inference is acceptable and you want the stronger model from our test. Use **Laya ONNX** when requests must stay local or work offline.

| Result | Laya ONNX | Jev 1.13.0 |
| --- | ---: | ---: |
| Question accuracy | 74% | **96%** |
| All three answers correct | 36% | **88%** |
| Warm latency, p50 | 397 ms | **202 ms** |
| Warm latency, p95 | 420 ms | **312 ms** |

<details>
<summary>Benchmark details</summary>

Measured September 20, 2026. We used 50 labeled support cases with three questions each, repeated five times after warm-up. TypeSafe's `jev-latest` alias returned `jev-1.13.0`. The test ran on an M3 Max MacBook Pro. Each model repeated its own answers exactly across all five runs.

This is a small synthetic benchmark, not a general leaderboard. Network latency, hardware, and workloads vary. Read the [method](benchmarks/README.md), [cases](benchmarks/support_cases.json), and [raw results](evidence/jev-vs-laya-2026-09-20.json) before using these numbers for a product decision.

On the same Mac, a shorter CPU-only run reduced Laya's p50 to 359 ms and memory use from 3.67 GiB to 3.14 GiB. See the [CPU report](evidence/jev-vs-laya-cpu-2026-09-20.json).

</details>

## Quick start

Install the CLI:

```sh
go install github.com/MstyAI/laya-onnx/cmd/laya-onnx@latest
```

Download and verify the model:

```sh
laya-onnx prepare
```

Preparation downloads about 819 MiB. It happens only when you ask for it. Check it at any time:

```sh
laya-onnx status
```

Run a decision:

```sh
laya-onnx eval <<'JSON'
{
  "state": "I was charged twice. Please refund the duplicate.",
  "questions": {
    "refund": {
      "type": "noul",
      "instructions": "Does the customer ask for money back?"
    }
  }
}
JSON
```

`eval` never downloads files. If the model is missing, it tells you to run `laya-onnx prepare` and exits.

You can also read a request from a file:

```sh
laya-onnx eval --request examples/request.json
```

## Question types

- **Choice** picks one named option and returns probabilities for every option.
- **Score** evaluates ordered levels and returns their probabilities and expected score.
- **Noul** returns the probability that a statement is true. Think of it as yes/no with a useful confidence value.

All questions in one request share the same state and run in one model batch.

<details>
<summary>Use the Go library</summary>

Applications can prepare and embed the model directly.

```go
package main

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

	// Ensure is the explicit preparation step. It downloads only when needed.
	paths, err := store.Ensure(ctx)
	if err != nil {
		log.Fatal(err)
	}

	engine, err := layaonnx.Open(layaonnx.Options{
		ModelDir:           paths.ModelDir,
		RuntimeLibrary:     paths.RuntimeLibrary,
		ExecutionProviders: layaonnx.DefaultExecutionProviders(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	refund, err := layaonnx.NewNoul("Does the customer ask for money back?")
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine.Evaluate(ctx, layaonnx.Request{
		State:     "Please refund the duplicate charge.",
		Questions: map[string]layaonnx.Question{"refund": refund},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(*result.Answers["refund"].Noul)
}
```

If your application manages its own files, pass the model directory and ONNX Runtime library directly to `layaonnx.Open`.

</details>

<details>
<summary>Model details and limits</summary>

The release uses the English Laya checkpoint at revision `c5d78730`. It is an FP16 ONNX export with 421 million parameters. We did not retrain or quantize the weights.

The release workflow compares the ONNX model with upstream Laya before publishing it. Downloads use pinned sizes and SHA-256 hashes and are installed through a staging directory.

The model has a 512-token budget for each question, including its instructions, options, and state. Extra state text is cut from the end. Use normal code for arithmetic, counting, date comparisons, and other work that should be exact.

Model answers are probabilistic. Test confidence thresholds on your own data before using them in a product.

Export scripts and validation evidence live in [`scripts`](scripts) and [`evidence`](evidence).

</details>

## License

The Go runtime is Apache-2.0. Laya code and model weights keep their upstream Apache-2.0 license and attribution. See [NOTICE](NOTICE).

Please report security issues through the process in [SECURITY.md](SECURITY.md).
