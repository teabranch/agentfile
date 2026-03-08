# MCP Token Cost Benchmarks

## Background

A widely-cited article by Jannik Reinhard argues that MCP servers are "context hogs" — the GitHub MCP server (93 tools) costs ~55,000 tokens, and an enterprise stack can exceed 150,000 tokens. CLI tools cost 0 tokens since models know them from training data. The article shows a 35x token reduction for CLI over MCP.

**Agentfile agents are fundamentally different.** They expose 6-15 focused tools per agent, not 93 platform-wide tools. Each agent has a single domain with a small, purpose-built toolset. This page documents our methodology for measuring and proving this difference.

## What Agentfile Measures Differently

The article measures platform-wrapper MCP servers: GitHub (93 tools), Jira, Confluence, etc. These are designed to expose an entire platform's API surface as MCP tools.

Agentfile agents follow a different pattern:
- **Focused domain**: each agent does one thing well (Go development, code review, debugging)
- **Small toolset**: 6-15 tools per agent, matching the task at hand
- **System prompt**: a structured prompt tailored to the domain (~2,000-3,000 tokens)
- **Multiple agents**: if you need more capabilities, add another focused agent

This is the "Unix philosophy" applied to MCP: small, composable agents rather than monolithic tool servers.

## Methodology

### Token Estimation

We provide two token counting methods:

1. **bytes/4 heuristic** (`BytesEstimator`): fast, dependency-free, consistent for relative comparisons. Overestimates by ~20% compared to real BPE tokenization.

2. **BPE tokenizer** (`BPECounter`): uses tiktoken's `cl100k_base` encoding (GPT-4's tokenizer). Not Claude's exact tokenizer, but a much more accurate proxy. Shows ~5 bytes per BPE token for JSON schemas, ~5.2 for natural language.

All benchmark tests run both tokenizers side-by-side. The bytes/4 heuristic makes our claims conservative — real BPE numbers show even larger advantages for focused agents.

### What We Measure

1. **Tool schema overhead**: each tool registered via MCP has a JSON schema (name, description, input parameters). We serialize tools as MCP JSON and count bytes/tokens.

2. **System prompt overhead**: the agent's system prompt delivered via `ServerOptions.Instructions` during the MCP handshake.

3. **Total context budget**: schema + prompt as a percentage of the 128K context window.

### Measurement Methods

- **Unit benchmarks** (`benchmarks/`): use in-memory MCP transport (`gomcp.NewInMemoryTransports()`) to capture exactly what the protocol sends, with no subprocess overhead.

- **Integration benchmarks** (`internal/integration/bench_test.go`): build real agent binaries, connect via `gomcp.CommandTransport`, and measure actual wire cost.

- **Synthetic scaling**: generate tool registries at varying sizes (6, 10, 15, 20, 30, 50, 93) with realistic schemas matching builtin tool complexity (~170 bytes/tool average).

## Results

Run `make bench-report` to generate current numbers.

### Scaling Curve

Tool schema tokens grow linearly with tool count (~333 bytes/tool for focused agents):

```
Focused-agent tools (Agentfile-style, 2-3 params):
   6 tools:    477 tokens  (0.4% of 128K)
  10 tools:    836 tokens  (0.7% of 128K)
  15 tools:  1,254 tokens  (1.0% of 128K)  <- recommended max
  93 tools:  7,762 tokens  (6.1% of 128K)

Platform-wrapper tools (GitHub MCP-style, 5-10 params):
   6 tools:  1,999 tokens  (1.6% of 128K)
  10 tools:  3,252 tokens  (2.5% of 128K)
  93 tools: 30,328 tokens (23.7% of 128K)
```

Platform tools average ~1,300 bytes/tool (3.9x larger than focused tools).

### Tokenizer Comparison

The bytes/4 heuristic overestimates by ~20% compared to BPE. With real BPE tokenization (cl100k_base):

```
10 focused tools:    686 BPE tokens (vs 836 heuristic)
93 platform tools: 25,004 BPE tokens (vs 30,328 heuristic)
```

Average BPE/heuristic ratio: 0.80. This means our heuristic-based claims are **conservative** — real token counts are even smaller.

