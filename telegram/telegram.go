package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		MessageThreadID int `json:"message_thread_id"`
	} `json:"message"`
}

// Client represents a Telegram bot client
type Client struct {
	Token  string
	ChatID string
}

// NewClient creates a new Telegram client
func NewClient(token, chatID string) *Client {
	return &Client{
		Token:  token,
		ChatID: chatID,
	}
}

// SendMessage sends a message to Telegram using the specified thread
func (c *Client) SendMessage(threadID int, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)

	params := url.Values{}
	params.Add("chat_id", c.ChatID)
	params.Add("text", message)
	params.Add("parse_mode", "Markdown")
	if threadID > 0 {
		params.Add("message_thread_id", fmt.Sprintf("%d", threadID))
	}

	resp, err := http.PostForm(apiURL, params)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

// GetUpdates retrieves updates from Telegram bot API
func (c *Client) GetUpdates(offset int) ([]Update, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", c.Token)
	params := url.Values{}
	params.Add("offset", fmt.Sprintf("%d", offset))
	params.Add("timeout", "30")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Ok     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Result, nil
}
