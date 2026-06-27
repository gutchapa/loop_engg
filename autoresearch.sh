#!/bin/bash
# No set -e — we want to capture failures, not crash

# Fast pre-check: syntax error in package.json?
python3 -c "import json; json.load(open('package.json'))" 2>/dev/null || true

# Step 1: Clean previous build
rm -rf .next 2>/dev/null || true

# Step 2: Install dependencies — capture all output
INSTALL_OUTPUT=$(npm install 2>&1 || true)

# Step 3: Build — capture exit code and output
BUILD_EXIT=0
BUILD_OUTPUT=$(npm run build 2>&1) || BUILD_EXIT=$?

# Step 4: Check build result
BUILD_TIME_S=$(echo "$BUILD_OUTPUT" | grep -oE 'Completed in [0-9]+m?s|[0-9]+\.[0-9]+s' | grep -oE '[0-9]+\.[0-9]+' | tail -1 || echo "0")
if [ -z "$BUILD_TIME_S" ]; then BUILD_TIME_S="0"; fi

if [ "$BUILD_EXIT" -eq 0 ]; then
  echo "METRIC build_ok=1"
  echo "METRIC build_time_s=$BUILD_TIME_S"
else
  echo "METRIC build_ok=0"
  echo "METRIC build_time_s=0"
fi

# Step 5: Run tests — use JSON reporter for reliable parsing
TESTS_EXIT=0
TESTS_OUTPUT=$(npx vitest run --reporter=json 2>/dev/null) || TESTS_EXIT=$?
TESTS_PASSED=$(echo "$TESTS_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('numPassedTests', 0))" 2>/dev/null || echo "0")

# Primary metric: test_count (number of passing tests)
echo "METRIC test_count=$TESTS_PASSED"

if [ "$TESTS_EXIT" -eq 0 ]; then
  echo "METRIC tests_ok=1"
else
  echo "METRIC tests_ok=0"
fi

# Print tail of build output for agent debugging
echo "---BUILD_OUTPUT_TAIL---"
echo "$BUILD_OUTPUT" | tail -15
echo "---TESTS_OUTPUT_TAIL---"
echo "$TESTS_OUTPUT" | tail -10
echo "---INSTALL_OUTPUT_TAIL---"
echo "$INSTALL_OUTPUT" | tail -10
