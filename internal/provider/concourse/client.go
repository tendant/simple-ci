package concourse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lei/simple-ci/pkg/logger"
)

// Client handles HTTP communication with Concourse ATC API
type Client struct {
	baseURL      string
	tokenManager *TokenManager
	httpClient   *http.Client
	logger       *logger.Logger
}

// Build represents a Concourse build
type Build struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"` // pending, started, succeeded, failed, errored, aborted
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
	CreateTime int64  `json:"create_time"`
}

// NewClient creates a new Concourse API client
func NewClient(baseURL string, tokenManager *TokenManager, log *logger.Logger) *Client {
	return &Client{
		baseURL:      baseURL,
		tokenManager: tokenManager,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       log,
	}
}

// doRequest performs an authenticated HTTP request with automatic token refresh
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	c.logger.Debug("provider: http request",
		"method", method,
		"path", path)

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		c.logger.Error("provider: failed to get token", "error", err)
		return nil, fmt.Errorf("get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		c.logger.Error("provider: failed to create request", "error", err)
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("provider: http request failed",
			"method", method,
			"path", path,
			"error", err)
		return nil, err
	}

	c.logger.Debug("provider: http response",
		"method", method,
		"path", path,
		"status", resp.StatusCode)

	// If 401, invalidate token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		c.logger.Info("provider: received 401, invalidating token and retrying",
			"method", method,
			"path", path)
		c.tokenManager.InvalidateToken()

		token, err := c.tokenManager.GetToken(ctx)
		if err != nil {
			c.logger.Error("provider: failed to refresh token", "error", err)
			return nil, fmt.Errorf("refresh token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			c.logger.Error("provider: retry request failed",
				"method", method,
				"path", path,
				"error", err)
		} else {
			c.logger.Info("provider: retry request succeeded",
				"method", method,
				"path", path,
				"status", resp.StatusCode)
		}
	}

	return resp, err
}

// CreateBuild triggers a new build for a job
func (c *Client) CreateBuild(ctx context.Context, team, pipeline, job string, params map[string]interface{}) (*Build, error) {
	path := fmt.Sprintf("/api/v1/teams/%s/pipelines/%s/jobs/%s/builds", team, pipeline, job)

	var body io.Reader
	if len(params) > 0 {
		jsonBody, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, fmt.Errorf("decode build response: %w", err)
	}

	return &build, nil
}

// GetBuild retrieves build information by ID
func (c *Client) GetBuild(ctx context.Context, buildID int) (*Build, error) {
	path := fmt.Sprintf("/api/v1/builds/%d", buildID)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, parseError(resp)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, fmt.Errorf("decode build: %w", err)
	}

	return &build, nil
}

// AbortBuild cancels a running build
func (c *Client) AbortBuild(ctx context.Context, buildID int) error {
	path := fmt.Sprintf("/api/v1/builds/%d/abort", buildID)

	resp, err := c.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	return nil
}

// StreamBuildEvents streams build events as Server-Sent Events
func (c *Client) StreamBuildEvents(ctx context.Context, buildID int, writer io.Writer) error {
	path := fmt.Sprintf("/api/v1/builds/%d/events", buildID)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	// Stream response body to writer
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Transform and write event
		event, err := parseConcourseEvent(line)
		if err != nil {
			continue // Skip malformed events
		}

		if event != "" {
			if _, err := writer.Write([]byte(event)); err != nil {
				return err
			}

			// Flush if writer supports it
			if f, ok := writer.(http.Flusher); ok {
				f.Flush()
			}
		}
	}

	return scanner.Err()
}
