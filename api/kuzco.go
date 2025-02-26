package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// KuzcoClient handles Kuzco API interactions
type KuzcoClient struct {
	httpClient *Client
}

// NewKuzcoClient creates a new Kuzco client
func NewKuzcoClient(client *Client) *KuzcoClient {
	return &KuzcoClient{
		httpClient: client,
	}
}

// GetMetrics retrieves metrics from Kuzco API
func (c *KuzcoClient) GetMetrics(query MetricsQuery) (float64, error) {
	var input string
	if query.Payload == nil {
		input = "{\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}"
	} else {
		inputJSON, err := json.Marshal(map[string]interface{}{
			"0": map[string]interface{}{
				"json": query.Payload,
			},
		})
		if err != nil {
			return 0, fmt.Errorf("error creating input: %w", err)
		}
		input = string(inputJSON)
	}

	respBody, err := c.httpClient.DoRequest("GET", query.Endpoint+"?batch=1&input="+input, nil, nil)
	if err != nil {
		return 0, err
	}

	var resp []MetricsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return 0, fmt.Errorf("empty response received")
	}

	// Try to handle different response formats
	switch v := resp[0].Result.Data.JSON.(type) {
	case float64:
		return v, nil
	case map[string]interface{}:
		if val, ok := v["json"].(float64); ok {
			return val, nil
		}
		// Handle null value in json field
		if v["json"] == nil {
			return 0, nil
		}
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case nil:
		return 0, nil
	}

	// If we can't parse it directly, try to parse it as a number
	if numStr, ok := resp[0].Result.Data.JSON.(string); ok {
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			return num, nil
		}
	}

	return 0, fmt.Errorf("invalid response format: %v", resp[0].Result.Data.JSON)
}

