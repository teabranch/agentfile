package benchmarks

import (
	"fmt"
	"strings"
)

// TurnProjection holds cost data for a specific turn count.
type TurnProjection struct {
	Turns           int `json:"turns"`
	PerTurnTokens   int `json:"per_turn_tokens"`
	CumulativeTotal int `json:"cumulative_total"`
}

// MultiTurnReport projects per-turn overhead across a conversation.
// In the Claude API, tool definitions are sent with every request — they are
// not a one-time handshake cost. A 20-turn conversation pays the tool schema
// tax 20 times.
type MultiTurnReport struct {
	Label        string           `json:"label"`
	ToolTokens   int              `json:"tool_tokens"`
	PromptTokens int              `json:"prompt_tokens"`
	PerTurn      int              `json:"per_turn"`
	Projections  []TurnProjection `json:"projections"`
}

// ProjectMultiTurn calculates cumulative token cost over multiple conversation turns.
func ProjectMultiTurn(label string, toolTokens, promptTokens int, turns []int) *MultiTurnReport {
	perTurn := toolTokens + promptTokens
	report := &MultiTurnReport{
		Label:        label,
		ToolTokens:   toolTokens,
		PromptTokens: promptTokens,
		PerTurn:      perTurn,
	}
	for _, t := range turns {
		report.Projections = append(report.Projections, TurnProjection{
			Turns:           t,
			PerTurnTokens:   perTurn,
			CumulativeTotal: perTurn * t,
		})
	}
	return report
}

// DefaultTurnCounts returns the standard set of turn counts for projections.
func DefaultTurnCounts() []int {
	return []int{1, 5, 10, 20, 50}
}

// MultiTurnComparison holds side-by-side projections for different configurations.
type MultiTurnComparison struct {
	Reports []*MultiTurnReport
}

// FormatMultiTurnComparison outputs a table comparing cumulative costs.
func FormatMultiTurnComparison(reports []*MultiTurnReport) string {
	var b strings.Builder
	b.WriteString("Multi-turn cumulative cost projection:\n")
	b.WriteString("(Tool definitions are resent with every Claude API request)\n\n")

	// Header.
	fmt.Fprintf(&b, "  %-35s  %8s", "Configuration", "Per-turn")
	if len(reports) > 0 && len(reports[0].Projections) > 0 {
		for _, p := range reports[0].Projections {
			fmt.Fprintf(&b, "  %8s", fmt.Sprintf("%dt", p.Turns))
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %-35s  %8s", strings.Repeat("-", 35), "--------")
	if len(reports) > 0 {
		for range reports[0].Projections {
			fmt.Fprintf(&b, "  %8s", "--------")
		}
	}
	b.WriteString("\n")

	// Rows.
	for _, r := range reports {
		fmt.Fprintf(&b, "  %-35s  %7dT", r.Label, r.PerTurn)
		for _, p := range r.Projections {
			fmt.Fprintf(&b, "  %7dT", p.CumulativeTotal)
		}
		b.WriteString("\n")
	}

	// Savings row if there are exactly 2 reports.
	if len(reports) == 2 {
		b.WriteString("\n  Savings (cumulative tokens avoided):\n")
		big := reports[0]
		small := reports[1]
		fmt.Fprintf(&b, "  %-35s  %7dT", "Reduction", big.PerTurn-small.PerTurn)
		for i, p := range big.Projections {
			saved := p.CumulativeTotal - small.Projections[i].CumulativeTotal
			fmt.Fprintf(&b, "  %7dT", saved)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %-35s  %7.0fx", "Ratio", float64(big.PerTurn)/float64(small.PerTurn))
		for i, p := range big.Projections {
			ratio := float64(p.CumulativeTotal) / float64(small.Projections[i].CumulativeTotal)
			fmt.Fprintf(&b, "  %7.0fx", ratio)
		}
		b.WriteString("\n")
	}

	return b.String()
}
