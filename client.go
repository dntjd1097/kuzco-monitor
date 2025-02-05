package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const baseURL = "https://relay.kuzco.xyz/api/trpc"

type Client struct {
	httpClient *http.Client
	token      string
	logger     func(string)
}

func NewClient(token string, logger func(string)) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      token,
		logger:     logger,
	}
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	if c.logger != nil {
		c.logger(fmt.Sprintf("🌐 Making %s request to: %s", method, endpoint))
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			if c.logger != nil {
				c.logger(fmt.Sprintf("❌ Error marshaling request body: %v", err))
			}
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
		if c.logger != nil {
			c.logger(fmt.Sprintf("📤 Request body: %s", string(jsonData)))
		}
	}

	url := endpoint
	if !strings.HasPrefix(endpoint, "http") {
		url = fmt.Sprintf("%s/%s", baseURL, endpoint)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		if c.logger != nil {
			c.logger(fmt.Sprintf("❌ Error creating request: %v", err))
		}
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger(fmt.Sprintf("❌ Error making request: %v", err))
		}
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger(fmt.Sprintf("❌ Error reading response: %v", err))
		}
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if c.logger != nil {
		c.logger(fmt.Sprintf("📥 Response: %s", string(respBody)))
	}

	return respBody, nil
}
