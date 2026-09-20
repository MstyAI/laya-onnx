#!/usr/bin/env python3
"""Export the pinned Laya checkpoint as a dynamic FP16 ONNX graph."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import laya
import onnx
import torch
from huggingface_hub import snapshot_download

DEFAULT_MODEL = "convaiinnovations/laya"
DEFAULT_REVISION = "c5d78730f3493e4fe16d61507ef4b78eef7318cf"


class DecisionGraph(torch.nn.Module):
    """Inference-only Laya graph used by the public runtime."""

    def __init__(self, model: torch.nn.Module) -> None:
        super().__init__()
        self.model = model

    def forward(
        self,
        input_ids: torch.Tensor,
        attention_mask: torch.Tensor,
        marker_pos: torch.Tensor,
        marker_mask: torch.Tensor,
        qtype: torch.Tensor,
    ) -> torch.Tensor:
        hidden = self.model.encoder(
            input_ids=input_ids, attention_mask=attention_mask
        ).last_hidden_state
        hidden = hidden + self.model.type_emb(qtype)[:, None, :]
        if self.model.head is not None:
            padding = ~attention_mask.bool()
            for layer in self.model.head.layers:
                hidden = layer(hidden, src_key_padding_mask=padding)
        indices = marker_pos.clamp(min=0)[:, :, None].expand(-1, -1, hidden.size(-1))
        markers = torch.gather(hidden, 1, indices)
        logits = self.model.scorer(markers).squeeze(-1).float()
        return logits.masked_fill(~marker_mask, -1e4)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--revision", default=DEFAULT_REVISION)
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args()


def export_graph(agent: laya.Agent, destination: Path) -> None:
    agent.model.half()
    graph = DecisionGraph(agent.model).eval()
    batch, sequence, options = 2, 32, 4
    inputs = (
        torch.zeros((batch, sequence), dtype=torch.int64),
        torch.ones((batch, sequence), dtype=torch.int64),
        torch.tensor([[3, 7, 11, 15], [3, 7, 11, 15]], dtype=torch.int64),
        torch.ones((batch, options), dtype=torch.bool),
        torch.tensor([0, 1], dtype=torch.int64),
    )
    dynamic_batch = torch.export.Dim("batch", min=1, max=64)
    dynamic_sequence = torch.export.Dim("sequence", min=8, max=512)
    dynamic_options = torch.export.Dim("options", min=2, max=255)
    torch.onnx.export(
        graph,
        inputs,
        destination,
        input_names=[
            "input_ids",
            "attention_mask",
            "marker_pos",
            "marker_mask",
            "qtype",
        ],
        output_names=["logits"],
        dynamic_shapes=(
            {0: dynamic_batch, 1: dynamic_sequence},
            {0: dynamic_batch, 1: dynamic_sequence},
            {0: dynamic_batch, 1: dynamic_options},
            {0: dynamic_batch, 1: dynamic_options},
            {0: dynamic_batch},
        ),
        dynamo=True,
        external_data=True,
        opset_version=18,
    )
    onnx.checker.check_model(str(destination))


def copy_metadata(checkpoint: Path, output: Path) -> None:
    sources = {
        "tokenizer.json": checkpoint / "tokenizer" / "tokenizer.json",
        "rl_agent_config.json": checkpoint / "rl_agent_config.json",
    }
    for name, source in sources.items():
        (output / name).write_bytes(source.read_bytes())


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def write_manifest(output: Path, model: str, revision: str) -> None:
    files = []
    for path in sorted(output.iterdir()):
        if path.is_file() and path.name != "manifest.json":
            files.append(
                {
                    "path": path.name,
                    "bytes": path.stat().st_size,
                    "sha256": sha256(path),
                }
            )
    manifest = {
        "schemaVersion": 1,
        "model": model,
        "revision": revision,
        "precision": "fp16",
        "files": files,
    }
    (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")


def main() -> None:
    args = parse_args()
    if args.output.exists() and any(args.output.iterdir()):
        raise SystemExit(f"refusing to overwrite non-empty directory: {args.output}")
    args.output.mkdir(parents=True, exist_ok=True)
    checkpoint = Path(snapshot_download(args.model, revision=args.revision))
    agent = laya.load(str(checkpoint), device="cpu")
    export_graph(agent, args.output / "model.onnx")
    copy_metadata(checkpoint, args.output)
    write_manifest(args.output, args.model, args.revision)


if __name__ == "__main__":
    main()
