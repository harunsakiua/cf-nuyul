package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	proxyAddr      = "http://YOUR_PROXY_USER:PASS@HOST:PORT"
	defaultPass    = "CfAb3xK9mQ!Secure99"
	sitekeySignup  = "0x4AAAAAAAJel0iaAR3mgkjp"
	sitekeyLogin   = "0x4AAAAAADlYAGQxVeGSor-h"

	// Boterdrop solver endpoint
	boterdropURL = "http://127.0.0.1:8000"
)

type Account struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	UserID    string `json:"user_id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	APIToken  string `json:"api_token,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

var usernames = []string{
	"dimasaskdow", "agussaputra", "ririsandri", "bayupamungkas",
	"siskaamelia", "adityapratama", "dewipertiwi", "fikrihidayat",
	"rianmaulana", "sariindah", "bayupratama", "dewisartika",
	"fikriramadhan", "ririsnurhaliza", "aguswibowo", "siskaoktavia",
}

var domains = []string{
	"YOUR_TEMPIK_DOMAIN_1", "YOUR_TEMPIK_DOMAIN_2", "YOUR_TEMPIK_DOMAIN_3", "YOUR_TEMPIK_DOMAIN_4",
}

// ==================== BOTERDROP SOLVER ====================

// boterdropSolve sends a Turnstile solving task to Boterdrop and polls for result.
// Returns token + empty user-agent (Boterdrop doesn't provide UA for Turnstile).
func boterdropSolve(sitekey, pageurl, action string) string {
	// Step 1: Submit task
	taskURL := fmt.Sprintf("%s/turnstile?url=https://dash.cloudflare.com/%s&sitekey=%s",
		boterdropURL, pageurl, sitekey)
	if action != "" {
		taskURL += "&action=" + action
	}

	resp, err := http.Get(taskURL)
	if err != nil {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var taskResp struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	json.Unmarshal(body, &taskResp)

	if taskResp.TaskID == "" {
		// Maybe direct result if no queue?
		var directResp struct {
			Status string `json:"status"`
			Value  string `json:"value"`
		}
		json.Unmarshal(body, &directResp)
		if directResp.Status == "success" && directResp.Value != "" {
			return directResp.Value
		}
		return ""
	}

	// Step 2: Poll for result
	for i := 0; i < 60; i++ { // max 60s
		time.Sleep(1 * time.Second)

		pollURL := fmt.Sprintf("%s/result?id=%s", boterdropURL, taskResp.TaskID)
		pollResp, err := http.Get(pollURL)
		if err != nil {
			continue
		}
		pollBody, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		var result struct {
			Status      string  `json:"status"`
			Value       string  `json:"value"`
			ElapsedTime float64 `json:"elapsed_time"`
		}
		json.Unmarshal(pollBody, &result)

		switch result.Status {
		case "success":
			if result.Value != "" {
				return result.Value
			}
			return ""
		case "error", "timeout":
			return ""
		default:
			// still processing, continue polling
			continue
		}
	}
	return ""
}

// ==================== 2CAPTCHA FALLBACK ====================

func submitCap2captcha(sitekey, pageurl, action string) string {
	url := fmt.Sprintf("https://2captcha.com/in.php?key=%s&method=turnstile&sitekey=%s&pageurl=https://dash.cloudflare.com/%s&action=%s&json=1",
		"YOUR_2CAPTCHA_API_KEY", sitekey, pageurl, action)
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d map[string]interface{}
	json.Unmarshal(body, &d)
	id, ok := d["request"].(string)
	if !ok || id == "" {
		return ""
	}

	ua := ""
	for i := 0; i < 60; i++ {
		time.Sleep(3 * time.Second)
		pollURL := fmt.Sprintf("https://2captcha.com/res.php?key=%s&action=get&id=%s&json=1", "YOUR_2CAPTCHA_API_KEY", id)
		pollResp, err := http.Get(pollURL)
		if err != nil {
			continue
		}
		pollBody, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()
		var pd map[string]interface{}
		json.Unmarshal(pollBody, &pd)
		if pd["status"] == nil {
			continue
		}
		if s, ok := pd["status"].(float64); !ok || int(s) != 1 {
			continue
		}
		token, _ := pd["request"].(string)
		if strings.Contains(token, "|") {
			parts := strings.SplitN(token, "|", 2)
			token = parts[0]
			if len(parts) > 1 {
				ua = parts[1]
			}
		}
		if token == "" {
			return ""
		}
		return token + "|" + ua
	}
	return ""
}

// solveCaptcha tries Boterdrop first, falls back to 2captcha.
// Returns "token" or "token|ua"
func solveCaptcha(sitekey, pageurl, action string) string {
	// Try Boterdrop first (fast, free)
	token := boterdropSolve(sitekey, pageurl, action)
	if token != "" {
		fmt.Printf("⚡B ")
		return token + "|Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	}
	// Fallback to 2captcha (proven)
	fmt.Printf("⏳2c ")
	result := submitCap2captcha(sitekey, pageurl, action)
	if result != "" {
		return result
	}
	return ""
}

// ==================== NET/HTTP HELPERS ====================

func httpGET(urlStr string) string {
	proxyURL, _ := url.Parse(proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   30 * time.Second,
	}
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func execCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	cmd.Run()
	return strings.TrimSpace(out.String())
}

// ==================== JSON HELPERS ====================

func jStr(j, path string) string {
	var d interface{}
	json.Unmarshal([]byte(j), &d)
	parts := strings.Split(path, ".")
	for _, p := range parts {
		if d == nil {
			return ""
		}
		switch v := d.(type) {
		case map[string]interface{}:
			d = v[p]
		case []interface{}:
			idx := 0
			fmt.Sscanf(p, "%d", &idx)
			if idx < len(v) {
				d = v[idx]
			} else {
				return ""
			}
		default:
			return ""
		}
	}
	if s, ok := d.(string); ok {
		return s
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func jBool(j string) bool {
	return strings.Contains(j, `"success":true`)
}

func jErr(j string) string {
	var d map[string]interface{}
	json.Unmarshal([]byte(j), &d)
	if errs, ok := d["errors"].([]interface{}); ok && len(errs) > 0 {
		if e, ok := errs[0].(map[string]interface{}); ok {
			if msg, ok := e["message"].(string); ok {
				return msg
			}
		}
	}
	return ""
}

// ==================== TEMPIK API ====================
// getVerifyLinkTempikSync is in tempikclient.go

// ==================== MAIN FLOW ====================

func saveResult(path string, acc *Account) {
	f, _ := os.Create(path)
	json.NewEncoder(f).Encode(acc)
	f.Close()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	resultFile := "results.json"

	name := usernames[rand.Intn(len(usernames))]
	suffix := rand.Intn(10000)
	domain := domains[rand.Intn(len(domains))]
	email := fmt.Sprintf("%s%d@%s", name, suffix, domain)
	pass := defaultPass

	fmt.Printf("📧 %s:%s\n", email, pass)

	acc := &Account{Email: email, Password: pass}

	// ===== STEP 1: BOOTSTRAP =====
	fmt.Print("[1] Bootstrap... ")
	t0 := time.Now()
	bootResp := httpGET("https://api.cloudflare.com/api/v4/system/bootstrap")
	sec := jStr(bootResp, "result.data.data.security_token")
	if sec == "" {
		acc.Error = "bootstrap failed"
		saveResult(resultFile, acc)
		return
	}
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// Start email search in background via Tempik
	emailCh := make(chan string, 1)
	go func() {
		time.Sleep(5 * time.Second)
		token := getVerifyLinkTempikSync(email, 120)
		if token != "" {
			emailCh <- token
		} else {
			emailCh <- ""
		}
	}()

	// ===== SOLVE CAPTCHAS (parallel: Boterdrop first, 2captcha fallback) =====
	fmt.Print("[4] Captchas... ")
	t0 = time.Now()

	type capResult struct {
		token string
		ua    string
	}
	signupCh := make(chan capResult, 1)
	loginCh := make(chan capResult, 1)

	go func() {
		c := solveCaptcha(sitekeySignup, "sign-up", "signup")
		tok, ua := c, ""
		if idx := strings.Index(c, "|"); idx >= 0 {
			tok = c[:idx]
			ua = c[idx+1:]
		}
		signupCh <- capResult{tok, ua}
	}()
	go func() {
		c := solveCaptcha(sitekeyLogin, "login", "login")
		tok, ua := c, ""
		if idx := strings.Index(c, "|"); idx >= 0 {
			tok = c[:idx]
			ua = c[idx+1:]
		}
		loginCh <- capResult{tok, ua}
	}()

	// Wait for signup captcha FIRST
	sr := <-signupCh
	signupTok := sr.token
	if signupTok == "" {
		acc.Error = "signup captcha failed"
		saveResult(resultFile, acc)
		return
	}
	ua := sr.ua
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	}

	// ===== STEP 3: SIGNUP POST =====
	fmt.Print("[3] Signup... ")
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	signupMap := map[string]interface{}{
		"email": email, "password": pass, "security_token": sec,
		"cf_challenge_response": signupTok, "method": "Onboarding: New_v2",
		"locale": "en-US", "legal_stamp": ts, "mrk_optin": true,
	}
	signupBody, _ := json.Marshal(signupMap)
	proxyURL, _ := url.Parse(proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   15 * time.Second,
	}
	req, _ := http.NewRequest("POST", "https://api.cloudflare.com/api/v4/user/create",
		bytes.NewReader(signupBody))
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		acc.Error = "signup: " + err.Error()
		saveResult(resultFile, acc)
		return
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	signupResp := string(respBody)
	if !jBool(signupResp) {
		acc.Error = "signup: " + jErr(signupResp)
		saveResult(resultFile, acc)
		return
	}
	acc.UserID = jStr(signupResp, "result.id")
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// Wait for login captcha
	lr := <-loginCh
	loginTok := lr.token
	if loginTok == "" {
		acc.Error = "login captcha failed"
		saveResult(resultFile, acc)
		return
	}
	if lr.ua != "" {
		ua = lr.ua
	}

	// ===== STEP 5: LOGIN =====
	fmt.Print("[5] Login... ")
	t0 = time.Now()
	loginBody := fmt.Sprintf(`{"email":"%s","password":"%s","cf_challenge_response":"%s"}`,
		email, pass, loginTok)
	vses2 := ""
	hdrFile := "/tmp/cf_login_hdr.txt"
	os.WriteFile(hdrFile, []byte(""), 0644)
	execCmd("curl", "-s", "-D", hdrFile, "-x", proxyAddr,
		"-X", "POST", "https://api.cloudflare.com/api/v4/login",
		"-H", "User-Agent: "+ua,
		"-H", "Content-Type: application/json",
		"-d", loginBody)
	hdrData, _ := os.ReadFile(hdrFile)
	for _, line := range strings.Split(string(hdrData), "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "set-cookie:") && strings.Contains(lower, "vses2=") {
			if start := strings.Index(lower, "vses2="); start >= 0 {
				v := lower[start+6:]
				if end := strings.IndexAny(v, "; \t"); end >= 0 {
					v = v[:end]
				}
				vses2 = v
				break
			}
		}
	}
	if vses2 == "" {
		acc.Error = "login failed"
		saveResult(resultFile, acc)
		return
	}
	CK := fmt.Sprintf("Cookie: vses2=%s; __cf_logged_in=1", vses2)
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// ===== STEP 6: EMAIL =====
	fmt.Print("[6] Email... ")
	t0 = time.Now()
	var verifyToken string
	select {
	case verifyToken = <-emailCh:
	case <-time.After(130 * time.Second):
		verifyToken = ""
	}
	if verifyToken == "" {
		acc.Error = "email timeout"
		saveResult(resultFile, acc)
		return
	}
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// ===== STEP 7: VERIFY =====
	fmt.Print("[7] Verify... ")
	t0 = time.Now()
	tokParam := verifyToken
	if tokParam == "" {
		acc.Error = "no token in URL"
		saveResult(resultFile, acc)
		return
	}

	vBody := fmt.Sprintf(`{"token":"%s"}`, tokParam)
	os.WriteFile("/tmp/cf_verify_body.json", []byte(vBody), 0644)
	verifyResp := execCmd("curl", "-s", "-x", proxyAddr,
		"-X", "PUT", "https://api.cloudflare.com/api/v4/user/email-verification",
		"-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"-H", "Content-Type: application/json",
		"-H", "Origin: https://dash.cloudflare.com",
		"-H", "x-cross-site-security: dash",
		"-H", CK,
		"-d", "@"+"/tmp/cf_verify_body.json")

	if jBool(verifyResp) {
		fmt.Print("✅ ")
	} else {
		errMsg := jErr(verifyResp)
		fmt.Printf("⚠️%s ", errMsg)
	}
	fmt.Printf("%.1fs\n", time.Since(t0).Seconds())

	// ===== STEP 8: ACCOUNT =====
	fmt.Print("[8] Account... ")
	t0 = time.Now()
	acctResp := execCmd("curl", "-s", "-x", proxyAddr,
		"https://api.cloudflare.com/api/v4/accounts?per_page=5",
		"-H", CK, "-H", "User-Agent: Mozilla/5.0")
	aid := jStr(acctResp, "result.0.id")
	if aid == "" || aid == "0" {
		acc.Error = "account: " + jErr(acctResp)
		saveResult(resultFile, acc)
		return
	}
	acc.AccountID = aid
	fmt.Printf("%s ", aid[:12])

	// ===== STEP 9: API TOKEN =====
	fmt.Print("token... ")
	t0 = time.Now()
	tokenBody := `{"name":"Workers AI","condition":{},"policies":[{"effect":"allow","resources":{"com.cloudflare.api.account.*":"*"},"permission_groups":[{"id":"a92d2450e05d4e7bb7d0a64968f83d11"},{"id":"bacc64e0f6c34fc0883a1223f938a104"}]}]}`
	tokenResp := execCmd("curl", "-s", "-x", proxyAddr,
		"-X", "POST", "https://api.cloudflare.com/api/v4/user/tokens",
		"-H", CK,
		"-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"-H", "Content-Type: application/json",
		"-H", "Origin: https://dash.cloudflare.com",
		"-H", "x-cross-site-security: dash",
		"-d", tokenBody)
	tok := jStr(tokenResp, "result.value")
	if tok == "" {
		acc.Error = "token: " + jErr(tokenResp)
		saveResult(resultFile, acc)
		return
	}
	acc.APIToken = tok
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// ===== SUCCESS =====
	acc.Success = true
	saveResult(resultFile, acc)

	line := fmt.Sprintf("%s | %s | %s | %s | %s\n", email, defaultPass, acc.UserID, aid, tok)
	f, _ := os.OpenFile("/home/ubuntu/akun_cf_boterdrop_tempik.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(line)
	f.Close()

	fmt.Printf("\n📝 results.json\n✅✅✅ SUKSES! %s  (⏱️ %.0fs)\n", email, time.Since(t0).Seconds())
}
