package api

import (
	"encoding/json"
	"fmt"
)

type Instance struct {
	Status          string `json:"status"`
	Model           string `json:"model"`
	Lane            string `json:"lane"`
	IP              string `json:"ip"`
	GPUModel        string `json:"gpuModel"`
	Version         string `json:"version"`
	VersionMismatch bool   `json:"versionMismatch"`
}

type Worker struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	InstanceCount      int                 `json:"instanceCount"`
	DailyCost          float64             `json:"dailyCost"`
	TokensPerInstance  int64               `json:"tokensPerInstance"`
	TokensLast24H      int64               `json:"tokensLast24h"`
	TotalTokens        int64               `json:"totalTokens"`
	GenerationsLast24H int                 `json:"generationsLast24h"`
	GenerationsHistory []GenerationHistory `json:"generationsHistory"`
	Instances          []Instance          `json:"instances"`
	TokenHistory       []TokenHistory      `json:"tokenHistory"`
}

type WorkerResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				Status  string `json:"status"`
				Workers []struct {
					ID         string `json:"_id"`
					Name       string `json:"name"`
					IsArchived bool   `json:"isArchived"`
					TeamID     string `json:"teamId"`
					Instances  []struct {
						Status          string `json:"status"`
						PoolAssignments []struct {
							Lane string `json:"lane"`
						} `json:"poolAssignments"`
						Info struct {
							Runtime   string `json:"runtime"`
							Version   string `json:"version"`
							IPAddress string `json:"ipAddress"`
							NvidiaSmi struct {
								GPU []struct {
									ProductName []string `json:"product_name"`
								} `json:"gpu"`
							} `json:"nvidiaSmi"`
						} `json:"info"`
					} `json:"instances"`
				} `json:"workers"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

func (c *Client) GetWorkers() ([]Worker, error) {
	// Load GPU prices
	gpuPrices, err := LoadGPUPrices("instance.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load GPU prices: %w", err)
	}

	// Get current CLI version first
	kuzcoClient := NewKuzcoClient(c)
	cliVersion, err := kuzcoClient.GetVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get CLI version: %w", err)
	}

	respBody, err := c.DoRequest("GET", "worker.list?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}

	var resp []WorkerResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	var workers []Worker
	for _, w := range resp[0].Result.Data.JSON.Workers {
		if w.IsArchived {
			continue
		}

		// Get metrics for this worker and log the response
		metricsPayload := map[string]interface{}{
			"workerId":     w.ID,
			"workerTeamId": w.TeamID,
		}

		tokens24h, err := kuzcoClient.GetMetrics(MetricsQuery{
			Endpoint: EndpointMetricsTokensLast24Hours,
			Payload:  metricsPayload,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get worker tokens last 24h: %w", err)
		}

		totalTokens, err := kuzcoClient.GetMetrics(MetricsQuery{
			Endpoint: EndpointMetricsTokensAllTime,
			Payload: map[string]interface{}{
				"workerId":     w.ID,
				"workerTeamId": w.TeamID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get worker total tokens: %w", err)
		}

		generations24h, err := kuzcoClient.GetMetrics(MetricsQuery{
			Endpoint: EndpointMetricsGenerationsLast24Hours,
			Payload: map[string]interface{}{
				"workerId":     w.ID,
				"workerTeamId": w.TeamID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get worker generations last 24h: %w", err)
		}

		// Get worker generations history
		genHistory, err := kuzcoClient.GetWorkerGenerationsHistory(w.ID, w.TeamID, 2)
		if err != nil {
			return nil, fmt.Errorf("failed to get worker generations history: %w", err)
		}

		worker := Worker{
			ID:                 w.ID,
			Name:               w.Name,
			InstanceCount:      len(w.Instances),
			TokensLast24H:      int64(tokens24h),
			TotalTokens:        int64(totalTokens),
			GenerationsLast24H: int(generations24h),
			Instances:          make([]Instance, 0, len(w.Instances)),
			GenerationsHistory: genHistory,
		}

		// Calculate TokensPerInstance only if there are instances
		if len(w.Instances) > 0 {
			worker.TokensPerInstance = int64(tokens24h) / int64(len(w.Instances))
		}

		// Calculate daily cost based on GPU type and instance count
		dailyCost := 0.0
		for _, inst := range w.Instances {
			var gpuModel string
			if len(inst.Info.NvidiaSmi.GPU) > 0 && len(inst.Info.NvidiaSmi.GPU[0].ProductName) > 0 {
				gpuModel = normalizeGPUName(inst.Info.NvidiaSmi.GPU[0].ProductName[0])
			}

			if price, ok := gpuPrices[gpuModel]; ok {
				dailyCost += price
			}

			var model, lane string
			if len(inst.PoolAssignments) > 0 {
				model = inst.Info.Runtime
				lane = inst.PoolAssignments[0].Lane
			}

			var version string
			var versionMismatch bool
			var versionDiff string
			if inst.Info.Version != "" {
				version = inst.Info.Version
				versionMismatch, versionDiff = compareVersions(version, cliVersion)
				if versionMismatch {
					version = fmt.Sprintf("%s (%s)", version, versionDiff)
				}
			}

			instance := Instance{
				Status:          inst.Status,
				Model:           model,
				Lane:            lane,
				IP:              inst.Info.IPAddress,
				GPUModel:        gpuModel,
				Version:         version,
				VersionMismatch: versionMismatch,
			}
			worker.Instances = append(worker.Instances, instance)
		}

		worker.DailyCost = dailyCost

		workers = append(workers, worker)
	}

	return workers, nil
}
