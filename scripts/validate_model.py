#!/usr/bin/env python3
"""Compare an exported graph with the pinned upstream Laya runtime."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import laya
import numpy as np
import onnxruntime as ort
from export_model import DEFAULT_MODEL, DEFAULT_REVISION
from huggingface_hub import snapshot_download
from laya.common import (
    QTYPES,
    build_sequence,
    collate_items,
    render_options,
    temp_bucket,
)

CASES = [
    (
        "I was charged twice and need the duplicate refunded today.",
        {
            "department": {
                "type": "choice",
                "instructions": "Which team should handle this?",
                "criteria": {
                    "billing": "payments and refunds",
                    "sales": "purchases",
                    "technical": "bugs",
                },
            },
            "refund": {
                "type": "noul",
                "instructions": "Does the customer ask for money back?",
            },
            "urgency": {
                "type": "score",
                "instructions": "How urgent is this?",
                "criteria": ["not urgent", "soon", "urgent", "critical"],
            },
        },
    ),
    (
        {"request": "Refactor this service and investigate its deadlock."},
        {
            "difficulty": {
                "type": "choice",
                "instructions": "How difficult is the requested work?",
                "criteria": {
                    f"level_{index}": f"difficulty level {index}" for index in range(12)
                },
            }
        },
    ),
    (
        "The release notes contain routine fixes. " * 180,
        {
            "relevant": {
                "type": "noul",
                "instructions": "Is this about a payment failure?",
            }
        },
    ),
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--revision", default=DEFAULT_REVISION)
    parser.add_argument("--onnx", type=Path, required=True)
    parser.add_argument("--output", type=Path)
    return parser.parse_args()


def softmax(values: np.ndarray, temperature: float) -> np.ndarray:
    values = values / temperature
    values = np.exp(values - values.max())
    return values / values.sum()


def prepare(agent: laya.Agent, state: object, questions: dict) -> tuple:
    ids = list(questions)
    items = []
    internal = []
    for question_id in ids:
        question = to_internal(questions[question_id])
        sequence, markers = build_sequence(
            agent.tok,
            state,
            question,
            agent.cfg["max_len"],
            agent.cfg["head_max_len"],
        )
        items.append(
            {"ids": sequence, "markers": markers, "qtype": QTYPES[question["t"]]}
        )
        internal.append(question)
    return ids, internal, collate_items([items], agent.tok.pad_token_id)


def to_internal(question: dict) -> dict:
    question_type = question["type"]
    criteria = question.get("criteria")
    if question_type == "choice" and isinstance(criteria, list):
        criteria = {value: None for value in criteria}
    instructions = question["instructions"]
    if not isinstance(instructions, str):
        instructions = json.dumps(instructions)
    return {"t": question_type, "ins": instructions, "crit": criteria}


def compare_case(
    session: ort.InferenceSession,
    agent: laya.Agent,
    case_index: int,
    state: object,
    questions: dict,
) -> list[dict]:
    reference = agent.predict(state, questions)
    ids, internal, batch = prepare(agent, state, questions)
    logits = session.run(
        ["logits"],
        {
            "input_ids": batch["input_ids"].numpy(),
            "attention_mask": batch["attention_mask"].numpy(),
            "marker_pos": batch["marker_pos"].numpy(),
            "marker_mask": batch["marker_mask"].numpy(),
            "qtype": batch["qtype"].numpy(),
        },
    )[0]
    comparisons = []
    for row, question_id in enumerate(ids):
        comparisons.append(
            compare_question(
                agent,
                case_index,
                question_id,
                internal[row],
                batch,
                row,
                logits[row],
                reference["answers"][question_id],
            )
        )
    return comparisons


def compare_question(
    agent: laya.Agent,
    case_index: int,
    question_id: str,
    question: dict,
    batch: dict,
    row: int,
    logits: np.ndarray,
    answer: dict,
) -> dict:
    count = len(render_options(question))
    question_type = QTYPES[question["t"]]
    temperature = agent.temperature_by_options.get(
        temp_bucket(question_type, count), agent.temperature[question_type]
    )
    probabilities = softmax(logits[:count], temperature)
    if question["t"] == "choice":
        keys = list(question["crit"])
        expected = keys.index(answer["choice"])
        reference_probabilities = list(answer["probabilities"].values())
    elif question["t"] == "score":
        expected = max(
            range(count), key=lambda index: answer["probabilities"][str(index)]
        )
        reference_probabilities = list(answer["probabilities"].values())
    else:
        expected = int(answer["noul"] >= 0.5)
        reference_probabilities = [1 - answer["noul"], answer["noul"]]
    actual = int(probabilities.argmax())
    result = {
        "case": case_index,
        "question": question_id,
        "options": count,
        "tokens": int(batch["attention_mask"][row].sum()),
        "expected": expected,
        "actual": actual,
        "maxProbabilityError": float(
            max(
                abs(probabilities[index] - reference_probabilities[index])
                for index in range(count)
            )
        ),
    }
    if actual != expected:
        raise RuntimeError(f"ONNX parity failed: {result}")
    return result


def main() -> None:
    args = parse_args()
    checkpoint = snapshot_download(args.model, revision=args.revision)
    agent = laya.load(checkpoint, device="cpu")
    session = ort.InferenceSession(str(args.onnx), providers=["CPUExecutionProvider"])
    comparisons = []
    for index, (state, questions) in enumerate(CASES):
        comparisons.extend(compare_case(session, agent, index, state, questions))
    report = {
        "model": args.model,
        "revision": args.revision,
        "onnx": args.onnx.name,
        "comparisons": comparisons,
    }
    rendered = json.dumps(report, indent=2) + "\n"
    if args.output:
        args.output.write_text(rendered)
    print(rendered, end="")


if __name__ == "__main__":
    main()
