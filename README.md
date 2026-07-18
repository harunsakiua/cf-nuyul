# CF-Nuyul — Cloudflare Workers AI Token Farming

> Mass Cloudflare account creation for Workers AI token farming.
> Free CAPTCHA (Boterdrop ~2.5s) + self-hosted disposable email (Tempik).

## 🎯 What is this?

Automated Cloudflare account signup pipeline that creates accounts en masse to farm Workers AI API tokens. Solves Turnstile CAPTCHA automatically and verifies email via disposable inbox.

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Boterdrop  │────→│  CF Signup   │────→│   Tempik     │
│  Turnstile  │     │  POST /user  │     │  Verify      │
│  ~2.5s      │     │  /create     │     │  Email       │
└─────────────┘     └──────┬───────┘     └──────┬───────┘
        ↓ (fallback)       │                    │
   ┌──────────┐            │                    │
   │ 2captcha │            │                    │
   └──────────┘            ▼                    ▼
                    ┌──────────────────────────────────┐
                    │  email:pass:token:aid → file     │
                    │  ~30 accounts/minute sustained    │
                    └──────────────────────────────────┘
```

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🤖 **Auto CAPTCHA** | Boterdrop (free, ~2.5s) with 2captcha fallback |
| 📧 **Disposable Email** | Tempik API — self-hosted, no Gmail OAuth |
| ⚡ **Mass Creation** | ~30 accounts/minute sustained |
| 🔄 **Hybrid Mode** | Boterdrop + 2captcha fallback automatically |
| 📊 **Live Monitoring** | Real-time success rate, captcha source tracking |

## 🏗️ Architecture

### Binaries

| File | Captcha | Email | Speed |
|------|---------|-------|-------|
| `boterdrop_main.go` | Boterdrop → 2captcha fallback | Tempik | ~30/min |
| `parallel_main.go` | 2captcha only | Tempik | ~18/min |
| `tempikclient.go` | Shared email client | Tempik API | — |

### Flow

```
1. Generate random email (nameXXXX@domain)
2. Solve Turnstile CAPTCHA (Boterdrop or 2captcha)
3. POST /api/v4/user/create (Cloudflare signup)
4. POST /api/v4/login (get session cookie)
5. Poll Tempik inbox for verification email
6. PUT /api/v4/user/email-verification (verify)
7. Output: email:password:token:account_id
```

## 🚀 Quick Start

### Prerequisites

- **Boterdrop** — [Boterdrop Solver](https://github.com/najibyahya/Boterdrop-Solver) running on `localhost:8000`
- **Tempik** — [Tempik](https://github.com/hirotomasato/tempik) self-hosted disposable email API
- **2captcha** (optional fallback) — API key with balance
- **Proxy** — Residential/mobile proxy for Cloudflare signup
- **Go 1.21+**

### Configuration

Edit the constants at the top of each file:

```go
// boterdrop_main.go
const boterdropURL = "http://localhost:8000"  // Your Boterdrop instance

// parallel_main.go
const captchaKey = "YOUR_2CAPTCHA_API_KEY"     // 2captcha API key

// tempikclient.go
var tempikBaseURL = "https://YOUR_TEMPIK_API_URL"  // Your Tempik deployment
var tempikDomains = []string{"YOUR_DOMAIN_1", "YOUR_DOMAIN_2"}
```

### Build & Run

```bash
# Boterdrop version (free captcha)
cd cf-go-boterdrop
go build -o cf-nuyul-boterdrop .
./cf-nuyul-boterdrop

# Parallel version (2captcha)
cd cf-go/parallel
go build -o cf-nuyul-parallel .
./cf-nuyul-parallel
```

### Batch Mode

```bash
#!/bin/bash
# batch_boterdrop.sh — infinite loop with 1s delay
while true; do
  ./cf-nuyul-boterdrop >> ~/akun_cf.txt
  sleep 1
done
```

## 📊 Performance

| Metric | Value |
|--------|-------|
| Accounts/min (Boterdrop) | ~30 |
| Accounts/min (Parallel) | ~18 |
| Boterdrop solve time | ~2.5s |
| 2captcha solve time | ~10-15s |
| Tempik email polling | ~3-7s |
| Success rate (Boterdrop) | ~85-95% |

## 🔧 Dependencies

| Service | Purpose | Repository |
|---------|---------|------------|
| **Boterdrop** | Free Turnstile CAPTCHA solver | [github.com/najibyahya/Boterdrop-Solver](https://github.com/najibyahya/Boterdrop-Solver) |
| **Tempik** | Self-hosted disposable email | [github.com/hirotomasato/tempik](https://github.com/hirotomasato/tempik) |
| 2captcha (optional) | Paid CAPTCHA fallback | [2captcha.com](https://2captcha.com) |

## ⚠️ Disclaimer

This tool is for **educational and research purposes only**. Cloudflare's Terms of Service prohibit automated account creation. The authors are not responsible for any misuse. Use at your own risk.

## 📄 License

MIT — do whatever you want. Don't blame us.
