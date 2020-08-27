package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap/zapcore"
)

const (
	GitHubBaseURL = "https://api.github.com"

	userInfoPath = "user"
	userOrgsPath = "user/orgs"
)

type ErrorResponse struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentationURL"`
}

type Client interface {
	GetUserInfo(string) (*UserInfo, error)
	GetOrgs(string) (Organizations, error)
}

type client struct {
	client  *http.Client
	baseURL string
}

var _ = (*client)(nil)

type Option func(c *client)

func WithBaseURL(url string) Option {
	return func(c *client) {
		c.baseURL = url
	}
}

func NewClient(opts ...Option) Client {
	c := &client{}
	for _, opt := range opts {
		opt(c)
	}

	if c.baseURL == "" {
		c.baseURL = GitHubBaseURL
	}

	c.client = http.DefaultClient
	return c
}

type Organization struct {
	Login string `json:"login,omitempty"`
}

type Organizations []Organization

func (orgs Organizations) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, org := range orgs {
		encoder.AppendString(org.Login)
	}

	return nil
}

type UserInfo struct {
	Login string `json:"login,omitempty"`
}

func (c client) GetUserInfo(token string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/"+userInfoPath, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req, token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := decodeError(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d with message %s", resp.StatusCode, err.Message)
	}

	var u UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}

	return &u, nil
}

func (c client) GetOrgs(token string) (Organizations, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/"+userOrgsPath, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req, token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := decodeError(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d with message %s", resp.StatusCode, err.Message)
	}

	var orgs Organizations
	json.NewDecoder(resp.Body).Decode(&orgs)
	return orgs, nil
}

func decodeError(r io.Reader) *ErrorResponse {
	resp := &ErrorResponse{}
	json.NewDecoder(r).Decode(resp)
	return resp
}

func (c client) addHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/json")
}

func (orgs Organizations) LoggedInto(org string) bool {
	for _, o := range orgs {
		if o.Login == org {
			return true
		}
	}

	return false
}
