// Package jwgl provides a shared HTTP client for communicating with
// the AUST jwglyd (教务管理) backend. It encapsulates the common
// pattern: POST form-encoded data → decode base64 business_data → return JSON bytes.
package jwgl

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)
const (
	BaseURL  = "https://jwglyd.aust.edu.cn/app-ws/ws/app-service"
	LoginURL = BaseURL + "/login"
)

// Client wraps an *http.Client with TLS verification disabled
// (matching the original Python code's verify=False).
type Client struct {
	HTTP *http.Client
}

// NewClient creates a Client that skips TLS verification and has a 30s timeout.
func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// apiResponse is the top-level shape returned by the jwglyd API.
type apiResponse struct {
	BusinessData string `json:"business_data"`
	ErrMsg       string `json:"err_msg"`
}

// Login authenticates against the jwglyd login endpoint and returns
// the decoded business_data JSON and the raw err_msg.
func (c *Client) Login(username, password string) (json.RawMessage, string, error) {
	form := url.Values{
		"user_code": {username},
		"passwd":    {base64.StdEncoding.EncodeToString([]byte(password))},
		"random":    {"89693"},
		"timestamp": {fmt.Sprintf("%d", time.Now().UnixMilli())},
	}

	body, err := c.postForm(LoginURL, form, "")
	if err != nil {
		return nil, "", fmt.Errorf("login request failed: %w", err)
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("login response unmarshal: %w", err)
	}

	if resp.BusinessData == "" {
		return nil, resp.ErrMsg, fmt.Errorf("no business_data in login response")
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.BusinessData)
	if err != nil {
		return nil, resp.ErrMsg, fmt.Errorf("base64 decode business_data: %w", err)
	}

	return json.RawMessage(decoded), resp.ErrMsg, nil
}

// LoginAndGetToken is a convenience that calls Login and extracts the token string.
func (c *Client) LoginAndGetToken(username, password string) (string, error) {
	raw, errMsg, err := c.Login(username, password)
	if err != nil {
		return "", err
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", fmt.Errorf("unmarshal token: %w", err)
	}
	if data.Token == "" {
		return "", fmt.Errorf("empty token, err_msg=%s", errMsg)
	}
	return data.Token, nil
}

// PostAPI sends a POST request to a jwglyd API endpoint with the given form
// values and token. It decodes the base64 business_data and returns the raw JSON bytes.
func (c *Client) PostAPI(endpoint string, form url.Values, token string) (json.RawMessage, error) {
	fullURL := BaseURL + endpoint

	body, err := c.postForm(fullURL, form, token)
	if err != nil {
		return nil, fmt.Errorf("api request %s failed: %w", endpoint, err)
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("api response unmarshal: %w", err)
	}

	if resp.BusinessData == "" {
		return nil, fmt.Errorf("no business_data in response from %s", endpoint)
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.BusinessData)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	if len(decoded) == 0 {
		return nil, fmt.Errorf("empty business_data from %s", endpoint)
	}

	return json.RawMessage(decoded), nil
}

// postForm performs a POST with form-encoded body and optional token header.
func (c *Client) postForm(rawURL string, form url.Values, token string) ([]byte, error) {
	req, err := http.NewRequest("POST", rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	if token != "" {
		req.Header.Set("token", token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
