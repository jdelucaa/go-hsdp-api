//
// Copyright 2018, Andy Lo-A-Foe
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/google/go-querystring/query"
	hsdpsigner "github.com/hsdp/go-signer"
)

const (
	libraryVersion    = "0.1.0"
	defaultIAMBaseURL = "https://iam-integration.us-east.philips-healthsuite.com/"
	defaultIDMBaseURL = "https://idm-integration.us-east.philips-healthsuite.com/"
	userAgent         = "go-hsdp-iam/" + libraryVersion
)

type tokenType int

const (
	oAuthToken tokenType = iota
)

type OptionFunc func(*http.Request) error

type Config struct {
	OAuth2ClientID   string
	OAuth2Secret     string
	SharedKey        string
	SecretKey        string
	BaseIAMURL       string
	BaseIDMURL       string
	OrgAdminUsername string
	OrgAdminPassword string
	IAMURL           string
	IDMURL           string
	RootOrgID        string
}

// A Client manages communication with HSDP IAM API
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	config *Config

	signer *hsdpsigner.Signer

	baseIAMURL *url.URL
	baseIDMURL *url.URL

	// token type used to make authenticated API calls.
	tokenType tokenType

	// token used to make authenticated API calls.
	token string

	// User agent used when communicating with the HSDP IAM API.
	UserAgent string

	Organizations *OrganizationsService
	Groups        *GroupsService
	Permissions   *PermissionsService
	Roles         *RolesService
}

// NewClient returns a new HSDP IAM API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide a valid oAuth bearer token.
func NewClient(httpClient *http.Client, config *Config) (*Client, error) {
	return newClient(httpClient, config)
}

func newClient(httpClient *http.Client, config *Config) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	c := &Client{client: httpClient, config: config, UserAgent: userAgent}
	if err := c.SetBaseIAMURL(defaultIAMBaseURL); err != nil {
		panic(err)
	}
	if err := c.SetBaseIDMURL(defaultIDMBaseURL); err != nil {
		panic(err)
	}
	signer, err := hsdpsigner.New(c.config.SharedKey, c.config.SecretKey, "", nil)
	if err != nil {
		return nil, err
	}
	c.signer = signer
	c.Organizations = &OrganizationsService{client: c}
	c.Groups = &GroupsService{client: c}
	c.Permissions = &PermissionsService{client: c}
	c.Roles = &RolesService{client: c}

	return c, nil
}

