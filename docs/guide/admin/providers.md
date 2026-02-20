# Providers

Providers are LLM backends that power Loom's agents. Loom supports any OpenAI-compatible API.

## Registering a Provider

```bash
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-provider",
    "name": "My vLLM Server",
    "type": "openai",
    "endpoint": "http://gpu-host:8000/v1",
    "model": "Qwen/Qwen2.5-Coder-32B-Instruct"
  }'
```

For cloud providers with API keys:

```bash
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "cloud-llm",
    "name": "Cloud Provider",
    "type": "openai",
    "endpoint": "https://api.example.com/v1",
    "model": "model-name",
    "api_key": "your-api-key"
  }'
```

API keys are encrypted at rest using the master password.

## Health Monitoring

Loom health-checks providers every 30 seconds via heartbeat workflows in Temporal. Provider states:

- **pending** -- Just registered, first heartbeat not yet received
- **healthy** -- Responding to chat completions successfully
- **failed** -- Last heartbeat detected an error

Check provider health:

```bash
curl http://localhost:8080/api/v1/providers | jq '.[] | {id, status, last_heartbeat_error}'
```

## Load Balancing

When multiple healthy providers are available, Loom uses weighted round-robin routing with health-aware failover. Configure routing weights via provider metadata.

## Supported Backends

- **vLLM** -- High-performance local inference (GPU required)
- **OpenAI API** -- GPT-4, GPT-3.5
- **Anthropic** -- Claude (via OpenAI-compatible proxy)
- **Any OpenAI-compatible endpoint** -- Ollama, LM Studio, text-generation-inference, etc.
