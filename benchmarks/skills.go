package benchmarks

import (
	"fmt"
	"strings"
)

// SkillTier represents one level of Agent Skills progressive disclosure.
// Agent Skills (Anthropic's pattern) load context in tiers: metadata first,
// full prompt on activation, and optionally referenced files.
type SkillTier struct {
	Name   string `json:"name"`   // "metadata", "loaded", "with-files"
	Tokens int    `json:"tokens"` // estimated tokens at this tier
	Label  string `json:"label"`  // human-readable description
}

// SkillCost models the context cost and capability set of an Agent Skill.
type SkillCost struct {
	Name  string      `json:"name"`
	Tiers []SkillTier `json:"tiers"`
	// What skills provide.
	HasTools        bool `json:"has_tools"`
	HasMemory       bool `json:"has_memory"`
	HasVersioning   bool `json:"has_versioning"`
	HasDistribution bool `json:"has_distribution"`
}

// AgentfileCost models the context cost and capability set of an Agentfile agent.
type AgentfileCost struct {
	Name         string `json:"name"`
	ToolTokens   int    `json:"tool_tokens"`
	PromptTokens int    `json:"prompt_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	// What Agentfile provides.
	HasTools        bool `json:"has_tools"`
	HasMemory       bool `json:"has_memory"`
	HasVersioning   bool `json:"has_versioning"`
	HasDistribution bool `json:"has_distribution"`
}

// SkillsComparison holds a side-by-side analysis of skills vs Agentfile.
type SkillsComparison struct {
	Skill        SkillCost     `json:"skill"`
	Agentfile    AgentfileCost `json:"agentfile"`
	TokenDelta   int           `json:"token_delta"`   // agentfile - skill loaded tier
	TokenRatio   float64       `json:"token_ratio"`   // agentfile / skill loaded tier
	FeatureDelta []string      `json:"feature_delta"` // capabilities Agentfile adds
}

// DefaultSkillTiers returns the three standard progressive disclosure tiers
// for Agent Skills, based on Anthropic's engineering blog and docs.
//
// The "loaded" tier was calibrated against Claude's count_tokens API.
// Live measurement of community agent prompts (cli-developer: 816T,
// code-reviewer: ~800T, debugger: ~850T) shows ~800T for a typical
// focused agent prompt. The original 3,000T estimate was based on
// blog post examples of larger, more general skill prompts.
//
// Sources:
//   - https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
//   - Live validation via Anthropic count_tokens API (benchmarks/live_test.go)
func DefaultSkillTiers() []SkillTier {
	return []SkillTier{
		{
			Name:   "metadata",
			Tokens: 100,
			Label:  "Name + description only (always in context)",
		},
		{
			Name:   "loaded",
			Tokens: 800,
			Label:  "Full skill prompt loaded into context (live-calibrated)",
		},
		{
			Name:   "with-files",
			Tokens: 5000,
			Label:  "Skill prompt + referenced file contents",
		},
	}
}

// EstimateSkillCost creates a SkillCost with the given tiers.
// Skills provide context-only instructions — no executable tools,
// no persistent memory, no versioning, no distribution.
func EstimateSkillCost(name string, tiers []SkillTier) *SkillCost {
	return &SkillCost{
		Name:            name,
		Tiers:           tiers,
		HasTools:        false,
		HasMemory:       false,
		HasVersioning:   false,
		HasDistribution: false,
	}
}

// CompareSkillsVsAgentfile compares an Agent Skill against an Agentfile agent,
// using the "loaded" tier (full prompt in context) as the comparison point.
func CompareSkillsVsAgentfile(skill *SkillCost, agentfile *AgentfileCost) *SkillsComparison {
	// Find loaded tier for comparison.
	loadedTokens := 0
	for _, t := range skill.Tiers {
		if t.Name == "loaded" {
			loadedTokens = t.Tokens
			break
		}
	}

	ratio := 0.0
	if loadedTokens > 0 {
		ratio = float64(agentfile.TotalTokens) / float64(loadedTokens)
	}

	var featureDelta []string
	if agentfile.HasTools && !skill.HasTools {
		featureDelta = append(featureDelta, "executable tools (MCP)")
	}
	if agentfile.HasMemory && !skill.HasMemory {
		featureDelta = append(featureDelta, "persistent memory")
	}
	if agentfile.HasVersioning && !skill.HasVersioning {
		featureDelta = append(featureDelta, "semantic versioning")
	}
	if agentfile.HasDistribution && !skill.HasDistribution {
		featureDelta = append(featureDelta, "one-command distribution")
	}

	return &SkillsComparison{
		Skill:        *skill,
		Agentfile:    *agentfile,
		TokenDelta:   agentfile.TotalTokens - loadedTokens,
		TokenRatio:   ratio,
		FeatureDelta: featureDelta,
	}
}

// FormatSkillsComparison outputs the skills vs Agentfile analysis as text.
func FormatSkillsComparison(cmp *SkillsComparison) string {
	var b strings.Builder
	b.WriteString("Agent Skills vs Agentfile comparison:\n\n")

	b.WriteString("  Agent Skills progressive disclosure tiers:\n")
	for _, t := range cmp.Skill.Tiers {
		fmt.Fprintf(&b, "    %-12s ~%dT  (%s)\n", t.Name+":", t.Tokens, t.Label)
	}

	fmt.Fprintf(&b, "\n  Agentfile agent (%s):\n", cmp.Agentfile.Name)
	fmt.Fprintf(&b, "    Tool schemas:  ~%dT\n", cmp.Agentfile.ToolTokens)
	fmt.Fprintf(&b, "    System prompt: ~%dT\n", cmp.Agentfile.PromptTokens)
	fmt.Fprintf(&b, "    Total:         ~%dT\n", cmp.Agentfile.TotalTokens)

	b.WriteString("\n  Token comparison (vs loaded skill tier):\n")
	fmt.Fprintf(&b, "    Skill loaded:    ~%dT\n", cmp.Skill.Tiers[1].Tokens)
	fmt.Fprintf(&b, "    Agentfile total: ~%dT\n", cmp.Agentfile.TotalTokens)
	fmt.Fprintf(&b, "    Delta:           %+dT (%.1fx)\n", cmp.TokenDelta, cmp.TokenRatio)

	b.WriteString("\n  Capabilities Agentfile adds over Skills:\n")
	for _, f := range cmp.FeatureDelta {
		fmt.Fprintf(&b, "    + %s\n", f)
	}

	b.WriteString("\n  Capabilities comparison:\n")
	fmt.Fprintf(&b, "    %-25s  Skills  Agentfile\n", "Feature")
	fmt.Fprintf(&b, "    %-25s  ------  ---------\n", strings.Repeat("-", 25))
	fmt.Fprintf(&b, "    %-25s  %-6s  %-9s\n", "Executable tools (MCP)", boolMark(cmp.Skill.HasTools), boolMark(cmp.Agentfile.HasTools))
	fmt.Fprintf(&b, "    %-25s  %-6s  %-9s\n", "Persistent memory", boolMark(cmp.Skill.HasMemory), boolMark(cmp.Agentfile.HasMemory))
	fmt.Fprintf(&b, "    %-25s  %-6s  %-9s\n", "Semantic versioning", boolMark(cmp.Skill.HasVersioning), boolMark(cmp.Agentfile.HasVersioning))
	fmt.Fprintf(&b, "    %-25s  %-6s  %-9s\n", "One-command distribution", boolMark(cmp.Skill.HasDistribution), boolMark(cmp.Agentfile.HasDistribution))

	return b.String()
}

func boolMark(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
