package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ─── Tempik API Client ────────────────────────────────────────────────────
// Drop-in replacement for Gmail API / IMAP in CF nuyul scripts.
// Uses self-hosted disposable email at tempik.YOUR_TEMPIK_DOMAIN_1

var tempikBaseURL = "https://tempik.YOUR_TEMPIK_DOMAIN_1"

// tempikHTTPGet performs an HTTP GET to Tempik API
func tempikHTTPGet(urlStr, sessionId string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if sessionId != "" {
		req.Header.Set("x-session-id", sessionId)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), nil
}

// tempikHTTPPost performs an HTTP POST to Tempik API
func tempikHTTPPost(urlStr, sessionId, bodyData string) (string, error) {
	req, err := http.NewRequest("POST", urlStr, strings.NewReader(bodyData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if sessionId != "" {
		req.Header.Set("x-session-id", sessionId)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), nil
}

// tempikCreateSession creates an anonymous Tempik session.
// One session = isolated inbox list (same browser concept).
func tempikCreateSession() string {
	resp, err := tempikHTTPGet(tempikBaseURL+"/api/session", "")
	if err != nil {
		return ""
	}
	var d struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal([]byte(resp), &d) != nil {
		return ""
	}
	return d.SessionID
}

// tempikCreateInbox creates an inbox for the given localPart + domain
// (e.g. "dimasaskdow1234", "YOUR_TEMPIK_DOMAIN_2" → dimasaskdow1234@YOUR_TEMPIK_DOMAIN_2)
func tempikCreateInbox(sessionId, localPart, domain string) bool {
	body := fmt.Sprintf(`{"localPart":"%s","domain":"%s"}`, localPart, domain)
	resp, err := tempikHTTPPost(tempikBaseURL+"/api/inboxes", sessionId, body)
	if err != nil {
		return false
	}
	return strings.Contains(resp, `"address"`)
}

// tempikPollVerification polls Tempik for verification email from CF.
// Returns the raw verification token, or "" on timeout.
func tempikPollVerification(sessionId, address string, timeoutSec int) string {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	pollInterval := 3 * time.Second

	for time.Now().Before(deadline) {
		encoded := url.QueryEscape(address)
		resp, err := tempikHTTPGet(tempikBaseURL+"/api/inboxes/"+encoded+"/messages", sessionId)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Parse messages array
		var messages []struct {
			Body    string `json:"body"`
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal([]byte(resp), &messages); err != nil || len(messages) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		// First message = newest (ordered by received_at DESC)
		bodyText := messages[0].Body
		if bodyText != "" {
			if token := extractVerificationToken(bodyText); token != "" {
				return token
			}
		}
		time.Sleep(pollInterval)
	}
	return ""
}

// extractVerificationToken searches email body for CF email-verification URL
// and extracts the token parameter value.
// Pattern: https://dash.cloudflare.com/email-verification?token=<base64url_token>
func extractVerificationToken(body string) string {
	patterns := []string{
		"https://dash.cloudflare.com/email-verification?token=",
		"dash.cloudflare.com/email-verification?token=",
	}
	return searchTokenInText(body, patterns)
}

func searchTokenInText(text string, patterns []string) string {
	for _, search := range patterns {
		idx := strings.Index(text, search)
		if idx < 0 {
			continue
		}
		tokenStart := idx + len(search)
		tokenEnd := tokenStart
		for tokenEnd < len(text) {
			c := text[tokenEnd]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' {
				tokenEnd++
			} else {
				break
			}
		}
		if tokenEnd > tokenStart {
			token := text[tokenStart:tokenEnd]
			if len(token) > 200 {
				return token
			}
		}
	}
	// Fallback: remove newlines (folded headers) and retry
	cleaned := strings.ReplaceAll(text, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	if cleaned != text {
		return searchTokenInText(cleaned, patterns)
	}
	return ""
}

// getVerifyLinkTempikSync is the main entry point — replaces getVerifyLinkSync.
// Creates a Tempik session → inbox → polls for CF verification email.
// Returns the raw verification token (not the full URL).
func getVerifyLinkTempikSync(targetEmail string, timeoutSec int) string {
	// Extract localPart and domain from email (e.g. "dimasaskdow1234@YOUR_TEMPIK_DOMAIN_2")
	parts := strings.SplitN(targetEmail, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	localPart := parts[0]
	domain := parts[1]

	// 1. Create Tempik session
	sessionId := tempikCreateSession()
	if sessionId == "" {
		return ""
	}

	// 2. Create inbox for this email on the correct domain
	if !tempikCreateInbox(sessionId, localPart, domain) {
		return ""
	}

	// 3. Poll for CF verification email
	return tempikPollVerification(sessionId, targetEmail, timeoutSec)
}
