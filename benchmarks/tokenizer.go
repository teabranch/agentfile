package benchmarks

import (
	"encoding/json"
	"fmt"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/teabranch/agentfile/pkg/tools"
)

// TokenCounter counts tokens in a string.
type TokenCounter interface {
	Count(s string) int
	Name() string
}

// BytesEstimator estimates tokens using the bytes/4 heuristic.
// Fast and dependency-free, but overestimates compression of structured text
// like JSON schemas. Useful for relative comparisons.
type BytesEstimator struct{}

func (BytesEstimator) Count(s string) int { return (len(s) + 3) / 4 }
func (BytesEstimator) Name() string       { return "bytes/4" }

// BPECounter counts tokens using tiktoken's cl100k_base encoding.
// This is the BPE tokenizer used by GPT-4 and is a reasonable proxy for
// Claude's tokenizer. Not exact, but much more accurate than bytes/4 for
// structured text like JSON schemas.
type BPECounter struct {
	enc *tiktoken.Tiktoken
}

var (
	bpeOnce    sync.Once
	bpeCounter *BPECounter
	bpeErr     error
)

// NewBPECounter returns a BPE token counter using cl100k_base encoding.
// The encoding data is downloaded on first use and cached. Returns an error
// if the encoding cannot be loaded (e.g., no network on first run).
func NewBPECounter() (*BPECounter, error) {
	bpeOnce.Do(func() {
		enc, err := tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			bpeErr = fmt.Errorf("initializing cl100k_base encoding: %w", err)
			return
		}
		bpeCounter = &BPECounter{enc: enc}
	})
	return bpeCounter, bpeErr
}

func (c *BPECounter) Count(s string) int {
	return len(c.enc.Encode(s, nil, nil))
}

func (c *BPECounter) Name() string { return "cl100k_base (BPE)" }

// TokenizerComparison holds side-by-side measurements from two counters.
type TokenizerComparison struct {
	Label      string  `json:"label"`
	ByteCount  int     `json:"byte_count"`
	Heuristic  int     `json:"heuristic_tokens"`
	BPE        int     `json:"bpe_tokens"`
	Ratio      float64 `json:"ratio"` // BPE / Heuristic
	BytesPerBP float64 `json:"bytes_per_bpe_token"`
}

// CompareTokenizers measures the same content with both counters.
func CompareTokenizers(label, content string, bpe *BPECounter) TokenizerComparison {
	heuristic := EstimateTokens(content)
	bpeCount := bpe.Count(content)
	ratio := 0.0
	if heuristic > 0 {
		ratio = float64(bpeCount) / float64(heuristic)
	}
	bytesPerBPE := 0.0
	if bpeCount > 0 {
		bytesPerBPE = float64(len(content)) / float64(bpeCount)
	}
	return TokenizerComparison{
		Label:      label,
		ByteCount:  len(content),
		Heuristic:  heuristic,
		BPE:        bpeCount,
		Ratio:      ratio,
		BytesPerBP: bytesPerBPE,
	}
}

// MeasureToolsWith measures tools using the given TokenCounter.
// If counter is nil, falls back to BytesEstimator.
func MeasureToolsWith(defs []*tools.Definition, counter TokenCounter) *RegistryMeasurement {
	if counter == nil {
		counter = BytesEstimator{}
	}

	m := &RegistryMeasurement{
		ToolCount: len(defs),
		PerTool:   make([]ToolMeasurement, 0, len(defs)),
	}

	for _, def := range defs {
		tool := mcpToolJSON{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		}
		data, _ := json.Marshal(tool)
		tokens := counter.Count(string(data))
		tm := ToolMeasurement{
			Name:         def.Name,
			SchemaBytes:  len(data),
			SchemaTokens: tokens,
		}
		m.PerTool = append(m.PerTool, tm)
		m.SchemaBytes += len(data)
		m.SchemaTokens += tokens
	}

	return m
}
