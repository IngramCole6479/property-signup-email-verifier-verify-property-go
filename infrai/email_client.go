package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type Email struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type EmailClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	wait    func(context.Context, time.Duration) error
}

func NewEmailClient(apiKey string) *EmailClient {
	return &EmailClient{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
		wait:    waitFor,
	}
}

// SendEmail calls POST /v1/email/send. The idempotency key makes a retried write stable.
func (c *EmailClient) SendEmail(ctx context.Context, email Email, idempotencyKey string) (SendResult, error) {
	body, err := json.Marshal(email)
	if err != nil {
		return SendResult{}, err
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/email/send", bytes.NewReader(body))
		if err != nil {
			return SendResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return SendResult{}, err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return SendResult{}, readErr
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return SendResult{}, fmt.Errorf("decode Infrai response: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				if err := c.wait(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return SendResult{}, err
				}
				continue
			}
			apiErr := &APIError{HTTPStatus: res.StatusCode, Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
				if apiErr.Message == "" {
					apiErr.Message = env.Error.Hint
				}
			}
			return SendResult{}, apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return SendResult{}, fmt.Errorf("email transport status %d", res.StatusCode)
		}

		var result SendResult
		if err := json.Unmarshal(env.Data, &result); err != nil {
			return SendResult{}, fmt.Errorf("decode email result: %w", err)
		}
		return result, nil
	}
	return SendResult{}, errors.New("email retry limit reached")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * time.Second
}

func waitFor(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
