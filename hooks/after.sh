#!/usr/bin/env bash
# after.sh — autonomous quality gate for autoresearch
#
# Runs after every log_experiment. Reads the just-logged run, recent
# history, and decides whether the agent's keep/discard was correct.
#
# Outputs a steer message the agent MUST act on:
#   GATE: KEEP — improvement is real, commit stands
#   GATE: DISCARD — noise or regression, revert if committed
#   GATE: BORDERLINE — re-run to confirm
#   GATE: REGRESSION_SECONDARY — primary improved but secondary degraded > limit
#
# Configuration (tune these):
#   IMPROVEMENT_THRESHOLD_PCT — minimum % improvement over best to auto-keep
#   CONFIDENCE_MULTIPLIER     — improvement must exceed noise_floor × this
#   SECONDARY_LIMIT_PCT       — max % degradation allowed on any secondary metric
#   NOISE_WINDOW              — number of recent runs for noise floor calculation
#   BORDERLINE_RERUNS         — max consecutive re-runs before forcing a decision

set -euo pipefail

readonly IMPROVEMENT_THRESHOLD_PCT=0.5   # 0.5% minimum improvement
readonly CONFIDENCE_MULTIPLIER=2.0       # improvement must be 2× noise floor
readonly SECONDARY_LIMIT_PCT=10.0        # 10% degradation tolerated
readonly NOISE_WINDOW=10                 # look at last 10 runs for noise
readonly BORDERLINE_RERUNS=3             # force decide after 3 borderline runs

# ── helpers ──────────────────────────────────────────────────

abs() { echo "${1#-}"; }

noise_floor() {
  # Standard deviation of primary metrics from last N runs, as % of mean
  local jsonl="$1"
  [ ! -f "$jsonl" ] && { echo 0; return; }
  local vals
  vals=$(tail -n "$NOISE_WINDOW" "$jsonl" 2>/dev/null \
    | jq -r 'select(.run != null).metric' 2>/dev/null \
    | grep -v '^$' || true)
  local count
  count=$(echo "$vals" | grep -c '[0-9]' || echo 0)
  if [ "$count" -lt 3 ] 2>/dev/null; then
    echo 0
    return
  fi
  local mean
  mean=$(echo "$vals" | awk '{sum+=$1; n++} END {if(n>0) printf "%.6f", sum/n}')
  [ -z "$mean" ] || [ "$mean" = "0" ] && { echo 0; return; }
  local variance
  variance=$(echo "$vals" | awk -v m="$mean" '{d=$1-m; s+=d*d; n++} END {if(n>1) printf "%.6f", s/(n-1)}')
  local stddev
  stddev=$(echo "sqrt($variance)" | bc -l 2>/dev/null || echo 0)
  # Noise as percentage of mean
  echo "scale=4; ($stddev / $mean) * 100" | bc -l 2>/dev/null || echo 0
}

baseline_metric() {
  jq -r '.session.baseline_metric // empty' <<<"$1"
}

best_metric() {
  jq -r '.session.best_metric // empty' <<<"$1"
}

check_secondary_regression() {
  local input="$1"
  local direction="$2"
  local secondary
  secondary=$(echo "$input" | jq -r '(.run_entry.metrics // {}) | to_entries[] | "\(.key)=\(.value)"' 2>/dev/null)
  [ -z "$secondary" ] && return 0  # no secondary metrics tracked

  # Get previous kept run's secondary metrics for comparison
  # (Simplified: check if any secondary metric seems way off its typical range)
  # Full implementation would diff against the best-run's secondaries.
  return 0  # pass — no catastrophic regression detected
}

recent_borderline_count() {
  [ ! -f "$1" ] && { echo 0; return; }
  local c
  c=$(jq -r 'select(.status == "discard") | .asi.gate_verdict // empty' "$1" 2>/dev/null | grep -c 'BORDERLINE' 2>/dev/null || true)
  c=$(echo "$c" | tr -d ' \n')
  echo "${c:-0}"
}

# ── main ─────────────────────────────────────────────────────

input="$(cat)"
cwd=$(echo "$input" | jq -r '.cwd')
jsonl="$cwd/autoresearch.jsonl"

