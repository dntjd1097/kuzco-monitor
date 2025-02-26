package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	DiskCostPerDay = 0.01 * 24 // $0.01 per hour * 24 hours
)

// Add this helper function to parse and compare versions
func compareVersions(v1, v2 string) (bool, string) {
	// Extract version numbers (e.g., "0.2.3" from "0.2.3-fe4d73f")
	v1Parts := strings.Split(v1, "-")[0]
	v2Parts := strings.Split(v2, "-")[0]

	v1Numbers := strings.Split(v1Parts, ".")
	v2Numbers := strings.Split(v2Parts, ".")

	// Compare each number part
	for i := 0; i < len(v1Numbers) && i < len(v2Numbers); i++ {
		n1, err1 := strconv.Atoi(v1Numbers[i])
		n2, err2 := strconv.Atoi(v2Numbers[i])

		if err1 != nil || err2 != nil {
			continue
		}

		if n1 != n2 {
			if n1 > n2 {
				return true, fmt.Sprintf("newer (%s > %s)", v1Parts, v2Parts)
			} else {
				return true, fmt.Sprintf("older (%s < %s)", v1Parts, v2Parts)
			}
		}
	}

	// If all numbers match, versions are considered the same
	return false, ""
}

// Add these methods to format metrics as key-value pairs
func (m GeneralMetrics) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`
General Metrics:
  CLI Version: %s
  Running Instance Count: %d
  Requests Per Minute: %d
  Tokens (Last 24h): %d
  Total Tokens: %d
  Generations (Last 24h): %d`,
		m.CLIVersion,
		m.RunningInstanceCount,
		m.RPM,
		m.TokensLast24Hours,
		m.TokensAllTime,
		m.GenerationsLast24Hours))

	if len(m.GenerationsHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n  Generation History (Last 1h):  %s: %d",
			m.GenerationsHistory[0].Date,
			m.GenerationsHistory[0].Value))
	}

	return b.String()
}

func (m UserMetrics) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`
User Metrics:
  Tokens (Last 24h): %d
  Total Tokens: %d
  Generations (Last 24h): %d
  Total Instances: %d`,
		m.TokensLast24Hours,
		m.TokensAllTime,
		m.GenerationsLast24Hours,
		m.TotalInstances))

	// Calculate total daily cost
	totalDailyCost := 0.0
	for _, w := range m.Workers {
		totalDailyCost += w.DailyCost
	}
	b.WriteString(fmt.Sprintf("\n  Total Daily Cost: $%.2f", totalDailyCost))

	// Calculate average tokens per instance
	if m.TotalInstances > 0 {
		tokensPerInstance := m.TokensLast24Hours / int64(m.TotalInstances)
		b.WriteString(fmt.Sprintf("\n  Tokens (24h)/Instance: %d", tokensPerInstance))
	}

	if len(m.GenerationsHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n  Generation History (Last 1h): %d",
			m.GenerationsHistory[0].Value))
	}

	if len(m.Workers) > 0 {
		b.WriteString("\n  Workers:")
		for _, w := range m.Workers {
			b.WriteString(w.String())
		}
	}

	return b.String()
}

func (m *Metrics) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`
General Metrics:
  CLI Version: %s
  Running Instance Count: %d
  Requests Per Minute: %d
  Tokens (Last 24h): %d
  Total Tokens: %d
  Generations (Last 24h): %d`,
		m.General.CLIVersion,
		m.General.RunningInstanceCount,
		m.General.RPM,
		m.General.TokensLast24Hours,
		m.General.TokensAllTime,
		m.General.GenerationsLast24Hours))

	if len(m.General.GenerationsHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n  Generation History (Last 1h):  %s: %d",
			m.General.GenerationsHistory[0].Date,
			m.General.GenerationsHistory[0].Value))
	}

	b.WriteString(fmt.Sprintf("\n\n%s", m.User)) // UserMetrics의 String() 메서드 호출
	return b.String()
}

func (w Worker) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n    %s (%s):", w.Name, w.ID))
	b.WriteString(fmt.Sprintf("\n      Tokens (24h): %d", w.TokensLast24H))
	b.WriteString(fmt.Sprintf("\n      Total Tokens: %d", w.TotalTokens))
	b.WriteString(fmt.Sprintf("\n      Generations (24h): %d", w.GenerationsLast24H))
	b.WriteString(fmt.Sprintf("\n      Daily Cost: $%.2f", w.DailyCost))

	// Calculate tokens per instance
	if w.InstanceCount > 0 {
		tokensPerInstance := w.TokensLast24H / int64(w.InstanceCount)
		b.WriteString(fmt.Sprintf("\n      Tokens (24h)/Instance: %d", tokensPerInstance))
	}

	if len(w.GenerationsHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n      Generation History (Last 1h): %d",
			w.GenerationsHistory[0].Value))
	}

	if len(w.TokenHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n      Tokens History (Last 1h): %d",
			w.TokenHistory[len(w.TokenHistory)-1].Value))
	}

	b.WriteString(fmt.Sprintf("\n      Instance Count: %d", w.InstanceCount))

	for _, inst := range w.Instances {
		b.WriteString(fmt.Sprintf("\n      - Status: %s", inst.Status))
		b.WriteString(fmt.Sprintf("\n      - Model: %s, Lane: %s, Version: %s",
			inst.Model, inst.Lane, inst.Version))
		b.WriteString(fmt.Sprintf("\n        IP: %s, GPU: %s",
			inst.IP, inst.GPUModel))
	}

	return b.String()
}

func PrintPrettierJson(data interface{}) {
	prettyJson, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal data: %v", err)
	}
	fmt.Println(string(prettyJson))
}

type GPUInstance struct {
	Gpu   string  `json:"Gpu"`
	Price float64 `json:"Price"`
}

func LoadGPUPrices(path string) (map[string]float64, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading GPU prices file: %w", err)
	}

	var instances []GPUInstance
	if err := json.Unmarshal(file, &instances); err != nil {
		return nil, fmt.Errorf("error parsing GPU prices: %w", err)
	}

	// Create a map for easier lookup
	prices := make(map[string]float64)
	for _, inst := range instances {
		// GPU cost per day + disk cost per day
		prices[inst.Gpu] = (inst.Price * 24) + DiskCostPerDay
	}

	return prices, nil
}

func normalizeGPUName(gpuName string) string {
	// Remove "NVIDIA GeForce " or "NVIDIA " prefix
	gpuName = strings.TrimPrefix(gpuName, "NVIDIA GeForce ")
	gpuName = strings.TrimPrefix(gpuName, "NVIDIA ")

	// Map common GPU names
	gpuMap := map[string]string{
		"RTX 3060":    "RTX 3060",
		"RTX 3060 Ti": "RTX 3060 Ti",
		"RTX 3070":    "RTX 3070",
		"RTX 3070 Ti": "RTX 3070 Ti",
		"RTX 3080":    "RTX 3080",
		"RTX 3080 Ti": "RTX 3080 Ti",
		"RTX 3090":    "RTX 3090",
		"RTX 4060":    "RTX 4060",
		"RTX 4060 Ti": "RTX 4060 Ti",
		"RTX 4080":    "RTX 4080S",
		"RTX 4090":    "RTX 4090",
		"A4000":       "RTX A4000",
	}

	if normalized, ok := gpuMap[gpuName]; ok {
		return normalized
	}
	return gpuName
}

// CodeBlock wraps text in a code block for Telegram markdown
func CodeBlock(text string) string {
	return fmt.Sprintf("```\n%s\n```", text)
}
