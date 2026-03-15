package php

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the PHP internal dialogue API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// VendorRecord is the shape of each entry returned by PHP's pending endpoint.
type VendorRecord struct {
	ID                        int64             `json:"id"`
	UUID                      string            `json:"uuid"`
	ServiceType               string            `json:"service_type"`
	Criminality               float64           `json:"criminality"`
	DialogueGenerationVersion int               `json:"dialogue_generation_version"`
	Profile                   VendorProfileData `json:"profile"`
}

// VendorProfileData is the nested profile block in the pending response.
type VendorProfileData struct {
	Archetype   string             `json:"archetype"`
	Personality map[string]float64 `json:"personality"`
	MarkupBase  float64            `json:"markup_base"`
}

// PendingResponse is the shape of GET /api/internal/vendor-dialogue/pending.
type PendingResponse struct {
	Count   int            `json:"count"`
	Vendors []VendorRecord `json:"vendors"`
}

// SubmitLinesRequest is the body for POST /api/internal/vendor-dialogue/{uuid}/lines.
type SubmitLinesRequest struct {
	LineType           string   `json:"line_type"`
	InteractionBucket  string   `json:"interaction_bucket"`
	TransactionContext string   `json:"transaction_context"`
	InventoryContext   string   `json:"inventory_context"`
	GenerationVersion  int      `json:"generation_version"`
	Lines              []string `json:"lines"`
}

// statusUpdateBody is the body for PATCH /api/internal/vendor-dialogue/{uuid}/status.
type statusUpdateBody struct {
	Status      string `json:"status"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchPending polls PHP for vendors needing dialogue generation.
// GET /api/internal/vendor-dialogue/pending
func (c *Client) FetchPending() ([]VendorRecord, error) {
	resp, err := c.doRequest("GET", c.baseURL+"/api/internal/vendor-dialogue/pending", nil)
	if err != nil {
		return nil, fmt.Errorf("FetchPending request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FetchPending: PHP returned status %d: %s", resp.StatusCode, string(body))
	}

	var result PendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("FetchPending: failed to decode response: %w", err)
	}

	return result.Vendors, nil
}

// UpdateStatus tells PHP the generation status for a vendor.
// status must be one of: "generating", "complete", "failed"
// generatedAt is an ISO8601 string, only sent when status == "complete".
// PATCH /api/internal/vendor-dialogue/{uuid}/status
func (c *Client) UpdateStatus(vendorUUID, status, generatedAt string) error {
	body := statusUpdateBody{Status: status}
	if status == "complete" && generatedAt != "" {
		body.GeneratedAt = generatedAt
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("UpdateStatus: failed to marshal body: %w", err)
	}

	url := fmt.Sprintf("%s/api/internal/vendor-dialogue/%s/status", c.baseURL, vendorUUID)
	resp, err := c.doRequest("PATCH", url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("UpdateStatus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("UpdateStatus: PHP returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SubmitLines sends generated lines for one dialogue scope to PHP.
// PHP validates and stores them.
// POST /api/internal/vendor-dialogue/{uuid}/lines
func (c *Client) SubmitLines(vendorUUID string, req SubmitLinesRequest) error {
	encoded, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("SubmitLines: failed to marshal body: %w", err)
	}

	url := fmt.Sprintf("%s/api/internal/vendor-dialogue/%s/lines", c.baseURL, vendorUUID)
	resp, err := c.doRequest("POST", url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("SubmitLines request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SubmitLines: PHP returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// doRequest executes an authenticated HTTP request to the PHP internal API.
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request to %s: %w", method, url, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.http.Do(req)
}
