# Jev comparison benchmark

This benchmark compares the local Laya ONNX engine with TypeSafe Jev on the same request bodies.

- 50 labeled customer-support states
- 3 questions per state: department Choice, refund Noul, urgency Score
- 1 unmeasured warm-up request per model
- 5 measured sequential passes, or 250 requests and 750 decisions per model
- one resident local engine and one reused HTTP client
- Noul threshold: `0.5`
- Score label: highest-probability level
- accuracy: first measured pass over the 50 unique cases
- stability: later predictions compared with each case's first prediction
- latency: all 250 measured requests

The case set is synthetic and intentionally small enough to audit. It measures this support workflow, not general reasoning quality. It does not measure parallel throughput, provider price, or changing network conditions.

Run `laya-onnx prepare` first, then:

```sh
export TYPESAFE_API_KEY=...
go run ./benchmarks/compare \
  -runs 5 \
  -machine "your machine description" \
  -output benchmark.json
```

Use `-local-cpu` to disable Core ML for a separate local diagnostic. Never commit API keys or place them in command-line arguments.
