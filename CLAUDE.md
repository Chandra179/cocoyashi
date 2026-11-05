# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go test ./...                                   # all tests
go test ./tests -run TestChargedOnceAtTripEnd   # one test
go vet ./...
python3 <tools>/check_rules.py ridehailing.rules.yaml
```

`<tools>` is `tools/business-rules/` in the zou repo (find it with `readlink -f ~/.claude/skills/business-rules`, then `../../../tools/business-rules`). It needs PyYAML and jsonschema. Exit 3 is expected while no passing run is recorded as evidence; never use `--no-base` or `--skip-shape` to get past an error.

## How the repo works

The rules come first and the code follows them.

- `ridehailing.rules.yaml` is the source of truth: each rule has a statement, scenarios (at least one required and one forbidden), a `check` pointing at a test file, and who decided it. Rule IDs (REQ-1, CANCEL-2, ...) appear in code comments.
- `ridehailing.markets.yaml` holds the per-market numbers. Rules name a parameter, and the number lives here. `null` means not set yet, and a market with a null fare rate must not offer rides.
- `ride/` is the in-memory core trip loop with no database or HTTP layer. It is a single `Service` with an injected `Clock` and `Gateway`, so time and payments are faked in tests. `Config` is the Go mirror of one market's slice of the markets file.
- `tests/` is one external test package (`package tests`). Each rule's `check.ref` points at a file here, named `<topic>_test.go`. `helpers_test.go` has the fake clock, fake gateway and the `assigned` / `inProgress` setup helpers.

The ride moves through `Requested → Offering → Assigned → Arrived → InProgress → Completed`. `Cancelled` and `NoDriver` end it early. A rider's active ride is dropped from `Service.active` when it ends, which is what lets them request again. Offer timeouts are evaluated lazily whenever the ride is read or acted on (`expire`), so there is no timer goroutine.

## Rules for changing things

- **Never mark a rule `confirmed` yourself, and never weaken a confirmed rule.** Only the owner confirms. What you infer is `proposed` with `source.kind: inferred`. A change to a confirmed rule's statement, scenarios, check or `required` needs the owner's decision and a higher `revision`. The checker errors otherwise.
- The schema rejects extra rule fields (no `phase` or `tags`), so MVP scoping uses `required` only.
- Test first: write the failing test in `tests/`, then the code. A new rule needs a required and a forbidden scenario.
- Money is `int64` in minor units. IDR has no minor unit, so the amounts are whole rupiah.
- Commit directly to `main` and push when asked. The history does this.
- Evidence (`evidence:` in the rules file) is not recorded yet. It needs a committed code version, and a sampled pass counts only within its sample.
