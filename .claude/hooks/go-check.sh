#!/usr/bin/env bash

f=$(jq -r '.tool_input.file_path // ""')
[[ "$f" == *.go ]] || exit 0

# Auto-format
gofmt -w "$f" 2>/dev/null || true

# Collect failures
out=""
vet=$(go vet ./... 2>&1) || out+="go vet:\n$vet\n\n"
test_out=$(go test ./... 2>&1) || out+="go test:\n$test_out\n\n"

[[ -z "$out" ]] && exit 0

ctx=$(printf '%s' "$out" | jq -Rs '.')
printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":%s}}' "$ctx"
