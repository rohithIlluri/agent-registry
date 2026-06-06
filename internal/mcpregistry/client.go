package mcpregistry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultBase = "https://registry.modelcontextprotocol.io/v0"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Package describes a runnable distribution of an MCP server.
type Package struct {
	RegistryType    string `json:"registryType"`    // "npm", "pypi", "docker", etc.
	Name            string `json:"name,omitempty"`
	Identifier      string `json:"identifier,omitempty"` // alternate field used in some entries
	Version         string `json:"version,omitempty"`
	Transport       string `json:"transport,omitempty"` // "stdio" | "http"
	Command         string `json:"command,omitempty"`
	Args            []string `json:"args,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	RuntimeHint     string `json:"runtimeHint,omitempty"`
	AutoUpdate      bool   `json:"autoUpdate,omitempty"`
}

// Repository is the source VCS reference for an MCP server.
type Repository struct {
	URL string `json:"url"`
}

// VersionDetail holds the published semver string.
type VersionDetail struct {
	Version string `json:"version"`
}

// Server is a single entry returned by the MCP Registry API.
type Server struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Repository    Repository    `json:"repository"`
	VersionDetail VersionDetail `json:"versionDetail"`
	Packages      []Package     `json:"packages"`
}

type listResponse struct {
	Servers    []Server `json:"servers"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// Client fetches data from the official MCP Registry API.
type Client struct {
	BaseURL string
}

// NewClient returns a Client targeting the default MCP Registry endpoint.
func NewClient() *Client {
	return &Client{BaseURL: defaultBase}
}

// ListAll pages through all servers, calling progress(n) after each page with
// the running total fetched so far.
func (c *Client) ListAll(progress func(n int)) ([]Server, error) {
	base := c.BaseURL
	if base == "" {
		base = defaultBase
	}
	var all []Server
	cursor := ""
	for {
		u, _ := url.Parse(base + "/servers")
		q := u.Query()
		q.Set("limit", "100")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", u, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
		}
		var page listResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		all = append(all, page.Servers...)
		if progress != nil {
			progress(len(all))
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return all, nil
}
