package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type TwilioWhatsAppSender struct {
	AccountSID string
	AuthToken  string
	From       string // example: whatsapp:+14155238886
}

func NewTwilioWhatsAppSender(accountSID, authToken, from string) *TwilioWhatsAppSender {
	return &TwilioWhatsAppSender{
		AccountSID: accountSID,
		AuthToken:  authToken,
		From:       from,
	}
}

func (s *TwilioWhatsAppSender) Send(ctx context.Context, msg WhatsAppMessage) (string, error) {
	to := strings.TrimSpace(msg.To)
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", s.From)
	form.Set("Body", msg.Body)

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.AccountSID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(s.AccountSID, s.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("twilio send failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Sid     string `json:"sid"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse twilio response: %w, body=%s", err, string(respBody))
	}

	if result.Sid == "" {
		return "", fmt.Errorf("twilio response missing sid: %s", string(respBody))
	}

	return result.Sid, nil
}
