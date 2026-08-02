package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// gbrainClient talks to a single gbrain instance over its MCP HTTP transport
// (gbrain serve --http). One client = one workspace's brain; the Router
// (workspace resolution, budget checks) is not yet built — Phase 0 is a
// single hardcoded shared brain, see docker-compose.yml gbrain-shared.
//
// Verified against a live instance 2026-07-31: SSE-framed JSON-RPC 2.0,
// protocol version 2025-06-18. The `query` tool takes {"query": "..."} and
// returns content[0].text as a JSON-encoded string (double-encoded — parse
// twice) holding an array of {slug, page_id, title, type, chunk_text}.
type gbrainClient struct {
	baseURL string // e.g. http://localhost:7333/mcp
	token   string // bearer token, minted via `gbrain auth create cos`
}

type gbrainPage struct {
	Slug      string `json:"slug"`
	PageID    int    `json:"page_id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	ChunkText string `json:"chunk_text"`
}

// gbrainFullPage is get_page's response shape (verified 2026-08-02): the
// full document body lives in compiled_truth, not "content" as the field
// name might suggest.
type gbrainFullPage struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	CompiledTruth string `json:"compiled_truth"`
}

type mcpRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int        `json:"id"`
	Method  string     `json:"method"`
	Params  mcpParams  `json:"params,omitempty"`
}

type mcpParams struct {
	Name      string         `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type mcpResponse struct {
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// call performs one MCP JSON-RPC call and returns the raw text of the first
// content block. The server frames every response as SSE ("event: message\n
// data: {...}\n\n") even for a single reply, so we scan for the first
// "data: " line rather than parsing as a stream.
func (c *gbrainClient) call(method string, params mcpParams) (string, error) {
	reqBody := mcpRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call gbrain: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gbrain returned status %d", resp.StatusCode)
	}

	var dataLine string
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // chunk_text can be large
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if dataLine == "" {
		return "", fmt.Errorf("no data line in gbrain response")
	}

	var mcpResp mcpResponse
	if err := json.Unmarshal([]byte(dataLine), &mcpResp); err != nil {
		return "", fmt.Errorf("parse mcp response: %w", err)
	}
	if mcpResp.Error != nil {
		return "", fmt.Errorf("gbrain error: %s", mcpResp.Error.Message)
	}
	if len(mcpResp.Result.Content) == 0 {
		return "", fmt.Errorf("empty gbrain response")
	}
	text := mcpResp.Result.Content[0].Text
	if mcpResp.Result.IsError {
		return "", fmt.Errorf("gbrain tool error: %s", text)
	}
	return text, nil
}

// query runs hybrid search (vector + keyword) and returns ranked pages.
// The chunk_text on each result is a fragment, not the full document — for
// answer synthesis use getPage on the slugs this returns, not chunk_text
// directly. See docs/03-gap-analysis.md and the 2026-08-02 grounding fix.
func (c *gbrainClient) query(question string) ([]gbrainPage, error) {
	text, err := c.call("tools/call", mcpParams{
		Name:      "query",
		Arguments: map[string]any{"query": question},
	})
	if err != nil {
		return nil, err
	}
	var pages []gbrainPage
	if err := json.Unmarshal([]byte(text), &pages); err != nil {
		return nil, fmt.Errorf("parse query results: %w", err)
	}
	return pages, nil
}

// getPage fetches a page's full body (compiled_truth) by slug. Used after
// query() to ground synthesis in complete documents instead of chunk
// fragments, which may cut off mid-section (e.g. an ADR's "Alternatives
// Considered" landing in a lower-ranked or truncated chunk).
func (c *gbrainClient) getPage(slug string) (*gbrainFullPage, error) {
	text, err := c.call("tools/call", mcpParams{
		Name:      "get_page",
		Arguments: map[string]any{"slug": slug},
	})
	if err != nil {
		return nil, err
	}
	var page gbrainFullPage
	if err := json.Unmarshal([]byte(text), &page); err != nil {
		return nil, fmt.Errorf("parse get_page result: %w", err)
	}
	return &page, nil
}
