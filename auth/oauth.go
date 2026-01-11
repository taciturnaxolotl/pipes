package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kierank/pipes/config"
	"github.com/kierank/pipes/store"
)

type OAuthClient struct {
	cfg    *config.Config
	db     *store.DB
	states map[string]*PKCEState // In-memory for MVP; use Redis in production
}

type PKCEState struct {
	CodeVerifier string
	RedirectURI  string
	CreatedAt    time.Time
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type UserInfo struct {
	Sub      string `json:"sub"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Photo    string `json:"picture,omitempty"`
	URL      string `json:"profile,omitempty"`
}

func NewOAuthClient(cfg *config.Config, db *store.DB) *OAuthClient {
	return &OAuthClient{
		cfg:    cfg,
		db:     db,
		states: make(map[string]*PKCEState),
	}
}

func (c *OAuthClient) GetAuthorizationURL() (string, error) {
	state, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	codeVerifier, err := generateRandomString(64)
	if err != nil {
		return "", fmt.Errorf("generate code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Store PKCE state (in-memory for now)
	c.states[state] = &PKCEState{
		CodeVerifier: codeVerifier,
		RedirectURI:  c.cfg.OAuthCallbackURL,
		CreatedAt:    time.Now(),
	}

	// Clean up old states (older than 10 minutes)
	go c.cleanupStates()

	authURL := fmt.Sprintf("%s/auth/authorize?"+
		"response_type=code&"+
		"client_id=%s&"+
		"redirect_uri=%s&"+
		"state=%s&"+
		"code_challenge=%s&"+
		"code_challenge_method=S256&"+
		"scope=profile%%20email",
		c.cfg.IndikoURL,
		url.QueryEscape(c.cfg.IndikoClientID),
		url.QueryEscape(c.cfg.OAuthCallbackURL),
		state,
		codeChallenge,
	)

	return authURL, nil
}

func (c *OAuthClient) HandleCallback(state, code string) (*store.User, *store.Session, error) {
	// Verify state
	pkceState, ok := c.states[state]
	if !ok {
		return nil, nil, fmt.Errorf("invalid state")
	}

	delete(c.states, state)

	// Exchange code for token
	tokenResp, err := c.exchangeCode(code, pkceState.CodeVerifier, pkceState.RedirectURI)
	if err != nil {
		return nil, nil, fmt.Errorf("exchange code: %w", err)
	}

	// Fetch user info
	userInfo, err := c.fetchUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch user info: %w", err)
	}

	// Create or update user
	user, err := c.db.GetUserByIndikoSub(userInfo.Sub)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		user, err = c.db.CreateUser(userInfo.Sub, userInfo.Username, userInfo.Name, userInfo.Email, userInfo.Photo, userInfo.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
	} else {
		// Update user info
		user.Username = userInfo.Username
		user.Name = userInfo.Name
		user.Email = userInfo.Email
		user.Photo = userInfo.Photo
		user.URL = userInfo.URL
		if err := c.db.UpdateUser(user); err != nil {
			return nil, nil, fmt.Errorf("update user: %w", err)
		}
	}

	// Create session
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days
	session, err := c.db.CreateSession(user.ID, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

func (c *OAuthClient) exchangeCode(code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", c.cfg.IndikoClientID)
	data.Set("code_verifier", codeVerifier)

	if c.cfg.IndikoClientSecret != "" {
		data.Set("client_secret", c.cfg.IndikoClientSecret)
	}

	tokenURL := fmt.Sprintf("%s/auth/token", c.cfg.IndikoURL)

	// Create request with explicit headers
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed (URL: %s): %s - %s", tokenURL, resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &tokenResp, nil
}

func (c *OAuthClient) fetchUserInfo(accessToken string) (*UserInfo, error) {
	userInfoURL := fmt.Sprintf("%s/userinfo", c.cfg.IndikoURL)

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed: %s - %s", resp.Status, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &userInfo, nil
}

func (c *OAuthClient) cleanupStates() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for state, pkceState := range c.states {
		if pkceState.CreatedAt.Before(cutoff) {
			delete(c.states, state)
		}
	}
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
