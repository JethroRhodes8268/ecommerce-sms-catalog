package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    envelopeError   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func NewClient(baseURL, key string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, key: key, http: httpClient}
}

// CreateSignature calls sms.signature.create with a caller-approved payload.
func (c *Client) CreateSignature(ctx context.Context, payload json.RawMessage, idempotencyKey string) (json.RawMessage, error) {
	return c.post(ctx, "/v1/sms/signature/create", payload, idempotencyKey)
}

// CreateTemplate calls sms.template.create with a caller-approved payload.
func (c *Client) CreateTemplate(ctx context.Context, payload json.RawMessage, idempotencyKey string) (json.RawMessage, error) {
	return c.post(ctx, "/v1/sms/template/create", payload, idempotencyKey)
}

func (c *Client) post(ctx context.Context, path string, payload json.RawMessage, idempotencyKey string) (json.RawMessage, error) {
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		var reply envelope
		if err := json.Unmarshal(body, &reply); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests {
			if err := wait(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return nil, err
			}
			continue
		}
		if !reply.OK {
			message := reply.Error.Message
			if message == "" {
				message = reply.Error.Hint
			}
			return nil, &Error{Code: reply.Error.Code, Message: message, Status: res.StatusCode}
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("remote status %d", res.StatusCode)
		}
		return reply.Data, nil
	}
	return nil, fmt.Errorf("retry limit reached")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
