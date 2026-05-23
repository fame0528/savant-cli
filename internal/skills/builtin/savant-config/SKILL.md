---
name: savant-config
description: Complete reference for Savant CLI configuration file format and options.
license: MIT
compatibility: savant-cli>=0.1.0
---

# Savant CLI Configuration Reference

## Config File Location

`~/.savant/config.json` - Global configuration file.

## Config Structure

```json
{
  "providers": [...],
  "default_provider": "opengateway",
  "default_model": "mimo-v2-pro",
  "smart_routing": {...},
  "theme": "cyberpunk",
  "permissions": {...},
  "max_turns": 100,
  "auto_compact": true,
  "auto_compact_threshold": 0.8
}
```

## Providers

Each provider in the `providers` array:

```json
{
  "name": "opengateway",
  "base_url": "https://opengateway.gitlawb.com/v1",
  "api_key": "your-api-key",
  "model": "mimo-v2-pro",
  "enabled": true
}
```

Supported providers:
- `opengateway` - Gitlawb OpenGateway (MiMo V2 Pro, free)
- `9router` - Local 9router gateway
- `ollama` - Local Ollama server
- Any OpenAI-compatible API

## Smart Routing

Routes simple messages to cheap models, complex messages to strong models:

```json
{
  "smart_routing": {
    "enabled": true,
    "simple_model": "mimo-v2-pro",
    "strong_model": "mimo-v2-pro",
    "simple_max_chars": 160,
    "simple_max_words": 28
  }
}
```

Keywords that trigger strong routing: plan, design, architect, refactor, debug, investigate, analyze, implement, optimize, review, audit, diagnose.

## Environment Variables

- `OPENAI_API_KEY` - API key for OpenAI-compatible providers
- `OLLAMA_HOST` - Ollama server URL (default: http://localhost:11434)
- `NINEROUTER_URL` - 9router gateway URL (default: http://localhost:20128)
