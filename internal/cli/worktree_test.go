package cli

import (
	"testing"
)

func TestWorktreeCmdHasWorkspaceAlias(t *testing.T) {
	// Verify the worktree command has "workspace" as an alias
	aliases := worktreeCmd.Aliases
	found := false
	for _, alias := range aliases {
		if alias == "workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("worktree command should have 'workspace' alias, got aliases: %v", aliases)
	}
}

func TestWorktreeSubcommands(t *testing.T) {
	// Verify worktree has expected subcommands that will work via workspace alias
	expectedSubcmds := []string{"add", "forget", "list", "status"}

	for _, name := range expectedSubcmds {
		found := false
		for _, cmd := range worktreeCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("worktree command should have '%s' subcommand", name)
		}
	}
}