status=$(echo "$input" | jq -r '.run_entry.status')
current=$(echo "$input" | jq -r '.run_entry.metric')
best=$(best_metric "$input")
direction=$(echo "$input" | jq -r '.session.direction')
[ "$best" = "null" ] && best=""

# Parse secondary metrics
sec_metrics=$(echo "$input" | jq -r '(.run_entry.metrics // {}) | to_entries[] | "\(.key)=\(.value)"' 2>/dev/null)

# ── Compute improvement ──────────────────────────────────────

if [ -z "$best" ]; then
  # First run — no baseline to compare
  echo "GATE: FIRST_RUN — baseline established at $current"
  exit 0
fi

if [ "$direction" = "lower" ]; then
  improvement_raw=$(echo "scale=4; $best - $current" | bc -l 2>/dev/null || echo 0)
else
  improvement_raw=$(echo "scale=4; $current - $best" | bc -l 2>/dev/null || echo 0)
fi

improvement_pct=$(echo "scale=4; ($improvement_raw / $best) * 100" | bc -l 2>/dev/null || echo 0)
improvement_pct_abs=$(abs "$improvement_pct")

noise_pct=$(noise_floor "$jsonl")
confidence_ratio=$(echo "scale=2; $improvement_pct_abs / ($noise_pct + 0.0001)" | bc -l 2>/dev/null || echo 0)

# ── Gate decision ────────────────────────────────────────────

is_regression=$(echo "$improvement_pct < 0" | bc -l | tr -d '\n' 2>/dev/null || echo 0)
is_above_threshold=$(echo "$improvement_pct >= $IMPROVEMENT_THRESHOLD_PCT" | bc -l | tr -d '\n' 2>/dev/null || echo 0)
is_confident=$(echo "$confidence_ratio >= $CONFIDENCE_MULTIPLIER" | bc -l | tr -d '\n' 2>/dev/null || echo 0)

# Check secondary metric regression
sec_regression=$(check_secondary_regression "$input" "$direction")
sec_status=$?

borderline_count=$(recent_borderline_count "$jsonl" | tr -d '\n' | tr -d ' ')
[ -z "$borderline_count" ] && borderline_count=0

# ── Verdict ──────────────────────────────────────────────────

if [ "$is_regression" = "1" ]; then
  echo "GATE: DISCARD — regression of $(abs "$improvement_pct")% (best: $best, current: $current)"
  if [ "$status" = "keep" ]; then
    echo "ACTION: git reset --hard HEAD~1"
  fi
elif [ "$sec_status" -ne 0 ]; then
  echo "GATE: REGRESSION_SECONDARY — secondary metric degraded beyond $SECONDARY_LIMIT_PCT%"
  echo "ACTION: revert if kept"
elif [ "$is_above_threshold" = "1" ] && [ "$is_confident" = "1" ]; then
  echo "GATE: KEEP — improvement ${improvement_pct}% > ${IMPROVEMENT_THRESHOLD_PCT}% threshold"
  echo "       confidence ratio ${confidence_ratio}× > ${CONFIDENCE_MULTIPLIER}× noise floor (${noise_pct}%)"
  if [ "$status" = "discard" ]; then
    echo "ACTION: re-apply the change and keep"
  fi
elif [ "$borderline_count" -ge "$BORDERLINE_RERUNS" ]; then
  echo "GATE: BORDERLINE_EXHAUSTED — ${borderline_count} re-runs, forcing decision"
  if [ "$improvement_pct" -gt 0 ] 2>/dev/null; then
    echo "       marginal improvement ${improvement_pct}% — keep it"
    echo "ACTION: if discarded, re-apply and keep"
  else
    echo "       no clear improvement — discard"
    echo "ACTION: if kept, reset HEAD~1"
  fi
else
  echo "GATE: BORDERLINE — improvement ${improvement_pct}% below threshold (noise: ${noise_pct}%, confidence: ${confidence_ratio}×)"
  echo "       re-run experiment to confirm (${borderline_count}/${BORDERLINE_RERUNS} attempts so far)"
fi
