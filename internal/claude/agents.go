package claude

// AgentConfig defines configuration for a specific AI coding agent
type AgentConfig struct {
	Name             string
	InstructionPaths []string // Preferred file locations in order
	MarkerStart      string
	MarkerEnd        string
}

// SupportedAgents maps agent type names to their configurations
var SupportedAgents = map[string]AgentConfig{
	"claude": {
		Name:             "Claude",
		InstructionPaths: []string{".claude/CLAUDE.md", "CLAUDE.md", "AGENTS.md", ".claude/AGENTS.md"},
		MarkerStart:      "<!-- juggler-instructions-start -->",
		MarkerEnd:        "<!-- juggler-instructions-end -->",
	},
	"cursor": {
		Name:             "Cursor",
		InstructionPaths: []string{".cursorrules", "AGENTS.md", ".claude/AGENTS.md"},
		MarkerStart:      "<!-- juggler-instructions-start -->",
		MarkerEnd:        "<!-- juggler-instructions-end -->",
	},
	"aider": {
		Name:             "Aider",
		InstructionPaths: []string{".aider.conf.yml", "AGENTS.md", ".claude/AGENTS.md"},
		MarkerStart:      "# juggler-instructions-start",
		MarkerEnd:        "# juggler-instructions-end",
	},
}

// GetAgentConfig returns the configuration for a given agent type
// Returns false if the agent type is not supported
func GetAgentConfig(agentType string) (AgentConfig, bool) {
	config, ok := SupportedAgents[agentType]
	return config, ok
}

// ListSupportedAgents returns a list of all supported agent type names
func ListSupportedAgents() []string {
	agents := make([]string, 0, len(SupportedAgents))
	for agentType := range SupportedAgents {
		agents = append(agents, agentType)
	}
	return agents
}
