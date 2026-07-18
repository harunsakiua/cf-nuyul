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
	"sync"
	"time"
)

var (
	proxyAddr      = "http://YOUR_PROXY_USER:PASS@HOST:PORT"
	captchaKey     = "YOUR_2CAPTCHA_API_KEY"
	defaultPass    = "CfAb3xK9mQ!Secure99"
	sitekeySignup  = "0x4AAAAAAAJel0iaAR3mgkjp"
	sitekeyLogin   = "0x4AAAAAADlYAGQxVeGSor-h"
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

// ==================== 2CAPTCHA v1 API ====================

func submitCap(sitekey, pageurl, action string) string {
	url := fmt.Sprintf("https://2captcha.com/in.php?key=%s&method=turnstile&sitekey=%s&pageurl=https://dash.cloudflare.com/%s&action=%s&json=1",
		captchaKey, sitekey, pageurl, action)
	r, err := http.Get(url)
	if err != nil {
		return ""
	}
	var tr struct{ Request string }
	json.NewDecoder(r.Body).Decode(&tr)
	r.Body.Close()
	if tr.Request == "" || tr.Request == "0" {
		return ""
	}
	return tr.Request
}

func waitCap(id string) (string, string) {
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		r, err := http.Get(fmt.Sprintf("https://2captcha.com/res.php?key=%s&action=get&id=%s&json=1", captchaKey, id))
		if err != nil {
			continue
		}
		var pr struct {
			Status    int    `json:"status"`
			Request   string `json:"request"`
			UserAgent string `json:"useragent"`
		}
		json.NewDecoder(r.Body).Decode(&pr)
		r.Body.Close()
		if pr.Status == 1 && pr.Request != "" {
			// Handle "token|ua" format from 2captcha
			tok := pr.Request
			ua := pr.UserAgent
			if idx := strings.Index(tok, "|"); idx >= 0 {
				parts := strings.SplitN(tok, "|", 2)
				tok = parts[0]
				if len(parts) > 1 && parts[1] != "" {
					ua = parts[1]
				}
			}
			return tok, ua
		}
	}
	return "", ""
}

// ==================== HTTP HELPERS ====================

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
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}