func (c *Client) Login(username, password string) error {
	req, err := c.NewIAMRequest("POST", "authorize/oauth2/token", nil, nil)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	form.Add("grant_type", "password")
	req.SetBasicAuth(c.config.OAuth2ClientID, c.config.OAuth2Secret)
	req.Body = ioutil.NopCloser(strings.NewReader(form.Encode()))
	req.ContentLength = int64(len(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	resp, err := c.Do(req, &tokenResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Login failed: %d", resp.StatusCode)
	}
	c.tokenType = oAuthToken
	c.token = tokenResponse.AccessToken
	return nil
}

func (c *Client) Token() string {
	return c.token
}

// BaseURL return a copy of the baseIAMURL.
func (c *Client) BaseIAMURL() *url.URL {
	u := *c.baseIAMURL
	return &u
}

// BaseURL return a copy of the baseIAMURL.
func (c *Client) BaseIDMURL() *url.URL {
	u := *c.baseIDMURL
	return &u
}

// SetBaseIAMURL sets the base URL for API requests to a custom endpoint. urlStr
// should always be specified with a trailing slash.
func (c *Client) SetBaseIAMURL(urlStr string) error {
	// Make sure the given URL end with a slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}

	var err error
	c.baseIAMURL, err = url.Parse(urlStr)
	return err
}

// SetBaseIDMURL sets the base URL for API requests to a custom endpoint. urlStr
// should always be specified with a trailing slash.
func (c *Client) SetBaseIDMURL(urlStr string) error {
	// Make sure the given URL end with a slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}

	var err error
	c.baseIDMURL, err = url.Parse(urlStr)
	return err
}

// NewIDMRequest creates an API request. A relative URL path can be provided in
// urlStr, in which case it is resolved relative to the base URL of the Client.
// Relative URL paths should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewIDMRequest(method, path string, opt interface{}, options []OptionFunc) (*http.Request, error) {
	u := *c.baseIDMURL
	// Set the encoded opaque data
	u.Opaque = c.baseIDMURL.Path + path

	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		u.RawQuery = q.Encode()
	}

	req := &http.Request{
		Method:     method,
		URL:        &u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}

	for _, fn := range options {
		if fn == nil {
			continue
		}

		if err := fn(req); err != nil {
			return nil, err
		}
	}

	if method == "POST" || method == "PUT" {
		bodyBytes, err := json.Marshal(opt)
		if err != nil {
			return nil, err
		}
		bodyReader := bytes.NewReader(bodyBytes)

		u.RawQuery = ""
		req.Body = ioutil.NopCloser(bodyReader)
		req.ContentLength = int64(bodyReader.Len())
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	switch c.tokenType {
	case oAuthToken:
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return req, nil
}

// NewIAMRequest creates an API request. A relative URL path can be provided in
// urlStr, in which case it is resolved relative to the base URL of the Client.
// Relative URL paths should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewIAMRequest(method, path string, opt interface{}, options []OptionFunc) (*http.Request, error) {
	u := *c.baseIAMURL
	// Set the encoded opaque data
	u.Opaque = c.baseIAMURL.Path + path

	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		u.RawQuery = q.Encode()
	}

	req := &http.Request{
		Method:     method,
		URL:        &u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}

	for _, fn := range options {
		if fn == nil {
			continue
		}

		if err := fn(req); err != nil {
			return nil, err
		}
	}

	if method == "POST" || method == "PUT" {
		bodyBytes, err := json.Marshal(opt)
		if err != nil {
			return nil, err
		}
		bodyReader := bytes.NewReader(bodyBytes)

		u.RawQuery = ""
		req.Body = ioutil.NopCloser(bodyReader)
		req.ContentLength = int64(bodyReader.Len())
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	switch c.tokenType {
	case oAuthToken:
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	return req, nil
}

// Response is a HSDP IAM API response. This wraps the standard http.Response
// returned from HSDP IAM and provides convenient access to things like errors
type Response struct {
	*http.Response
	Message string
}

// newResponse creates a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	return response
}

func (c *Client) DoSigned(req *http.Request, v interface{}) (*Response, error) {
	c.signer.SignRequest(req)
	return c.Do(req, v)
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return response, err
}

type ErrorResponse struct {
	Response *http.Response
	Message  string
}

func (e *ErrorResponse) Error() string {
	path, _ := url.QueryUnescape(e.Response.Request.URL.Opaque)
	u := fmt.Sprintf("%s://%s%s", e.Response.Request.URL.Scheme, e.Response.Request.URL.Host, path)
	return fmt.Sprintf("%s %s: %d %s", e.Response.Request.Method, u, e.Response.StatusCode, e.Message)
}

// CheckResponse checks the API response for errors, and returns them if present.
func CheckResponse(r *http.Response) error {
	switch r.StatusCode {
	case 200, 201, 202, 204, 304:
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		var raw interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			errorResponse.Message = "failed to parse unknown error format"
		}

		errorResponse.Message = parseError(raw)
	}

	return errorResponse
}

func parseError(raw interface{}) string {
	switch raw := raw.(type) {
	case string:
		return raw

	case []interface{}:
		var errs []string
		for _, v := range raw {
			errs = append(errs, parseError(v))
		}
		return fmt.Sprintf("[%s]", strings.Join(errs, ", "))

	case map[string]interface{}:
		var errs []string
		for k, v := range raw {
			errs = append(errs, fmt.Sprintf("{%s: %s}", k, parseError(v)))
		}
		sort.Strings(errs)
		return strings.Join(errs, ", ")

	default:
		return fmt.Sprintf("failed to parse unexpected error type: %T", raw)
	}
}

// WithContext runs the request with the provided context
func WithContext(ctx context.Context) OptionFunc {
	return func(req *http.Request) error {
		*req = *req.WithContext(ctx)
		return nil
	}
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}
