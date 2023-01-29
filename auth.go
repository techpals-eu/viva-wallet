package vivawallet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// Authenticate retrieves the access token to continue making requests to Viva's API. It
// returns the full response of the API and stores the token and expiration time for
// later use.
func (c Client) Authenticate() (*TokenResponse, error) {
	uri := c.tokenEndpoint()
	auth := authBody(c.Config)

	grant := []byte("grant_type=client_credentials")
	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(grant))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to perform access token request %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("non successful response")
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	response := &TokenResponse{}
	if jsonErr := json.Unmarshal(body, response); jsonErr != nil {
		return nil, jsonErr
	}

	expiry := time.Now().Add(time.Second * time.Duration(response.ExpiresIn))
	c.SetToken(response.AccessToken, expiry)

	return response, nil
}

// AuthToken returns the token value
func (c Client) AuthToken() string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	t := c.tokenValue.value
	return t
}

// SetToken sets the token value and the expiration time of the token.
func (c Client) SetToken(tokenValue string, expires time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.tokenValue = token{
		value:   tokenValue,
		expires: expires,
	}
}

// HasAuthExpired returns true if the expiry time of the token has passed and false
// otherwise.
func (c Client) HasAuthExpired() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	expires := c.tokenValue.expires
	now := time.Now()
	return now.After(expires)
}

func authBody(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c Client) tokenEndpoint() string {
	return fmt.Sprintf("%s/%s", c.authUri(), "/connect/token")
}

func (c Client) authUri() string {
	if isDemo(c.Config) {
		return "https://demo-accounts.vivapayments.com"
	}
	return "https://accounts.vivapayments.com"
}
