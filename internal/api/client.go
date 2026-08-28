package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/truestandard/seo-cli/internal/config"
)

type Client struct {
	BaseURL string
	token   string
	http    *http.Client
	rpcID   atomic.Int64
}

type Query map[string]string

func New(cfg config.Config) *Client {
	return &Client{
		BaseURL: strings.TrimRight(cfg.APIURL, "/"),
		token:   cfg.Token,
		http:    &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *Client) Get(path string, query Query) (any, error) {
	return c.request(http.MethodGet, path, query, nil)
}

func (c *Client) Post(path string, payload any) (any, error) {
	return c.request(http.MethodPost, path, nil, payload)
}

func (c *Client) Patch(path string, payload any) (any, error) {
	return c.request(http.MethodPatch, path, nil, payload)
}

func (c *Client) Delete(path string) (any, error) {
	return c.request(http.MethodDelete, path, nil, nil)
}

func (c *Client) Whoami() (map[string]any, error) {
	data, err := c.Get("/api/v1/whoami", nil)
	if err != nil {
		return nil, err
	}
	identity, _ := data.(map[string]any)
	if identity == nil {
		identity = map[string]any{}
	}
	return identity, nil
}

func ProjectPath(slug, suffix string) string {
	return "/api/v1/projects/" + url.PathEscape(slug) + suffix
}

func (c *Client) request(method, path string, query Query, payload any) (any, error) {
	target := c.BaseURL + path
	if len(query) > 0 {
		values := url.Values{}
		for k, v := range query {
			if v != "" {
				values.Set(k, v)
			}
		}
		if encoded := values.Encode(); encoded != "" {
			target += "?" + encoded
		}
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, &APIError{Code: "request_error", Message: err.Error()}
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, &APIError{Code: "request_error", Message: err.Error()}
	}
	c.setHeaders(req, payload != nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Code: "network_error", Message: fmt.Sprintf("could not reach %s: %v", c.BaseURL, err)}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, ParseAPIError(resp.StatusCode, raw)
	}
	return decode(raw)
}

func (c *Client) setHeaders(req *http.Request, hasBody bool) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "seo-cli")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
}

func decode(raw []byte) (any, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return nil, nil
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, &APIError{Code: "invalid_response", Message: "the server returned a body that is not JSON"}
	}
	return data, nil
}

type rpcResponse struct {
	ID     any             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (c *Client) McpRequest(method string, params any) (json.RawMessage, error) {
	id := c.rpcID.Add(1)
	raw, status, err := c.mcpPost(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	var response rpcResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, &APIError{Code: "mcp_invalid_response", Message: "the MCP endpoint returned a body that is not JSON-RPC", Status: status}
	}
	if response.Error != nil {
		return nil, &APIError{Code: fmt.Sprintf("mcp_%d", response.Error.Code), Message: response.Error.Message, Status: status}
	}
	return response.Result, nil
}

func (c *Client) McpNotify(method string, params any) error {
	_, _, err := c.mcpPost(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
	return err
}

func (c *Client) mcpPost(payload any) ([]byte, int, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, &APIError{Code: "request_error", Message: err.Error()}
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return nil, 0, &APIError{Code: "request_error", Message: err.Error()}
	}
	c.setHeaders(req, true)
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, &APIError{Code: "network_error", Message: fmt.Sprintf("could not reach %s: %v", c.BaseURL, err)}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, ParseAPIError(resp.StatusCode, raw)
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		raw = lastSseData(raw)
	}
	return raw, resp.StatusCode, nil
}

func lastSseData(raw []byte) []byte {
	var last []byte
	for _, block := range strings.Split(string(raw), "\n\n") {
		var lines []string
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "data:") {
				lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if len(lines) > 0 {
			last = []byte(strings.Join(lines, "\n"))
		}
	}
	return last
}
