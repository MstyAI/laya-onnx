# Contributing

Issues and pull requests are welcome.

Before opening a pull request:

```sh
make check
```

Changes to model formatting, tokenization, calibration, or ONNX export must include parity evidence against the pinned upstream checkpoint. Do not update model or runtime revisions without updating hashes, validation results, and release notes together.

Keep public APIs small and additive. Describe any observable behavior change in the pull request.
