# cocoyashi

A ride-hailing app, specified as business rules first and built test-first in Go. First market: Indonesia (ID).

## Layout

| Path | What it is |
|---|---|
| `ridehailing.rules.yaml` | The business rules: what must hold, who decided it, how it is checked. |
| `ridehailing.markets.yaml` | The per-market numbers the rules refer to (fees, windows, rates). `null` means not set yet. |
| `ride/` | The core trip loop: request, dispatch, cancel, fare, settle. In memory, no database or HTTP yet. |
| `tests/` | Go tests. Each rule's `check.ref` points at one of these files. |

## Run the tests

```bash
go test ./...
```

## Check the rules

The checker lives in the business-rules tools folder (`tools/business-rules/check_rules.py` in the zou repo).

```bash
python3 <tools>/check_rules.py ridehailing.rules.yaml
```

Exit codes: 0 acceptable, 1 structural error, 2 a confirmed rule failed, 3 a required rule has no evidence yet. Exit 3 is expected until passing test runs are recorded as evidence in the rules file.

## Status

- 34 rules, all confirmed. 24 are required and have no recorded evidence yet.
- LEGAL-1, LEGAL-2, DATA-1 and SAFE-3 are optional for the MVP build but must pass before live rides in Indonesia.
- The fare rates (`fare_base`, `fare_per_km`, `fare_per_minute`, `fare_min`) and the Indonesia cancel fee are not final. No ride gets an estimate in a market until the rates are set.
- Not built yet: driver gating and selfie checks, trip safety, privacy masking, ratings, refunds, surge, promos, account suspension, data retention and the admin console.

## Working on a rule

1. Change the rule in `ridehailing.rules.yaml`. Changing a confirmed rule needs the owner's decision and a higher `revision`.
2. Write the failing test in `tests/` first, then the code in `ride/`.
3. Run `go test ./...` and the rules checker before committing.
