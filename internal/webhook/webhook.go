package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

// Payload is a webhook payload.
type Payload struct {
	Username string `json:"username"`
	IconURL  string `json:"icon_url"`
	Text     string `json:"text"`
}

// Send sends a Mattermost webhook to the provided URL.
func Send(ctx context.Context, webhookURL string, payload *Payload) error {
	if len(payload.Username) == 0 {
		return errors.New("payload username value not set")
	}
	if len(payload.Text) == 0 {
		return errors.New("payload text value not set")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook")
	}

	return nil
}
