package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectStats 项目级 LLM 用量累计（存于 .nova/usage_stats.json）。
type ProjectStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	WriteRuns        int `json:"write_runs"`
}

func (s ProjectStats) TotalTokens() int {
	return s.PromptTokens + s.CompletionTokens
}

func statsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".nova", "usage_stats.json")
}

// Load 读取项目用量；文件不存在时返回零值。
func Load(projectRoot string) (ProjectStats, error) {
	data, err := os.ReadFile(statsPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectStats{}, nil
		}
		return ProjectStats{}, err
	}
	var s ProjectStats
	if err := json.Unmarshal(data, &s); err != nil {
		return ProjectStats{}, err
	}
	return s, nil
}

// AddWriteRun 累加一次写章流水线的 token 用量。
func AddWriteRun(projectRoot string, prompt, completion int) (ProjectStats, error) {
	s, err := Load(projectRoot)
	if err != nil {
		return ProjectStats{}, err
	}
	s.PromptTokens += prompt
	s.CompletionTokens += completion
	s.WriteRuns++
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return ProjectStats{}, err
	}
	dir := filepath.Join(projectRoot, ".nova")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ProjectStats{}, err
	}
	if err := os.WriteFile(statsPath(projectRoot), data, 0o644); err != nil {
		return ProjectStats{}, err
	}
	return s, nil
}

// EstimateCostUSD 按常见 OpenAI 定价粗估美元成本（$/1M tokens，仅供参考）。
func EstimateCostUSD(model string, prompt, completion int) float64 {
	// 单位：美元 / 100 万 tokens
	inRate, outRate := 2.5, 10.0 // gpt-4o 档
	switch {
	case strings.Contains(model, "mini"):
		inRate, outRate = 0.15, 0.60
	case strings.Contains(model, "4o"):
		inRate, outRate = 2.5, 10.0
	case strings.Contains(model, "3.5"):
		inRate, outRate = 0.5, 1.5
	case strings.Contains(strings.ToLower(model), "deepseek"):
		inRate, outRate = 0.14, 0.28
	}
	return float64(prompt)/1_000_000*inRate + float64(completion)/1_000_000*outRate
}
