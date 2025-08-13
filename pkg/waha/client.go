package waha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries    = 3
)

type Client struct {
	baseURL     string
	sessionName string
	client      *http.Client
}

type SendTextRequest struct {
	ChatID  string `json:"chatId"`
	Text    string `json:"text"`
	Session string `json:"session"`
}

type SessionRequest struct {
	Name string `json:"name"`
}

func NewClient(baseURL, sessionName string) *Client {
	return &Client{
		baseURL:     baseURL,
		sessionName: sessionName,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) StartSession() error {
	req := SessionRequest{Name: c.sessionName}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling session request: %v", err)
	}

	resp, err := c.doWithRetry("POST", "/api/sessions", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error starting session: %v", err)
	}
	defer resp.Body.Close()

	// 409 Conflict significa que a sessão já existe, o que é OK para nós
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("unexpected status code starting session: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetScreenshot() error {
	resp, err := c.doWithRetry("GET", fmt.Sprintf("/api/screenshot?session=%s", c.sessionName), nil)
	if err != nil {
		return fmt.Errorf("error getting screenshot: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code getting screenshot: %d", resp.StatusCode)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("error creating data directory: %v", err)
	}

	// Create the file
	out, err := os.Create("data/qrcode.png")
	if err != nil {
		return fmt.Errorf("error creating qrcode file: %v", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving screenshot: %v", err)
	}

	return nil
}

func (c *Client) SendText(chatID, text string) error {
	req := SendTextRequest{
		ChatID:  chatID,
		Text:    text,
		Session: c.sessionName,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling send text request: %v", err)
	}

	resp, err := c.doWithRetry("POST", "/api/sendText", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error sending text: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error sending text (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) doWithRetry(method, path string, body io.Reader) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if body != nil {
			if seeker, ok := body.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
			}
		}

		req, err := http.NewRequest(method, c.baseURL+path, body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err = c.client.Do(req)

		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %v", err)
}
