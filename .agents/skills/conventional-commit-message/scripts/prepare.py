#!/usr/bin/env python3
"""Prepare one eval run directory for commit-message evaluation.

This script is intentionally skill-local. It reads eval metadata from the
skill's own evals.json, finds the configured fixture and patch for one eval,
and prepares both skill and baseline fixture copies so the evaluated
agent sees a real git surface with staged changes.

Usage:
    python scripts/prepare.py --eval-id 1 --eval-run-dir <path-to-eval-dir>
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


def fail(message: str) -> "None":
    print(f"Error: {message}", file=sys.stderr)
    sys.exit(1)


def run(cmd: list[str], cwd: Path, error_prefix: str) -> str:
    result = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    if result.returncode != 0:
        details = result.stderr.strip() or result.stdout.strip() or f"exit code {result.returncode}"
        fail(f"{error_prefix} in {cwd}:\n{details}")
    return result.stdout.strip()


def load_eval(skill_root: Path, eval_id: str) -> dict:
    evals_path = skill_root / "evals" / "evals.json"
    data = json.loads(evals_path.read_text(encoding="utf-8"))

    for eval_def in data.get("evals", []):
        if str(eval_def.get("id")) == eval_id:
            return eval_def

    fail(f"eval id={eval_id} not found in {evals_path}")


def resolve_skill_relative_file(skill_root: Path, relative_path: str, label: str) -> Path:
    root = skill_root.resolve()
    resolved = (skill_root / relative_path).resolve()

    try:
        resolved.relative_to(root)
    except ValueError:
        fail(f"{label} '{relative_path}' escapes the skill root")

    if not resolved.exists():
        fail(f"{label} '{relative_path}' not found at {resolved}")

    if not resolved.is_file():
        fail(f"{label} '{relative_path}' is not a file at {resolved}")

    return resolved


def resolve_fixture_repo(eval_run_dir: Path, run_type: str, fixture_name: str) -> Path:
    in_workdir = eval_run_dir / run_type / fixture_name
    if in_workdir.is_dir():
        return in_workdir

    external = eval_run_dir / f"{run_type}_fixtures" / fixture_name
    if external.is_dir():
        return external

    fail(
        f"fixture '{fixture_name}' was not found for run type '{run_type}' under "
        f"{eval_run_dir / run_type} or {eval_run_dir / f'{run_type}_fixtures'}"
    )


def ensure_fresh_repo(repo_dir: Path) -> None:
    git_dir = repo_dir / ".git"
    if git_dir.exists():
        fail(
            f"{repo_dir} already contains a .git directory. "
            "Run this script against a fresh prepare_fixture output."
        )


def prepare_repo(repo_dir: Path, patch_path: Path) -> None:
    ensure_fresh_repo(repo_dir)

    run(["git", "init"], repo_dir, "git init failed")
    run(["git", "config", "user.name", "Codex Eval Fixture"], repo_dir, "git config user.name failed")
    run(
        ["git", "config", "user.email", "codex-eval-fixture@example.com"],
        repo_dir,
        "git config user.email failed",
    )
    run(["git", "config", "commit.gpgsign", "false"], repo_dir, "git config commit.gpgsign failed")

    run(["git", "add", "."], repo_dir, "git add baseline failed")
    run(["git", "commit", "-m", "chore: baseline fixture state"], repo_dir, "baseline commit failed")
    run(
        ["git", "apply", "--ignore-space-change", "--ignore-whitespace", str(patch_path)],
        repo_dir,
        "git apply failed",
    )
    run(["git", "add", "-A"], repo_dir, "git add staged changes failed")


def main() -> None:
    parser = argparse.ArgumentParser(description="Prepare one eval run directory for commit-message evals.")
    parser.add_argument("--eval-id", required=True, help="Eval id from evals/evals.json")
    parser.add_argument("--eval-run-dir", required=True, help="Path to the eval-N directory created by prepare_fixture.py")
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    skill_root = script_dir.parent
    eval_run_dir = Path(args.eval_run_dir).expanduser().resolve()

    if not eval_run_dir.exists():
        fail(f"eval run dir does not exist: {eval_run_dir}")

    eval_def = load_eval(skill_root, args.eval_id)
    fixture_name = eval_def.get("fixture")
    if not fixture_name:
        fail(f"eval id={args.eval_id} does not define a fixture")

    patch_relative = eval_def.get("prepare_patch")
    if not patch_relative:
        fail(f"eval id={args.eval_id} does not define prepare_patch")

    patch_path = resolve_skill_relative_file(skill_root, patch_relative, "prepare_patch")

    prepared = {}
    for run_type in ("skill", "baseline"):
        repo_dir = resolve_fixture_repo(eval_run_dir, run_type, fixture_name)
        prepare_repo(repo_dir, patch_path)
        prepared[run_type] = str(repo_dir)

    print(json.dumps({"eval_id": str(args.eval_id), "prepared_repos": prepared}, indent=2))


if __name__ == "__main__":
    main()
