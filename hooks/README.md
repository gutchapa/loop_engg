# Loop Engineering Hooks

Autonomous quality gates that enforce keep/discard decisions.

## after.sh

Statistical gate — runs after every experiment log:
- Computes noise floor from run history
- Validates improvement exceeds threshold
- Checks secondary metrics for regression
- Outputs machine-actionable `GATE:` verdict

## Adding more hooks

See `/autoresearch.hooks/SKILL.md` for the full hook contract.
Both `before.sh` and `after.sh` are supported.
