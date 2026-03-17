# Space Wars 3002 - LLM Dialogue dialogue generation tool

Offline dialogue generator for Space Wars 3002. Reads vendor profiles from the database, generates flavour text using a local `llama.cpp` instance, validates the output, and stores the results. Designed to run as a batch job — no LLM calls happen during gameplay.

Currently generates vendor dialogue. Intended to expand to pirates, crew, and other NPCs over time.

---

## How it works

1. Fetches vendor profiles with `dialogue_generation_status = 'pending'` or `'failed'`
2. For each vendor, iterates a 22-scope generation matrix (greetings, inventory pitches, deal outcomes, farewells — keyed by interaction bucket, transaction context, and item category)
3. Builds a structured prompt from the vendor's personality, criminality, markup, and service type
4. Calls the local `llama.cpp` server and validates the output (word count, character limit, deduplication, meta-commentary rejection)
5. Stores accepted lines either directly to the database or via the PHP internal HTTP API
6. Marks each vendor `complete` or `failed`

---

## Requirements

- Go 1.21+
- A running [llama.cpp](https://github.com/ggerganov/llama.cpp) server with the OpenAI-compatible API enabled (`--api`)
- MariaDB access to the Space Wars 3002 database

---

## Setup

```bash
cp .env.example .env
# Edit .env with your database credentials, LLM URL, and model name
```

---

## Running

```bash
# Build the binary
make build

# Run directly
make run

# Process only N vendors
./vendor-dialogue-generator --rows=5

# Dry run (no LLM calls, no DB writes)
DRY_RUN=true make run
```

---

## Configuration

All configuration is via environment variables. Copy `.env.example` for a full list with defaults. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | — | Database host (required) |
| `DB_NAME` | — | Database name (required) |
| `LLM_BASE_URL` | — | llama.cpp server URL (required) |
| `LLM_MODEL` | — | Model name (required) |
| `LLM_TEMPERATURE` | `0.4` | Keep low — small models drift at high temps |
| `WORKER_COUNT` | `1` | Concurrent vendor workers |
| `BATCH_SIZE` | `5` | Vendors fetched per run |
| `GENERATION_VERSION` | `1` | Increment when prompts or model changes |
| `USE_HTTP_API` | `false` | Submit lines via PHP API instead of direct DB |
| `DRY_RUN` | `false` | Skip LLM calls and DB writes |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

---

## HTTP API mode

By default the service writes directly to the database (`USE_HTTP_API=false`). When `USE_HTTP_API=true` it instead polls the PHP backend for pending vendors and submits lines via the PHP internal API, which runs PHP-side validation before storage.

```
USE_HTTP_API=true
PHP_BASE_URL=http://localhost:8000
PHP_INTERNAL_TOKEN=your-shared-secret
```

---

## Development

```bash
make test    # run tests
make lint    # go vet
make tidy    # go mod tidy
```

Debug logging shows full prompt content and per-attempt details:

```bash
LOG_LEVEL=debug DRY_RUN=true make run
```

---

## Documentation

Full documentation lives in `docs/` (git submodule → `space-wars-3002-docs`):

| Document | Location |
|----------|----------|
| File and directory reference | `docs/reference/GO_SERVICE_STRUCTURE.md` |
| Go technical design | `docs/design/vendor_dialogue_go_technical_design.md` |
| Go + PHP joint design | `docs/design/vendor_dialogue_joint_design_go_php.md` |
| Phased architecture plan | `docs/design/vendor_dialogue_phased_architecture.md` |
| Implementation plan | `docs/implementation_plans/vendor_dialogue_phase2_go_implementation_plan.md` |
