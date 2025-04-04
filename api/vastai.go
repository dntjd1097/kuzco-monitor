package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// LogResponse represents the response from the log request
type LogResponse struct {
	Success         bool   `json:"success"`
	ResultURL       string `json:"result_url"`
	TempDownloadURL string `json:"temp_download_url"`
	Message         string `json:"msg"`
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
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	startOfDay := yesterday // yesterday 00:00:00
	endOfDay := today       // today 00:00:00

	// Create filter JSON
	selectFilters := fmt.Sprintf(`{"when":{"gte":%d,"lte":%d},"service":{"in":["paypal","paypal_manual","crypto.com","coinbase","stripe_connect","stripe_payments","stripe","wise_manual","instance_prepay","transfer"]},"type":{"in":["charge"]},"amount_cents":{}}`,
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
		quantity, _ := strconv.ParseFloat(charge.Quantity, 64)
		rate, _ := strconv.ParseFloat(charge.Rate, 64)
		totalCost += quantity * rate
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

// RequestInstanceLogs requests logs for a specific instance
func (c *VastaiClient) RequestInstanceLogs(instanceID int) (*LogResponse, error) {
	fullURL := fmt.Sprintf("%sinstances/request_logs/%d", c.baseURL, instanceID)

	req, err := http.NewRequest("PUT", fullURL, strings.NewReader("{}"))
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
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var logResp LogResponse
	if err := json.Unmarshal(body, &logResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &logResp, nil
}

// CheckInstanceLogs checks if the instance logs contain the heartbeat timeout error
// Returns true if heartbeat timeouts are detected continuously for 3 minutes
func (c *VastaiClient) CheckInstanceLogs(url string) (bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to get logs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read logs: %w", err)
	}

	// Split logs into lines and check for timeout patterns
	lines := strings.Split(string(body), "\n")

	// Maps to track timeouts in each minute of the last 3 minutes
	timeoutDetected := make(map[int]bool)
	now := time.Now()

	// Initialize the last 3 minutes as not having timeouts
	for i := 0; i < 3; i++ {
		timeoutDetected[i] = false
	}

	for _, line := range lines {
		if strings.Contains(line, "Failed to send heartbeat: TimeoutError: timeout") {
			// Parse the timestamp from the log line
			if timestamp, err := parseLogTimestamp(line); err == nil {
				// Calculate how many minutes ago this timeout occurred
				minutesAgo := int(now.Sub(timestamp).Minutes())

				// Only consider timeouts from the last 3 minutes
				if minutesAgo >= 0 && minutesAgo < 3 {
					timeoutDetected[minutesAgo] = true
					log.Printf("Detected heartbeat timeout from %d minutes ago: %s",
						minutesAgo, timestamp.Format(time.RFC3339))
				}
			}
		}
	}

	// Check if we have timeouts in all 3 consecutive minutes
	consecutiveTimeouts := timeoutDetected[0] && timeoutDetected[1] && timeoutDetected[2]

	if consecutiveTimeouts {
		log.Printf("Detected heartbeat timeouts continuously for the last 3 minutes")
		return true, nil
	}

	return false, nil
}

// parseLogTimestamp parses the timestamp from a log line
func parseLogTimestamp(line string) (time.Time, error) {
	// Example log line format:
	// [oafdhq0JRIp1jpxrKhGwW|startWorkerEventLoop]: 2025-02-26T18:07:58.305Z
	parts := strings.Split(line, ": ")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid log line format")
	}

	// Extract the timestamp part
	timestampStr := strings.Split(parts[1], " ")[0]
	return time.Parse(time.RFC3339Nano, timestampStr)
}

// RebootInstance reboots a specific instance
func (c *VastaiClient) RebootInstance(instanceID int) error {
	fullURL := fmt.Sprintf("%sinstances/reboot/%d/", c.baseURL, instanceID)

	req, err := http.NewRequest("PUT", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reboot failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// MonitorAndRebootInstances monitors all instances and reboots them if they have heartbeat timeout
func (c *VastaiClient) MonitorAndRebootInstances(sendAlert func(string, string) error) error {
	return c.StartContinuousMonitoring(sendAlert, false, nil)
}

// StartContinuousMonitoring continuously monitors instances for heartbeat timeouts and reboots them if necessary
// Set stopOnFirstExecution to true to run only once (useful for testing)
func (c *VastaiClient) StartContinuousMonitoring(
	sendAlert func(string, string) error,
	stopOnFirstExecution bool,
	stopChan <-chan struct{},
) error {
	log.Printf("Starting continuous instance monitoring...")
	// Check every minute for timeout issues over a 3-minute period
	monitoringInterval := 1 * time.Minute

	// Function to check and reboot instances
	checkAndReboot := func() error {
		// Check General.RunningInstanceCount first
		generalMetrics := GlobalHourlyStats.GetStats()
		if generalMetrics.TotalInstances.Current == 0 {
			log.Printf("General.RunningInstanceCount is 0, skipping monitoring")
			return nil
		}

		instances, err := c.GetInstances()
		if err != nil {
			return fmt.Errorf("failed to get instances: %w", err)
		}

		for _, instance := range instances {
			// Request logs for the instance
			log.Printf("Requesting logs for instance %d (status: %s)...", instance.ID, instance.ActualStatus)
			logResp, err := c.RequestInstanceLogs(instance.ID)
			if err != nil {
				log.Printf("Failed to request logs for instance %d: %v", instance.ID, err)
				continue
			}

			// Wait a few seconds for the logs to be available
			time.Sleep(5 * time.Second)
			// Check if logs contain heartbeat timeout
			hasTimeout, err := c.CheckInstanceLogs(logResp.TempDownloadURL)
			if err != nil {
				log.Printf("Failed to check logs for instance %d: %v", instance.ID, err)
				continue
			}

			if hasTimeout {
				// Double check General.RunningInstanceCount before rebooting
				currentMetrics := GlobalHourlyStats.GetStats()
				if currentMetrics.TotalInstances.Current == 0 {
					log.Printf("General.RunningInstanceCount is 0, skipping reboot for instance %d", instance.ID)
					if sendAlert != nil {
						message := fmt.Sprintf("⚠️ Reboot Skipped\nInstance ID: %d\n사유: General.RunningInstanceCount가 0입니다.", instance.ID)
						if err := sendAlert(message, "error"); err != nil {
							log.Printf("Failed to send skip alert: %v", err)
						}
					}
					continue
				}

				log.Printf("Heartbeat timeout detected continuously for 3 minutes on instance %d, rebooting... (General.RunningInstanceCount: %d)",
					instance.ID, currentMetrics.TotalInstances.Current)

				if err := c.RebootInstance(instance.ID); err != nil {
					log.Printf("Failed to reboot instance %d: %v", instance.ID, err)
					if sendAlert != nil {
						message := fmt.Sprintf("⚠️ Instance Reboot Failed\nInstance ID: %d\nError: %v", instance.ID, err)
						if err := sendAlert(message, "error"); err != nil {
							log.Printf("Failed to send reboot error alert: %v", err)
						}
					}
					continue
				}
				log.Printf("Successfully rebooted instance %d", instance.ID)
				if sendAlert != nil {
					message := fmt.Sprintf("✅ Instance Reboot Success\nInstance ID: %d가 성공적으로 재시작되었습니다.\nGeneral.RunningInstanceCount: %d",
						instance.ID, currentMetrics.TotalInstances.Current)
					if err := sendAlert(message, "status"); err != nil {
						log.Printf("Failed to send reboot success alert: %v", err)
					}
				}
			}
		}

		return nil
	}

	// If we only want to run once (for testing)
	if stopOnFirstExecution {
		return checkAndReboot()
	}

	// Start continuous monitoring in a goroutine
	go func() {
		for {
			if err := checkAndReboot(); err != nil {
				log.Printf("Error during instance monitoring: %v", err)
			}

			// Check if we should stop monitoring
			if stopChan != nil {
				select {
				case <-stopChan:
					log.Printf("Stopping instance monitoring...")
					return
				case <-time.After(monitoringInterval):
					// Continue to next iteration
				}
			} else {
				// No stop channel provided, just sleep
				time.Sleep(monitoringInterval)
			}
		}
	}()

	return nil
}

// GetInstances returns all instances
func (c *VastaiClient) GetInstances() ([]struct {
	ActualStatus string `json:"actual_status"`
	ID           int    `json:"id"`
}, error) {
	fullURL := c.baseURL + "instances/"

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
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var vastaiResp VastaiInstancesResponse
	if err := json.Unmarshal(body, &vastaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return vastaiResp.Instances, nil
}
