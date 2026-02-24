# Detailed Configuration Guide

*Read this in [Vietnamese](configuration-guide.md).*

> **Objective**: Obtain all mandatory API keys and credentials required to boot the system 100%.

---

## Prerequisite Checklist

Ensure you have the following information available (check [x] when done):

- [ ] **Telegram Bot Token** - Required (UI Chat Interface)
- [ ] **Gemini API Key** - Required (AI brain routing)
- [ ] **Voyage AI API Key** - Required (embeddings functionality)
- [ ] **Memos Access Token** - Required (backend storage)
- [ ] **Ngrok Auth Token** - Required (webhook tunneling)
- [ ] **Google Calendar Credentials** - Optional (auto-scheduling)
- [ ] **Webhook Secret** - Optional (Git trigger automation)

---

## 1. Telegram Bot Token

**Purpose**: Operates as the singular conversational UI for the system.

**Time required**: ~2 minutes

### Steps

1. **Open Telegram** and query `@BotFather`.

2. **Create New Bot**:

   ```
   You: /newbot
   BotFather: Alright, a new bot. How are we going to call it?
   
   You: My Task Manager Bot
   BotFather: Good. Now let's choose a username for your bot.
   
   You: my_task_manager_bot
   BotFather: Done! Here's your token: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```

3. **Copy the token** directly into the `.env` root file:

   ```bash
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```

### Security Alert

- **DO NOT** commit the token into your Git history.
- **DO NOT** share the token publicly.
- If compromised, utilize the `/revoke` command with BotFather to generate a new key entity.

### Verification

```bash
# Test bot token
curl https://api.telegram.org/bot<YOUR_TOKEN>/getMe

# Verified Response:
# {"ok":true,"result":{"id":123456789,"is_bot":true,"first_name":"My Task Manager Bot",...}}
```

---

## 2. Memos Access Token

**Purpose**: Gives the API capability to Read/Write tasks silently into the self-hosted Memos cluster.

**Time required**: ~3 minutes

### Steps

1. **Boot systems**:

   ```bash
   make up
   # Equivalent manually: docker compose up -d
   ```

2. **Visit Memos** locally at <http://localhost:5230>

3. **Initialize the Admin role**:
   - Click "Sign up"
   - Select Username: `admin` (or generic)
   - Assign strong backend password
   - Proceed "Sign up"

4. **Generate Authorization Token**:
   - Click your **Avatar** (Bottom-Left UI block)
   - Interface -> **Settings**
   - Navigation Tab -> **Tokens**
   - Select **Create** (or **New Token**)
   - Label Description: `Backend API`
   - Expiration Duration: **Never**
   - Click **Create**

5. **Copy the payload** into your `.env`:

   ```bash
   MEMOS_ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   ```

6. **Reset background instances**:

   ```bash
   make restart
   ```

### Verification

```bash
# Check endpoint health with token
curl -H "Authorization: Bearer <YOUR_TOKEN>" \
     http://localhost:5230/api/v1/memo

# Acknowledged Server Response: {"memos":[...]}
```

---

## 3. Gemini API Key

**Purpose**: Employs the "AI Brain" routing algorithms for Agent Orchestrating, Natural Language Comprehension, and Reasoning engines.

**Time required**: ~2 minutes

**Pricing metric**: FREE tier covers exactly 15 requests/minute, capped at +1M tokens/day.

### Steps

