package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Status string

const (
	StatusDone        Status = "completed"
	StatusPartial     Status = "partial"
	StatusNeedsAction Status = "needs_action"
	StatusFailed      Status = "failed"
)

type Report struct {
	Stage       string   `json:"stage"`
	Status      Status   `json:"status"`
	Summary     string   `json:"summary"`
	Artifacts   []string `json:"artifacts,omitempty"`
	Issues      []string `json:"issues,omitempty"`
	NextSteps   []string `json:"next_steps,omitempty"`
}

func (r *Report) PrintText() {
	statusLabel := map[Status]string{
		StatusDone:        "已完成",
		StatusPartial:     "部分完成",
		StatusNeedsAction: "需要你处理",
		StatusFailed:      "未完成",
	}
	fmt.Printf("\n=== %s ===\n", r.Stage)
	fmt.Printf("状态: %s\n", statusLabel[r.Status])
	fmt.Printf("%s\n\n", r.Summary)
	if len(r.Artifacts) > 0 {
		fmt.Println("产出:")
		for _, a := range r.Artifacts {
			fmt.Printf("  - %s\n", a)
		}
		fmt.Println()
	}
	if len(r.Issues) > 0 {
		fmt.Println("问题:")
		for _, i := range r.Issues {
			fmt.Printf("  - %s\n", i)
		}
		fmt.Println()
	}
	if len(r.NextSteps) > 0 {
		fmt.Println("下一步:")
		for _, n := range r.NextSteps {
			fmt.Printf("  - %s\n", n)
		}
	}
}

func (r *Report) Print(format string) error {
	switch strings.ToLower(format) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	default:
		r.PrintText()
		return nil
	}
}
