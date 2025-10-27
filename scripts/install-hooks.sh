#!/bin/bash
# Install Juggler hooks for automatic session tracking
# Currently supports: Claude Code

set -e

HOOKS_DIR="$HOME/.claude/hooks"
JUGGLER_BIN="$(which juggler || echo "$HOME/.local/bin/juggler")"

echo "Installing Juggler hooks to $HOOKS_DIR"
echo "(Claude Code integration)"

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install user-prompt-submit hook (tracks activity)
cat > "$HOOKS_DIR/user-prompt-submit" <<'EOF'
#!/bin/bash
# Juggler: Track activity on each user message

JUGGLER_BIN="${JUGGLER_BIN:-juggler}"

# Silently update activity (errors are ignored)
$JUGGLER_BIN track-activity 2>/dev/null || true
EOF

chmod +x "$HOOKS_DIR/user-prompt-submit"
echo "✓ Installed user-prompt-submit hook"

echo ""
echo "Hooks installed successfully!"
echo ""
echo "To start tracking a session, run:"
echo "  juggler start"
echo ""
echo "To see all sessions:"
echo "  juggler status"
echo ""
echo "To find what needs attention:"
echo "  juggler next"