1. **Boot into** [Google AI Studio](https://aistudio.google.com/app/apikey)

2. **Login authorized access** utilizing normal Google Credentials.

3. **Mint the API Key**:
   - Click panel button **Create API key**
   - Attach to default Cloud Project
   - Click verification **Create API key in new project**
   - Extract string: `AIzaSyA...`

4. **Inject locally into `.env`**:

   ```bash
   GEMINI_API_KEY="AIzaSyA..."
   ```

### Quick Tips

- **Primary Model Target**: `gemini-2.0-flash-exp` (highest accuracy vs cost)
- **Constraint Cap**: 15 req/min (FREE).
- **Window Limitation**: Over 1 Million tokens allowed input.

### Verification

```bash
# Hit Gemini Endpoints directly
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent?key=<YOUR_KEY>" \
  -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"Hello"}]}]}'

# Expected Body: {"candidates":[{"content":{"parts":[{"text":"Hello! ..."}]}}]}
```

---

## 4. Voyage AI API Key

**Purpose**: High-fidelity embeddings generation for the Qdrant Vector Semantic Search Pipeline (RAG Architecture).

**Time required**: ~3 minutes

**Pricing metric**: FREE tier cap 100M tokens/month limit.

### Steps

1. **Initialize identity** natively at [Voyage AI Dashboard](https://dash.voyageai.com/)

2. **Confirm Auth** within inbox and proceed into portal.

3. **Extract Authorization String**:
   - Primary Sidebar -> **API Keys**
   - Block Click -> **Create new secret key**
   - Descriptor Assignment: `Task Management`
   - Finish **Create**
   - Collect Payload: `pa-...`

4. **Insert structurally to `.env`**:

   ```bash
   VOYAGE_API_KEY="pa-..."
   ```

### Model Info Architecture

- **Defined Model Identifier**: `voyage-3` (Standardized 1024 dimensional scope)
- **Primary Feature Array**: Multilingual capabilities, deep reasoning.

### Verification

```bash
# Push payload to cloud endpoint
curl https://api.voyageai.com/v1/embeddings \
  -H "Authorization: Bearer <YOUR_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"input":["Hello world"],"model":"voyage-3"}'

# 200 OK Body Format: {"object":"list","data":[{"object":"embedding","embedding":[0.123,...],"index":0}],...}
```

---

## 5. Ngrok Auth Token

**Purpose**: Bridging internal localhost boundaries to secure HTTPS internet addresses to cleanly ingest external Git Webhooks.

**Time required**: ~2 minutes

**Pricing metric**: FREE limits 1 single concurrent tunnel.

### Why exactly is Ngrok required?

Inversion-of-control protocols defined by Github/Gitlab cannot interface blindly with localhost.

- Host routing denies generic inbound access.
- Ngrok initiates a secure bidirectional mapping: `https://abc123.ngrok.io` -> `localhost:8080`.

### Steps

1. **Create Account Identity** via [Ngrok Authentication](https://dashboard.ngrok.com/signup)

2. **Isolate Primary Auth Token**:
   - Access Dashboard -> **Getting Started** -> **Your Authtoken**
   - Extract string: `2a...`

3. **Install to internal `.env` node**:

   ```bash
   NGROK_AUTHTOKEN=2a...
   ```

4. **Flush and Boot environments**:

   ```bash
   make down
   make up
   ```

### Verification

1. **Examine Internal Ngrok Console**: <http://localhost:4040>

2. **Scrape designated public endpoint string**:

   ```bash
   curl http://localhost:4040/api/tunnels | jq '.tunnels[0].public_url'
   # Captured Format: "https://abc123.ngrok.io"
   ```

3. **Confirm Webhook Reachability Layer**:

   ```bash
   curl https://abc123.ngrok.io/health
   # Success Body Assertion: {"status":"ok"}
   ```

---

## 6. Google Calendar Credentials (Optional)

**Purpose**: Autologistics scheduling logic and temporal reasoning collision checks.

**Time required**: ~10 minutes

### Why denotes this as Optional?

Code architecture scales down gracefully:

- Presence True: Event blocks, deep integrations, automatic synchronization operations.
- Presence False: Database tasks confined simply uniquely inside the Memos core instance.

### A: Retrieve Root `google-credentials.json`

1. **Access Web Portal System** [Google Cloud Console](https://console.cloud.google.com/)

2. **Generate Sub-Project Wrapper**:
   - Header **Select a project** -> **New Project**
   - Input Assignment: `Task Management`
   - Proceed **Create**

3. **Grant Library API Entitlements**:
   - Nav Sidebar -> **APIs & Services** -> **Library**
   - Query "Google Calendar API"
   - Click action **Enable**

4. **Mint Client Side OAuth Identities**:
   - Primary Sidebar -> **APIs & Services** -> **Credentials**
   - Panel **Create Credentials** -> **OAuth client ID**
   - Application Type Configurator: **Desktop app**
   - Display String: `Task Management Desktop`
   - Finalize **Create**

5. **Exfiltrate JSON Dump**:
   - Export Arrow (Download action state)
   - Store strictly at mapping root `secrets/google-credentials.json`

### B: Build User Root `token.json`

1. **Launch internal Auth Go subroutine**:

   ```bash
   go run scripts/gcal-auth/main.go secrets/google-credentials.json
   ```

2. **Follow execution terminal text context**:

   ```
   Go to the following link in your browser:
   https://accounts.google.com/o/oauth2/auth?...
   
   Enter authorization code:
   ```

3. **Authorize Web Proxy Client**:
   - Hit hyper-reference
   - Designate Google Entity
   - Authoritive **Allow**
   - Scrap Auth Payload Code from Target HTTPS URL: `?code=4/0A...`
   - Dump inside input STDIN terminal wrapper.

4. **Observe token output generation log**:

   ```
   Token saved to: secrets/token.json
   ```

5. **Add mapping into local `.env` definition**:

   ```bash
   GOOGLE_CALENDAR_CREDENTIALS=secrets/google-credentials.json
   ```

### Verification

```bash
# Call script iteratively to pull cache details instead
go run scripts/gcal-auth/main.go secrets/google-credentials.json
```

---

## 7. Webhook Secret (Optional)

**Purpose**: Defend HTTP inbound routing limits from DDoS attacks, enforcing HMAC verifications across the Gitlab / Github protocols.

**Time required**: ~1 minute

### Steps

1. **Generate complex secret string payload**:

   ```bash
   # Unix Option: OpenSSL
   openssl rand -hex 32
   
   # Python Routine Definition Alternative
   python3 -c "import secrets; print(secrets.token_hex(32))"
   
   # Output Capture Array: a1b2c3d4e5f6...
   ```

2. **Dump strictly to `.env` layer context**:

   ```bash
   WEBHOOK_SECRET=a1b2c3d4e5f6...
   WEBHOOK_ENABLED=true
   ```

3. **Bind directly on GitHub UI Configuration**:
   - Source Code Repo -> **Settings** -> **Webhooks** -> **Add webhook**
   - Definition Payload Hook: `https://your-ngrok-url.ngrok.io/webhook/github`
   - Strict Content Type Assertion: `application/json`
   - Auth Validation Secret: `a1b2c3d4e5f6...`
   - Subscriptions Active: **Pull requests**, **Pushes**
   - Conclude **Add webhook**

### Verification

1. **Github UI Interface test framework**:
   - Navigate to Tab -> **Recent Deliveries**
   - Click UI **Redeliver** -> Assert Status 200

2. **Internal Pod Log Traversal Test**:

   ```bash
   make logs
   # Should directly isolate string block: "GitHub signature verification passed"
   ```

---

## Completion Finalization

### Verified `.env` Layout Template Reference

```bash
# ===== REQUIRED BLOCKS =====
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
MEMOS_ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
GEMINI_API_KEY="AIzaSyA..."
VOYAGE_API_KEY="pa-..."
NGROK_AUTHTOKEN=2a...

# ===== OPTIONAL METADATA =====
GOOGLE_CALENDAR_CREDENTIALS=secrets/google-credentials.json
WEBHOOK_ENABLED=true
WEBHOOK_SECRET=a1b2c3d4e5f6...

# ===== ADVANCED EXPOSURES =====
GEMINI_TIMEZONE="Asia/Ho_Chi_Minh"
WEBHOOK_RATE_LIMIT_PER_MIN=60
WEBHOOK_ALLOWED_IPS=  # Empty defaults blanket accept all subnets
```

### Next steps Phase Shift Progression

1. **Basic Control Telemetry Validation**:

   ```
   Telegram String Source: "Deadline for Project Subnet C is March 15th"
   UI Receiver Assertion: Task logged mapping established.
   ```

2. **Search Entity Validation Call**:

   ```
   Telegram Protocol: /search deadline
   Agent ReAct Assertion: Extract 1 correlated hit mapped to context bounds...
   ```

---

## Troubleshooting Guide Parameters

### Telegram End Target Dead Timeouts

```bash
# Internal Process Telemetry Isolation Check
make logs

# Ensure local runtime target API reachability 
curl https://api.telegram.org/bot<TOKEN>/getMe
```

### Route 401 Disallow Constraints at Memos API

```bash
# Expired node payload bounds inside settings
# Target Token Refresh across Memos Core UI endpoint boundary
# Inject replacement string -> Execute `make restart`
```

### Route Empty Query RAG Index Bounds

```bash
# Reset internal embedding map space fully into Database Layer Vector Targets
go run scripts/backfill-embeddings/main.go
```

**Good Luck Deploying the Code!**
