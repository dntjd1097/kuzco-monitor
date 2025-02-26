package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// VastaiClient handles Vast.ai API interactions
type VastaiClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// VastaiCharge represents a billing charge from Vast.ai
type VastaiCharge struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Timestamp   int64  `json:"timestamp"`
	Quantity    string `json:"quantity"`
	Rate        string `json:"rate"`
	Amount      string `json:"amount"`
	InstanceID  int    `json:"instance_id"`
}

// VastaiErrorResponse represents an error response from Vast.ai API
type VastaiErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"msg"`
}

// VastaiInstancesResponse represents the response from Vast.ai instances API
type VastaiInstancesResponse struct {
	InstancesFound int `json:"instances_found"`
	Instances      []struct {
		ActualStatus string `json:"actual_status"`
		ID           int    `json:"id"`
	} `json:"instances"`
}

// VastaiCreditResponse represents the credit information from Vast.ai
type VastaiCreditResponse struct {
	Current struct {
		Charges    float64 `json:"charges"`
		ServiceFee float64 `json:"service_fee"`
		Total      float64 `json:"total"`
		Credit     float64 `json:"credit"`
	} `json:"current"`
}

// VastaiCredit represents the credit information with timestamp
type VastaiCredit struct {
	Credit    float64 `json:"credit"`
	Timestamp string  `json:"timestamp"`
}

// NewVastaiClient creates a new Vast.ai client
func NewVastaiClient(token string) *VastaiClient {
	return &VastaiClient{
		baseURL:    VastaiAPI,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		token:      token,
	}
}

// GetDailyCost retrieves the daily cost from Vast.ai for the previous day (UTC)
func (c *VastaiClient) GetDailyCost() (float64, error) {
	// Calculate yesterday's UTC time start and end timestamps
	now := time.Now().UTC()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	startOfDay := yesterday
	endOfDay := yesterday.Add(24 * time.Hour).Add(-1 * time.Second)

	// Create filter JSON
	selectFilters := fmt.Sprintf(`{"when":{"gte":%d,"lte":%d},"service":{"in":["instance_prepay"]},"type":{"in":["charge"]},"amount_cents":{}}`,
		startOfDay.Unix(), endOfDay.Unix())

	// Construct URL
	fullURL := fmt.Sprintf("%sinvoices?select_filters=%s", c.baseURL, url.QueryEscape(selectFilters))

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error response
	if resp.StatusCode != http.StatusOK {
		var errResp VastaiErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil {
			return 0, fmt.Errorf("API error (HTTP %d): %s - %s",
				resp.StatusCode, errResp.Error, errResp.Message)
		}
		return 0, fmt.Errorf("request failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	// Parse response
	var charges []VastaiCharge
	if err := json.Unmarshal(body, &charges); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate total cost
	var totalCost float64
	for _, charge := range charges {
		amount, _ := strconv.ParseFloat(charge.Amount, 64)
		totalCost += amount
	}

	return totalCost, nil
}

// GetInstanceCount retrieves the number of instances from Vast.ai
func (c *VastaiClient) GetInstanceCount() (int, error) {
	fullURL := c.baseURL + "instances/"

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var vastaiResp VastaiInstancesResponse
	if err := json.Unmarshal(body, &vastaiResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return vastaiResp.InstancesFound, nil
}

// GetCredit retrieves the current credit balance from Vast.ai
func (c *VastaiClient) GetCredit() (*VastaiCredit, error) {
	fullURL := c.baseURL + "users/current/invoices/"

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp VastaiErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil {
			return nil, fmt.Errorf("API error (HTTP %d): %s - %s",
				resp.StatusCode, errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("request failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	var creditResp VastaiCreditResponse
	if err := json.Unmarshal(body, &creditResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &VastaiCredit{
		Credit:    creditResp.Current.Credit,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}
