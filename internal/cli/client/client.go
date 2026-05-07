package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
)

// The CLI intentionally uses the same session cookie + CSRF flow as the console
// for now. Bearer/deploy tokens will come later when API token scope, revocation,
// and CI-oriented auth are introduced together.
const DefaultTimeout = 3 * time.Second

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Options struct {
	BaseURL string
	Timeout time.Duration
}

func New(opts Options) *Client {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Register(ctx context.Context, username string, email string, password string) (string, error) {
	var res dtos.RegisterUserResponse
	err := c.post(ctx, "/api/v1/auth/register", nil, dtos.RegisterUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	}, http.StatusCreated, &res)
	if err != nil {
		return "", err
	}

	return res.ID, nil
}

func (c *Client) SignIn(ctx context.Context, username string, password string) (*Session, error) {
	var res dtos.SignInUserResponse
	cookies, err := c.postWithCookies(ctx, "/api/v1/auth/signin", nil, dtos.SignInUserRequest{
		Username: username,
		Password: password,
	}, http.StatusOK, &res)
	if err != nil {
		return nil, err
	}

	token := findCookieValue(cookies, api.SessionCookieName)
	if token == "" {
		return nil, fmt.Errorf("signin succeeded, but the server did not return a session cookie")
	}

	csrfToken := findCookieValue(cookies, api.CSRFCookieName)
	if csrfToken == "" {
		csrfToken = res.CSRFToken
	}
	if csrfToken == "" {
		return nil, fmt.Errorf("signin succeeded, but the server did not return a CSRF token")
	}

	return &Session{
		UserID:       res.ID,
		SessionToken: token,
		CSRFToken:    csrfToken,
		ServerURL:    c.baseURL,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (c *Client) CreateProject(ctx context.Context, session *Session, req dtos.CreateProjectRequest) (*dtos.CreateProjectResponse, error) {
	var res dtos.CreateProjectResponse
	err := c.post(ctx, "/api/v1/projects", session, req, http.StatusCreated, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetProject(ctx context.Context, session *Session, id string) (*dtos.CreateProjectResponse, error) {
	var res dtos.CreateProjectResponse
	err := c.get(ctx, "/api/v1/projects/"+url.PathEscape(id), session, http.StatusOK, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) UploadImage(ctx context.Context, session *Session, projectID string, imageTag string, tarballPath string) error {
	if err := requireCSRFToken(session); err != nil {
		return err
	}

	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	path := "/api/v1/projects/" + url.PathEscape(projectID) + "/images/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, file)
	if err != nil {
		return err
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", "application/x-tar")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(api.ImageTagHeader, imageTag)
	addSessionCookies(req, session)
	setCSRFHeader(req, session)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeError(resp)
	}

	return nil
}

func (c *Client) DeployProject(ctx context.Context, session *Session, projectID string) (*dtos.CreateProjectResponse, error) {
	var res dtos.CreateProjectResponse
	err := c.post(ctx, "/api/v1/projects/"+url.PathEscape(projectID)+"/deploy", session, struct{}{}, http.StatusOK, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) post(ctx context.Context, path string, session *Session, body any, wantStatus int, dest any) error {
	_, err := c.postWithCookies(ctx, path, session, body, wantStatus, dest)
	return err
}

func (c *Client) get(ctx context.Context, path string, session *Session, wantStatus int, dest any) error {
	_, err := c.getWithCookies(ctx, path, session, wantStatus, dest)
	return err
}

func (c *Client) getWithCookies(ctx context.Context, path string, session *Session, wantStatus int, dest any) ([]*http.Cookie, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	addSessionCookies(req, session)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		return nil, decodeError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return nil, err
	}

	return resp.Cookies(), nil
}

func (c *Client) postWithCookies(ctx context.Context, path string, session *Session, body any, wantStatus int, dest any) ([]*http.Cookie, error) {
	if err := requireCSRFToken(session); err != nil {
		return nil, err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if session != nil {
		addSessionCookies(req, session)
		setCSRFHeader(req, session)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		return nil, decodeError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return nil, err
	}

	return resp.Cookies(), nil
}

func requireCSRFToken(session *Session) error {
	if session == nil || session.CSRFToken != "" {
		return nil
	}

	return fmt.Errorf("session is missing a CSRF token; run `rivet signin` again")
}

func addSessionCookies(req *http.Request, session *Session) {
	if session == nil {
		return
	}

	req.AddCookie(&http.Cookie{
		Name:  api.SessionCookieName,
		Value: session.SessionToken,
	})
	if session.CSRFToken == "" {
		return
	}

	req.AddCookie(&http.Cookie{
		Name:  api.CSRFCookieName,
		Value: session.CSRFToken,
	})
}

func setCSRFHeader(req *http.Request, session *Session) {
	if session == nil || session.CSRFToken == "" {
		return
	}

	req.Header.Set(api.CSRFHeaderName, session.CSRFToken)
}

func decodeError(resp *http.Response) error {
	var apiErr dtos.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	if apiErr.Message != "" {
		return fmt.Errorf("%s", apiErr.Message)
	}
	if apiErr.Error != "" {
		return fmt.Errorf("%s", apiErr.Error)
	}

	return fmt.Errorf("request failed with status %s", resp.Status)
}

func findCookieValue(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}

	return ""
}
