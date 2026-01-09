package concourse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenManager handles Concourse authentication and token caching
type TokenManager struct {
	baseURL     string
	team        string
	username    string
	password    string
	bearerToken string // Optional: pre-configured token

	mu            sync.RWMutex
	token         string
	tokenExpiry   time.Time
	refreshMargin time.Duration
}

// NewTokenManager creates a new token manager
func NewTokenManager(baseURL, team, username, password, bearerToken string, refreshMargin time.Duration) *TokenManager {
	tm := &TokenManager{
		baseURL:       baseURL,
		team:          team,
		username:      username,
		password:      password,
		bearerToken:   bearerToken,
		refreshMargin: refreshMargin,
	}

	// If bearer token is provided, use it and set expiry far in future
	if bearerToken != "" {
		tm.token = bearerToken
		tm.tokenExpiry = time.Now().Add(365 * 24 * time.Hour) // 1 year (won't auto-refresh)
	}

	return tm
}

// TokenResponse represents the response from Concourse token endpoint
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetToken returns a valid token, refreshing if necessary
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if tm.token != "" && time.Now().Before(tm.tokenExpiry.Add(-tm.refreshMargin)) {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Need to refresh
	return tm.refreshToken(ctx)
}

// InvalidateToken forces token refresh on next GetToken call
func (tm *TokenManager) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = ""
	tm.tokenExpiry = time.Time{}
}

// refreshToken fetches a new token from Concourse
func (tm *TokenManager) refreshToken(ctx context.Context) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring write lock
	if tm.token != "" && time.Now().Before(tm.tokenExpiry.Add(-tm.refreshMargin)) {
		return tm.token, nil
	}

	// Fetch new token from Concourse
	tokenResp, err := tm.fetchTokenFromConcourse(ctx)
	if err != nil {
		return "", err
	}

	tm.token = tokenResp.AccessToken
	tm.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return tm.token, nil
}

// fetchTokenFromConcourse makes the actual HTTP request to get a token
func (tm *TokenManager) fetchTokenFromConcourse(ctx context.Context) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/sky/issuer/token", tm.baseURL)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", tm.username)
	data.Set("password", tm.password)
	data.Set("scope", "openid")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Concourse requires basic auth with fly:Zmx5
	req.SetBasicAuth("fly", "Zmx5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token fetch failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	return &tokenResp, nil
}
