# Conventional Commit Eval Workflow

This skill's evals are designed to judge commit message generation from a real git surface, not from a narrated scenario.

The fixture copy created by `skill-creator` is only a plain working directory. It is not a git repo and it does not contain staged changes by default. This skill adds a local preparation step so each eval can create the intended staged diff before the evaluated agent is asked for a commit message.

This file is maintainer guidance for building and preparing eval scenarios for this skill.

## What This Skill Owns

- `evals/evals.json` defines the eval metadata.
- Each eval declares:
  - `fixture`
  - `prepare_patch`
- `scripts/prepare.py` reads that metadata from this skill and prepares both the `with_skill` and `without_skill` fixture copies for one eval.

`scripts/prepare.py` takes:

- `--eval-id`
- `--eval-run-dir`

## Operating Procedure

Use this sequence when running these evals with `skill-creator`.

1. Run `skill-creator/scripts/prepare_fixture.py`.
Why: this creates the isolated `eval-N/with_skill` and `eval-N/without_skill` directories and copies the configured fixture into each.

2. For each generated eval directory, run `scripts/prepare.py --eval-id <id> --eval-run-dir <path-to-eval-N>`.
Why: this is the step that turns the copied fixture into a disposable git repo, makes the baseline commit, applies the eval's `prepare_patch`, and stages the resulting changes in both configs.

3. Run `skill-creator/scripts/run_skill_evals.py` against the prepared run root from step 1.
Why: `run_skill_evals.py` expects the prepared run root created by `prepare_fixture.py` and executes the evals against the git state created in step 2.

4. Only inside step 3 should the evaluated agent be asked to generate the commit message from the staged changes.
Why: before preparation there is no git history and no staged diff to inspect, so the core behavior under test does not exist yet.

## Creating A New Scenario Patch

Use this sequence when adding a new scenario to `evals/evals.json`.

1. Add the new eval entry to `evals/evals.json` with its `id`, `eval_name`, `fixture`, and `prepare_patch`.
Why: the scenario metadata must exist before the fixture preparation workflow can materialize the right disposable workspace and before the patch can be linked back into the eval.

2. Run `skill-creator/scripts/prepare_fixture.py` for this skill so you get a fresh prepared run root.
Why: patch authoring must happen against the same disposable fixture copy shape that the eval harness uses, not against the source fixture repo directly.

3. Enter the copied fixture directory for the new eval under the prepared run root.
Why: this directory contains the code you will turn into the scenario, but at this point it is only plain files and is not a git repo yet.

4. Initialize git in that copied fixture directory, add the baseline files, and commit the baseline fixture state.
Why: the scenario patch must represent the delta from a known baseline so `scripts/prepare.py` can recreate that exact change set later.

5. Make the code changes you want the eval to expose.
Why: these edits are the actual scenario under test. The evaluated agent should later infer the commit message from this change set.

6. Make sure git is tracking the full intended scenario by adding all relevant files.
Why: untracked or unstaged files will be missing from the patch, which means the recreated eval surface will not match the scenario you authored.

7. Create the patch file from the tracked scenario delta and save it in this skill's `fixtures/` directory.
Why: `scripts/prepare.py` applies the configured patch file after creating the baseline commit, so the scenario must live as a reusable patch in `fixtures/`.

8. Point the eval's `prepare_patch` field at that new file in `fixtures/`.
Why: this is how the eval links the metadata in `evals/evals.json` to the scenario patch that will be applied during preparation.

## Patch Authoring Result

After this process is complete:

- `evals/evals.json` names the fixture and the patch file for the new scenario
- `fixtures/<scenario>.patch` contains the tracked delta from the committed baseline fixture state
- `scripts/prepare.py` can recreate the scenario by initializing git, committing the baseline, applying the patch, and staging the result

## What To Trust

When preparing one eval:

- trust `fixture` from `evals/evals.json` to identify the repo under each config
- trust `prepare_patch` from `evals/evals.json` to identify the scenario patch
- trust `files[]` only for files that should be copied into the run directory for the evaluated agent to access
