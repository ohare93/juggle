#!/bin/bash
# Juggler: Track activity on each user message

JUGGLER_BIN="${JUGGLER_BIN:-juggler}"

# Silently update activity (errors are ignored)
$JUGGLER_BIN track-activity 2>/dev/null || true
