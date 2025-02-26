package api

import (
	"encoding/json"
	"fmt"
	"time"
)

type LoginRequest struct {
	JSON struct {
		Email          string      `json:"email"`
		Password       string      `json:"password"`
		TwoFactorToken interface{} `json:"twoFactorToken"`
	} `json:"json"`
	Meta struct {
		Values struct {
			TwoFactorToken []string `json:"twoFactorToken"`
		} `json:"values"`
	} `json:"meta"`
}

type User struct {
	ID                                string        `json:"_id"`
	Email                             string        `json:"email"`
	FirstName                         string        `json:"firstName"`
	LastName                          string        `json:"lastName"`
	EmailVerifiedAt                   time.Time     `json:"emailVerifiedAt"`
	EmailVerificationTokenRequestedAt interface{}   `json:"emailVerificationTokenRequestedAt"`
	EmailVerificationRequired         bool          `json:"emailVerificationRequired"`
	Wallets                           []interface{} `json:"wallets"`
	IsDisabled                        bool          `json:"isDisabled"`
	IsBanned                          bool          `json:"isBanned"`
	Username                          string        `json:"username"`
	UserTeamType                      string        `json:"userTeamType"`
	CreatedAt                         time.Time     `json:"createdAt"`
	UpdatedAt                         time.Time     `json:"updatedAt"`
}

func (c *Client) Login(email, password string) (string, string, error) {
	loginReq := LoginRequest{
		JSON: struct {
			Email          string      `json:"email"`
			Password       string      `json:"password"`
			TwoFactorToken interface{} `json:"twoFactorToken"`
		}{
			Email:          email,
			Password:       password,
			TwoFactorToken: nil,
		},
		Meta: struct {
			Values struct {
				TwoFactorToken []string `json:"twoFactorToken"`
			} `json:"values"`
		}{
			Values: struct {
				TwoFactorToken []string `json:"twoFactorToken"`
			}{
				TwoFactorToken: []string{"undefined"},
			},
		},
	}

	body := map[string]LoginRequest{"0": loginReq}

	respBody, err := c.DoRequest("POST", EndpointUserLogin+"?batch=1", body, nil)
	if err != nil {
		return "", "", err
	}

	var loginResp []struct {
		Result struct {
			Data struct {
				JSON struct {
					Token string `json:"token"`
					User  User   `json:"user"`
				} `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return "", "", fmt.Errorf("error parsing response: %w", err)
	}

	if len(loginResp) == 0 {
		return "", "", fmt.Errorf("empty response received")
	}
	if loginResp[0].Result.Data.JSON.Token == "" {
		return "", "", fmt.Errorf("failed to login account : %s", email)
	}

	return loginResp[0].Result.Data.JSON.Token, loginResp[0].Result.Data.JSON.User.ID, nil
}
