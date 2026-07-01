package usage_test

import (
	"testing"

	"github.com/tanlian/agent_nova/internal/usage"
)

func TestEstimateCostUSD(t *testing.T) {
	tests := []struct {
		model              string
		prompt, completion int
		minUSD             float64
	}{
		{"gpt-4o-mini", 15000, 18545, 0.005},
		{"gpt-4o", 15000, 18545, 0.05},
		{"", 33545, 0, 0.05},
	}
	for _, tc := range tests {
		got := usage.EstimateCostUSD(tc.model, tc.prompt, tc.completion)
		if got < tc.minUSD {
			t.Errorf("EstimateCostUSD(%q, %d, %d) = %.6f, want >= %.4f",
				tc.model, tc.prompt, tc.completion, got, tc.minUSD)
		}
	}
}