func httpPOST(urlStr, body string) string {
	proxyURL, _ := url.Parse(proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   30 * time.Second,
	}
	req, _ := http.NewRequest("POST", urlStr, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}

func curlLogin(email, pass, capTok, userAgent string) string {
	body := fmt.Sprintf(`{"email":"%s","password":"%s","cf_challenge_response":"%s"}`, email, pass, capTok)
	ua := userAgent
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	}
	hdrFile, _ := os.CreateTemp("", "cfhdr*.txt")
	hdrPath := hdrFile.Name()
	hdrFile.Close()
	defer os.Remove(hdrPath)

	cmd := exec.Command("curl", "-s", "-D", hdrPath, "-x", proxyAddr,
		"-X", "POST", "https://api.cloudflare.com/api/v4/login",
		"-H", "User-Agent: "+ua,
		"-H", "Content-Type: application/json",
		"-d", body)
	cmd.Run()

	hdrData, _ := os.ReadFile(hdrPath)
	hdrContent := string(hdrData)

	for _, line := range strings.Split(hdrContent, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "set-cookie:") && strings.Contains(line, "vses2=") {
			if idx := strings.Index(line, "vses2="); idx >= 0 {
				v := line[idx+6:]
				if idx2 := strings.IndexAny(v, ";\r\n"); idx2 >= 0 {
					v = v[:idx2]
				}
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func execCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	cmd.Run()
	return strings.TrimSpace(out.String())
}

// ==================== TEMPIK API ====================
// getVerifyLinkTempikSync is in ../tempikclient.go (shared with serial version)

func getVerifyLinkIMAP(targetEmail string, timeoutSec int) string {
	// Replaced by Tempik — call the shared Tempik client
	return getVerifyLinkTempikSync(targetEmail, timeoutSec)
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
			if s, ok := v[p].(string); ok {
				return s
			}
			d = v[p]
		case []interface{}:
			if p == "0" && len(v) > 0 {
				d = v[0]
			} else {
				return ""
			}
		default:
			return ""
		}
	}
	return ""
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
	if len(j) > 120 {
		return j[:120] + "..."
	}
	return j
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ==================== MAIN ====================

func main() {
	rand.Seed(time.Now().UnixNano())
	resultFile := "results.json"

	uname := usernames[rand.Intn(len(usernames))]
	dom := domains[rand.Intn(len(domains))]
	email := uname + fmt.Sprintf("%d", rand.Intn(9999)) + "@" + dom
	pass := defaultPass
	acc := &Account{Email: email, Password: pass}
	fmt.Printf("📧 %s:%s\n\n", email, pass)

	t0 := time.Now()
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"

		// Start Tempik immediately (parallel — polling for email, with retry)
		emailCh := make(chan string, 1)
		go func() {
			time.Sleep(6 * time.Second)
			for i := 0; i < 3; i++ {
				result := getVerifyLinkIMAP(email, 120)
				if result != "" {
					emailCh <- result
					return
				}
				time.Sleep(30 * time.Second) // wait before retry
			}
			emailCh <- ""
		}()

	// ===== STEP 1: BOOTSTRAP =====
	fmt.Print("[1] Bootstrap... ")
	bootResp := httpGET("https://api.cloudflare.com/api/v4/system/bootstrap")
	sec := jStr(bootResp, "result.data.data.security_token")
	if sec == "" {
		acc.Error = "bootstrap failed"
		saveResult(resultFile, acc)
		return
	}
	fmt.Println("✅")

	// ===== STEP 2-3: PARALLEL CAPTCHA =====
	var signupTok, loginTok, ua, ua2 string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		fmt.Print("[2] Signup captcha... ")
		challengeResp := httpGET("https://api.cloudflare.com/api/v4/captcha/challenge?context=signup")
		sk := jStr(challengeResp, "result.key")
		if sk == "" { sk = sitekeySignup }
		task := submitCap(sk, "sign-up", "signup")
		if task == "" { signupTok = "ERR"; return }
		tok, u := waitCap(task)
		if tok == "" { signupTok = "ERR"; return }
		signupTok, ua = tok, u
		fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		fmt.Print("[3] Login captcha... ")
		loginChallenge := httpGET("https://api.cloudflare.com/api/v4/captcha/challenge?context=login")
		sk := jStr(loginChallenge, "result.key")
		if sk == "" { sk = sitekeyLogin }
		task := submitCap(sk, "login", "login")
		if task == "" { loginTok = "ERR"; return }
		tok, u := waitCap(task)
		if tok == "" { loginTok = "ERR"; return }
		loginTok, ua2 = tok, u
		fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())
	}()

	wg.Wait()

	if signupTok == "" || strings.HasPrefix(signupTok, "ERR") {
		acc.Error = "signup captcha failed"
		saveResult(resultFile, acc); return
	}
	if loginTok == "" || strings.HasPrefix(loginTok, "ERR") {
		acc.Error = "login captcha failed"
		saveResult(resultFile, acc); return
	}
	if ua != "" { userAgent = ua }
	if ua2 != "" { userAgent = ua2 }

	// ===== STEP 4: SIGNUP =====
	fmt.Print("[4] Signup... ")
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	signupMap := map[string]interface{}{
		"email": email, "password": pass, "security_token": sec,
		"cf_challenge_response": signupTok, "method": "Onboarding: New_v2",
		"locale": "en-US", "legal_stamp": ts, "mrk_optin": true,
		"opt_ins": map[string]interface{}{}, "mrktCheckboxDisplayed": false,
		"hCaptchaDisplayed": false,
	}
	signupBytes, _ := json.Marshal(signupMap)
	resp := httpPOST("https://api.cloudflare.com/api/v4/user/create", string(signupBytes))
	if !jBool(resp) {
		acc.Error = "signup: " + jErr(resp)
		saveResult(resultFile, acc)
		return
	}
	acc.UserID = jStr(resp, "result.id")
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	// ===== STEP 5: LOGIN =====

	fmt.Print("[5] Login... ")
	vses2 := curlLogin(email, pass, loginTok, userAgent)
	if vses2 == "" {
		acc.Error = "login failed"
		saveResult(resultFile, acc)
		return
	}
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	CK := fmt.Sprintf("Cookie: vses2=%s; __cf_logged_in=1", vses2)

	// ===== STEP 6: EMAIL =====
	fmt.Print("[6] Email... ")
	var verifyToken string
	select {
	case verifyToken = <-emailCh:
	case <-time.After(180 * time.Second):
	}
	if verifyToken == "" {
		acc.Error = "email timeout"
		saveResult(resultFile, acc); return
	}
	fmt.Printf("✅ %.1fs\n", time.Since(t0).Seconds())

	tokParam := verifyToken
	if tokParam == "" {
		acc.Error = "no token in URL"
		saveResult(resultFile, acc)
		return
	}

	// ===== STEP 7: VERIFY =====
	fmt.Print("[7] Verify... ")
	vBody := fmt.Sprintf(`{"token":"%s"}`, tokParam)
	verifyResp := execCmd("curl", "-s", "-x", proxyAddr,
		"-X", "PUT", "https://api.cloudflare.com/api/v4/user/email-verification",
		"-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"-H", "Content-Type: application/json",
		"-H", "Origin: https://dash.cloudflare.com",
		"-H", "x-cross-site-security: dash",
		"-H", CK,
		"-d", vBody)
	if jBool(verifyResp) {
		fmt.Print("✅ ")
	} else {
		fmt.Printf("⚠️%s ", jErr(verifyResp))
	}
	fmt.Printf("%.1fs\n", time.Since(t0).Seconds())

	// ===== STEP 8: ACCOUNT =====
	fmt.Print("[8] Account... ")
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

	// Save complete data
	acc.Success = true
	saveResult(resultFile, acc)
	
	// Append to accounts file with full details
	line := fmt.Sprintf("%s | %s | %s | %s | %s\n", email, defaultPass, acc.UserID, aid, tok)
	f, _ := os.OpenFile("/home/ubuntu/akun_cf_parallel_tempik.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(line)
	f.Close()
	
	fmt.Printf("\n✅✅✅ SUKSES! %s  (⏱️ %.0fs)\n", email, time.Since(t0).Seconds())
	fmt.Printf("📁 hasil: %s\n", resultFile)
}

func saveResult(path string, acc *Account) {
	data, _ := json.MarshalIndent(acc, "", "  ")
	os.WriteFile(path, data, 0644)
	fmt.Printf("\n📝 %s\n", path)
	if acc.Success {
		fmt.Printf("✅✅✅ SUKSES! %s\n", acc.Email)
	} else {
		fmt.Printf("❌ GAGAL: %s\n", acc.Error)
	}
}