// GetRunningInstanceCount retrieves the count of running instances
func (c *KuzcoClient) GetRunningInstanceCount() (int, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsRunningInstanceCount,
		Payload:  nil,
	})
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// GetRPM retrieves the requests per minute
func (c *KuzcoClient) GetRPM() (int, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsRPM,
		Payload:  nil,
	})
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// GetTokensLast24Hours retrieves the token count for the last 24 hours
func (c *KuzcoClient) GetTokensLast24Hours() (int64, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsTokensLast24Hours,
		Payload:  map[string]interface{}{},
	})
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// GetTokensAllTime retrieves the total token count
func (c *KuzcoClient) GetTokensAllTime() (int64, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsTokensAllTime,
		Payload:  map[string]interface{}{},
	})
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// GetGenerationsLast24Hours retrieves the generation count for the last 24 hours
func (c *KuzcoClient) GetGenerationsLast24Hours() (int, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsGenerationsLast24Hours,
		Payload:  map[string]interface{}{},
	})
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// GetUserTokensLast24Hours retrieves the user's token count for the last 24 hours
func (c *KuzcoClient) GetUserTokensLast24Hours(userID string) (int64, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsTokensLast24Hours,
		Payload:  map[string]interface{}{"workerTeamId": userID},
	})
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// GetUserTokensAllTime retrieves the user's total token count
func (c *KuzcoClient) GetUserTokensAllTime(userID string) (int64, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsTokensAllTime,
		Payload:  map[string]interface{}{"workerTeamId": userID},
	})
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// GetUserGenerationsLast24Hours retrieves the user's generation count for the last 24 hours
func (c *KuzcoClient) GetUserGenerationsLast24Hours(userID string) (int, error) {
	value, err := c.GetMetrics(MetricsQuery{
		Endpoint: EndpointMetricsGenerationsLast24Hours,
		Payload: map[string]interface{}{
			"workerTeamId": userID,
		},
	})
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// GetGenerationsHistory retrieves the generation history
func (c *KuzcoClient) GetGenerationsHistory(hoursBack int) ([]GenerationHistory, error) {
	respBody, err := c.httpClient.DoRequest("GET", fmt.Sprintf("%s?batch=1&input={\"0\":{\"json\":{\"hoursBack\":%d}}}",
		EndpointMetricsGenerationsHistory, hoursBack), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp []GenerationHistoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	return resp[0].Result.Data.JSON, nil
}

// GetUserGenerationsHistory retrieves the user's generation history
func (c *KuzcoClient) GetUserGenerationsHistory(userID string, hoursBack int) ([]GenerationHistory, error) {
	respBody, err := c.httpClient.DoRequest("GET", fmt.Sprintf("%s?batch=1&input={\"0\":{\"json\":{\"hoursBack\":%d,\"workerTeamId\":\"%s\"}}}",
		EndpointMetricsGenerationsHistory, hoursBack, userID), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp []GenerationHistoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	return resp[0].Result.Data.JSON, nil
}

// GetWorkerGenerationsHistory retrieves the worker's generation history
func (c *KuzcoClient) GetWorkerGenerationsHistory(workerID string, teamID string, hoursBack int) ([]GenerationHistory, error) {
	respBody, err := c.httpClient.DoRequest("GET", fmt.Sprintf("%s?batch=1&input={\"0\":{\"json\":{\"hoursBack\":%d,\"workerId\":\"%s\",\"workerTeamId\":\"%s\"}}}",
		EndpointMetricsGenerationsHistory, hoursBack, workerID, teamID), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp []GenerationHistoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	return resp[0].Result.Data.JSON, nil
}

// GetVersions retrieves the CLI version information
func (c *KuzcoClient) GetVersions() (string, error) {
	respBody, err := c.httpClient.DoRequest("GET", EndpointSystemBucketVersions+"?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}", nil, nil)
	if err != nil {
		return "", err
	}

	var resp []VersionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		return "", fmt.Errorf("empty response received")
	}

	return resp[0].Result.Data.JSON.CLIVersion, nil
}

// GetAllMetrics retrieves all metrics for a user
func (c *KuzcoClient) GetAllMetrics(userID string) (*Metrics, error) {
	metrics := &Metrics{}
	var err error

	// Get version info
	if metrics.General.CLIVersion, err = c.GetVersions(); err != nil {
		return nil, fmt.Errorf("failed to get CLI version: %w", err)
	}

	// Get General metrics
	if metrics.General.RunningInstanceCount, err = c.GetRunningInstanceCount(); err != nil {
		return nil, fmt.Errorf("failed to get running instance count: %w", err)
	}
	if metrics.General.RPM, err = c.GetRPM(); err != nil {
		return nil, fmt.Errorf("failed to get RPM: %w", err)
	}

	if metrics.General.TokensLast24Hours, err = c.GetTokensLast24Hours(); err != nil {
		return nil, fmt.Errorf("failed to get tokens last 24h: %w", err)
	}

	if metrics.General.TokensAllTime, err = c.GetTokensAllTime(); err != nil {
		return nil, fmt.Errorf("failed to get total tokens: %w", err)
	}

	if metrics.General.GenerationsLast24Hours, err = c.GetGenerationsLast24Hours(); err != nil {
		return nil, fmt.Errorf("failed to get generations last 24h: %w", err)
	}

	if metrics.General.GenerationsHistory, err = c.GetGenerationsHistory(2); err != nil {
		return nil, fmt.Errorf("failed to get generations history: %w", err)
	}

	// Get User metrics
	if metrics.User.TokensLast24Hours, err = c.GetUserTokensLast24Hours(userID); err != nil {
		return nil, fmt.Errorf("failed to get user tokens last 24h: %w", err)
	}

	if metrics.User.TokensAllTime, err = c.GetUserTokensAllTime(userID); err != nil {
		return nil, fmt.Errorf("failed to get user total tokens: %w", err)
	}

	if metrics.User.GenerationsLast24Hours, err = c.GetUserGenerationsLast24Hours(userID); err != nil {
		return nil, fmt.Errorf("failed to get user generations last 24h: %w", err)
	}

	if metrics.User.GenerationsHistory, err = c.GetUserGenerationsHistory(userID, 2); err != nil {
		return nil, fmt.Errorf("failed to get user generations history: %w", err)
	}

	// Get Worker information
	if metrics.User.Workers, err = c.httpClient.GetWorkers(); err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}

	// Calculate totals
	metrics.User.TotalInstances = 0
	metrics.User.TotalDailyCost = 0
	for _, w := range metrics.User.Workers {
		metrics.User.TotalInstances += w.InstanceCount
		metrics.User.TotalDailyCost += w.DailyCost
	}

	// Calculate tokens per instance
	if metrics.User.TotalInstances > 0 {
		metrics.User.TokensPerInstance = metrics.User.TokensLast24Hours / int64(metrics.User.TotalInstances)
	}

	// Calculate share and efficiency
	if metrics.General.TokensLast24Hours > 0 {
		metrics.User.Share = float64(metrics.User.TokensLast24Hours) / float64(metrics.General.TokensLast24Hours)
		if metrics.User.Share > 0 {
			metrics.User.Efficiency = (metrics.User.TotalDailyCost / (metrics.User.Share * 100))
		}
	}

	return metrics, nil
}
