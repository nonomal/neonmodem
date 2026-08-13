package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RequestError struct {
	StatusCode int
	Err        error
}

func (re *RequestError) Error() string {
	return re.Err.Error()
}

func (re *RequestError) Unwrap() error {
	return re.Err
}

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

type ClientConfig struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
	Logger     Logger
}

type Client struct {
	httpClient *http.Client
	endpoint   string
	token      string
	logger     Logger

	Board   BoardService
	Topics  TopicsService
	Posts   PostsService
	Session SessionService
}

func NewDefaultClientConfig(
	endpoint string,
	proxy string,
	token string,
	logger Logger,
) ClientConfig {
	var httpTransport *http.Transport = http.DefaultTransport.(*http.Transport).
		Clone()

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			if logger != nil {
				logger.Error(err)
			}
		} else {
			if logger != nil {
				logger.Debugf("setting up http proxy transport: %s\n", proxyURL.String())
			}
			httpTransport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	httpClient := &http.Client{
		Transport: httpTransport,
		Timeout:   time.Second * 10,
	}

	return ClientConfig{
		Endpoint:   endpoint,
		Token:      token,
		HTTPClient: httpClient,
		Logger:     logger,
	}
}

func NewClient(cc *ClientConfig) *Client {
	c := new(Client)
	c.logger = cc.Logger
	c.httpClient = cc.HTTPClient
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: time.Second * 10}
	}
	c.endpoint = strings.TrimRight(cc.Endpoint, "/")
	c.token = cc.Token

	c.Board = &BoardServiceHandler{client: c}
	c.Topics = &TopicsServiceHandler{client: c}
	c.Posts = &PostsServiceHandler{client: c}
	c.Session = &SessionServiceHandler{client: c}

	return c
}

func (c *Client) NewRequest(
	ctx context.Context,
	method string,
	location string,
	body interface{},
) (*http.Request, error) {
	parsedURL, err := url.Parse(c.endpoint + location)
	if err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	if body != nil {
		if err = json.NewEncoder(buffer).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		parsedURL.String(),
		buffer,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "Neon Modem Overdrive")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.token)

	return req, nil
}

type ErrorBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
}

var ErrNotAnAPI = errors.New(
	"the server did not respond with JSON, so this URL does not appear to " +
		"point at a Hyperuplink API",
)

func isJSON(res *http.Response) bool {
	mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return mediatype == "application/json" ||
		strings.HasSuffix(mediatype, "+json")
}

func (c *Client) Do(
	ctx context.Context,
	req *http.Request,
	content interface{},
) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < http.StatusOK ||
		res.StatusCode > http.StatusNoContent {
		if !isJSON(res) {
			return &RequestError{
				StatusCode: res.StatusCode,
				Err: fmt.Errorf(
					"%s %s returned status %d: %w",
					req.Method, req.URL.String(), res.StatusCode, ErrNotAnAPI,
				),
			}
		}

		var errbody ErrorBody
		if err := json.Unmarshal(body, &errbody); err == nil && errbody.Error != "" {
			msg := errbody.Error
			if len(errbody.Fields) > 0 {
				var parts []string
				for field, reason := range errbody.Fields {
					parts = append(parts, fmt.Sprintf("%s: %s", field, reason))
				}
				msg = fmt.Sprintf("%s (%s)", msg, strings.Join(parts, ", "))
			}
			return &RequestError{
				StatusCode: res.StatusCode,
				Err:        errors.New(msg),
			}
		}

		return &RequestError{
			StatusCode: res.StatusCode,
			Err:        fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body)),
		}
	}

	if content != nil {
		if !isJSON(res) {
			return &RequestError{
				StatusCode: res.StatusCode,
				Err: fmt.Errorf(
					"%s %s: %w",
					req.Method, req.URL.String(), ErrNotAnAPI,
				),
			}
		}

		if err = json.Unmarshal(body, content); err != nil {
			return &RequestError{
				StatusCode: res.StatusCode,
				Err: fmt.Errorf(
					"%s %s returned a malformed response: %w",
					req.Method, req.URL.String(), err,
				),
			}
		}
	}

	return nil
}
