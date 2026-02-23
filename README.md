# Autonomous Task Management

AI-powered task management system with Telegram interface, Memos storage, and Google Calendar integration.

## Architecture

- **Frontend**: Telegram Bot (voice + text)
- **Backend**: Golang orchestrator
- **Storage**: Memos (local, markdown-based)
- **Memory**: Qdrant (vector database)
- **Scheduler**: Google Calendar

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Telegram Bot Token (get from @BotFather)
- Google Calendar API credentials (see `documents/google-calendar-setup.md`)

### Setup

1. Clone repository:

   ```bash
   git clone <repo-url>
   cd autonomous-task-management
   ```

2. Configure environment:

   ```bash
   cp .env.example .env
   # Edit .env with your tokens
   nano .env
   ```

3. Start services:

   ```bash
   docker compose up -d
   ```

4. Verify setup:

   ```bash
   bash scripts/verify-setup.sh
   ```

5. Access services:
   - Memos: <http://localhost:5230>
   - Qdrant Dashboard: <http://localhost:6333/dashboard>
   - Backend API: <http://localhost:8080>
   - Health Check: <http://localhost:8080/health>

### Initial Configuration

1. Initialize Memos tags:

   ```bash
   bash scripts/init-memos.sh
   ```

2. Follow the instructions to set up your tag schema in Memos UI

3. Configure Google Calendar (first time only):
   - See `documents/google-calendar-setup.md`
   - Authorize when prompted

## Development

### Build locally

```bash
make build
```

### Run tests

```bash
make test
```

### View logs

```bash
docker compose logs -f backend
```

## Project Structure

See `documents/convention/` for detailed coding conventions.

## Roadmap

- [ ] Phase 1: Infrastructure Setup
- [ ] Phase 2: Core Engine (Telegram + LLM + Bulk Processing)
- [ ] Phase 3: RAG & Agent Tools
- [ ] Phase 4: Automation & Webhooks
