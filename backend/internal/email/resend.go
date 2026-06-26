package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

type resendSender struct {
	apiKey string
	from   string
	client *http.Client
}

func newResendSender(apiKey, from string) *resendSender {
	return &resendSender{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *resendSender) send(to, subject, body string) error {
	payload := map[string]interface{}{
		"from":    r.from,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("resend api %s: %s", res.Status, string(respBody))
	}
	return nil
}
