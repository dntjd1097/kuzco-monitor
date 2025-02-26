package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	alertState AlertState
}

func NewClient() *Client {
	return &Client{
		baseURL:    KuzcoAPI, // Kuzco API URL 수정
		httpClient: &http.Client{},
	}
}

// SetBaseURL allows changing the base URL (useful for testing)
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *Client) SetToken(token string) {
	c.token = token
}

// APIError는 API 호출 시 발생하는 에러를 나타내는 구조체입니다
type APIError struct {
	StatusCode int
	RawBody    []byte
	Response   *VastaiErrorResponse
}

func (e *APIError) Error() string {
	if e.Response != nil {
		return fmt.Sprintf("API error (HTTP %d): %s - %s",
			e.StatusCode, e.Response.Error, e.Response.Message)
	}
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, string(e.RawBody))
}

// DoRequest sends an HTTP request and returns the response
func (c *Client) DoRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// URL 처리: path가 전체 URL인 경우 그대로 사용, 아닌 경우 baseURL과 결합
	var url string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		url = path
	} else {
		url = c.baseURL + path
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add default headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 에러 상태 코드 처리 (400, 401, 403)
	if resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden {

		apiError := &APIError{
			StatusCode: resp.StatusCode,
			RawBody:    respBody,
		}

		// API 에러 응답 파싱 시도
		var errResp VastaiErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
			apiError.Response = &errResp
		}

		return nil, apiError
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