### Multi-Turn Cost Projection

Tool definitions are resent with every Claude API request. Over a 20-turn conversation:

```
Configuration                          Per-turn     1t        5t       10t       20t       50t
GitHub MCP (93 tools, article)          55000T    55000T   275000T   550000T  1100000T  2750000T
Agentfile (10 tools + prompt)            1681T     1681T     8405T    16810T    33620T    84050T

Savings (cumulative tokens avoided):
  Reduction                             53319T    53319T   266595T   533190T  1066380T  2665950T
  Ratio                                    33x       33x       33x       33x       33x       33x
```

Over 20 turns: GitHub MCP costs 1,100,000T cumulative. Agentfile costs 33,620T. **33x reduction.**

### Claude Code Baseline Analysis

Claude Code itself consumes context before any MCP servers are loaded:

```
Claude Code system prompt: ~3,500 tokens
Claude Code built-in tools (23): ~5,700 tokens
Total baseline: ~9,200 tokens (7.2% of 128K)

Marginal cost of adding agents:
  10-tool agent         +1,374 tokens (+14.9% over baseline, 7.2% -> 8.3% of 128K)
  5 community agents    +7,536 tokens (+81.9% over baseline, 7.2% -> 13.1% of 128K)
  GitHub MCP (93 tools) +55,000 tokens (+597.8% over baseline, 7.2% -> 50.2% of 128K)
```

A focused Agentfile agent adds ~15% over Claude Code's existing baseline. The GitHub MCP server adds ~600%.

### Article Methodology Comparison

Using BPE tokenization for the most accurate comparison:

```
Article: GitHub MCP (93 tools) = ~55,000 tokens
Our platform-style (93 tools)  = ~25,004 tokens
Our focused-style (10 tools)   = ~686 tokens

Focused 10-tool agent vs article's 93-tool server: 80x smaller
```

The gap between our platform-style generation (~25K) and the article's measurement (~55K) is because real GitHub MCP schemas have deeper nesting, more enums, and longer descriptions than our synthetic versions. This makes the 80x comparison conservative.

## The 50-Tool Anti-Pattern

50+ tools per agent IS an anti-pattern. The data shows why:

- At 50 tools, schema alone consumes ~7% of the context window
- Add a system prompt and you're approaching 10%
- Stack multiple such servers and you've consumed the entire context

If you need 50 tools, you need multiple agents. Each agent should have:
- A focused domain (one area of expertise)
- 6-15 tools maximum
- A system prompt tailored to its domain

## Design Principle

**One agent, one domain, 6-15 tools max.**

This keeps each agent under 5% of the context budget, leaving 95%+ for actual conversation and reasoning. Even loading 5 focused agents simultaneously stays under 25% — far below the 43% consumed by a single GitHub MCP server.

The multi-turn analysis makes this even more compelling: over a typical 20-turn coding session, the cumulative token cost difference between a focused agent and a platform MCP server is over 1 million tokens.

## How to Reproduce

```bash
# Unit benchmarks
make bench

# Integration benchmarks (builds real binaries)
make bench-integration

# Human-readable comparison report
make bench-report

# All benchmarks
make bench-all
```

### Individual test commands

```bash
# Scaling curve
go test -run TestScalingCurve -v ./benchmarks/

# Community agent measurements
go test -run TestCommunityAgents -v ./benchmarks/

# Anti-pattern analysis
go test -run TestAntiPatternThreshold -v ./benchmarks/

# Full comparison with article
go test -run TestComparisonWithArticle -v ./benchmarks/

# Tokenizer comparison (bytes/4 vs BPE)
go test -run TestTokenizerComparison -v ./benchmarks/

# Multi-turn cost projection
go test -run TestMultiTurnProjection -v ./benchmarks/

# Claude Code baseline analysis
go test -run TestClaudeCodeBaseline -v ./benchmarks/

# Article methodology side-by-side
go test -run TestArticleMethodology -v ./benchmarks/

# Real binary measurements (requires agent builds)
go test -tags integration -run TestMeasureAllAgents -v -timeout 120s ./internal/integration/
```
