package main

import (
	"encoding/json"
	"fmt"
)

type Service struct {
	client *Client
	logger func(string)
}

func NewService(token string, logger func(string)) *Service {
	return &Service{
		client: NewClient(token, logger),
		logger: logger,
	}
}

func (s *Service) Login(email, password string) (string, error) {
	if s.logger != nil {
		s.logger("Attempting login...")
	}

	loginReq := LoginRequest{}
	loginReq.JSON.Email = email
	loginReq.JSON.Password = password
	loginReq.JSON.TwoFactorToken = nil
	loginReq.Meta.Values.TwoFactorToken = []string{"undefined"}

	body := map[string]LoginRequest{"0": loginReq}
	respBody, err := s.client.doRequest("POST", "user.login?batch=1", body)
	if err != nil {
		if s.logger != nil {
			s.logger(fmt.Sprintf("Login failed: %v", err))
		}
		return "", err
	}

	var loginResp []LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		if s.logger != nil {
			s.logger(fmt.Sprintf("Failed to parse login response: %v", err))
		}
		return "", fmt.Errorf("error parsing login response: %w", err)
	}

	if len(loginResp) == 0 {
		if s.logger != nil {
			s.logger("Empty login response received")
		}
		return "", fmt.Errorf("empty login response")
	}

	if s.logger != nil {
		s.logger("Login successful")
	}
	return loginResp[0].Result.Data.JSON.Token, nil
}

func (s *Service) GetOnlineWorkers() (int, error) {
	if s.logger != nil {
		s.logger("Fetching online workers count...")
	}

	respBody, err := s.client.doRequest("GET", "instance.countOnline?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}", nil)
	if err != nil {
		if s.logger != nil {
			s.logger(fmt.Sprintf("Failed to get online workers: %v", err))
		}
		return 0, err
	}

	var resp []CountOnlineResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		if s.logger != nil {
			s.logger(fmt.Sprintf("Failed to parse online workers response: %v", err))
		}
		return 0, fmt.Errorf("error parsing response: %w", err)
	}

	if len(resp) == 0 {
		if s.logger != nil {
			s.logger("Empty response received for online workers")
		}
		return 0, fmt.Errorf("empty response")
	}

	if s.logger != nil {
		s.logger(fmt.Sprintf("Successfully retrieved online workers count: %d", resp[0].Result.Data.JSON.Count))
	}
	return resp[0].Result.Data.JSON.Count, nil
}
