package opencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type createSessionResponse struct {
	ID string `json:"id"`
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type promptRequest struct {
	Parts   []messagePart `json:"parts"`
	NoReply bool          `json:"noReply,omitempty"`
}

func CreateSession(serverURL string) (string, error) {
	resp, err := http.Post(serverURL+"/session", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session %d: %s", resp.StatusCode, string(body))
	}

	var result createSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode session response: %w", err)
	}

	return result.ID, nil
}

func SendMessage(serverURL, sessionID, query string) error {
	req := promptRequest{
		Parts: []messagePart{
			{Type: "text", Text: query},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/prompt_async", serverURL, sessionID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send message %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
