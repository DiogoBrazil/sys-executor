# ADR 003 — Inferência local com sidecar llama-server

- **Status:** Aceita (08/11/2026), confirmada na F0-02 em CPU (`--no-webui`, `--api-key`, Job Object, `json_schema`). GPU apenas após sonda segura (D-08, D-09.1); limite de memória do Job aborta com `0xC0000409` e o orçamento soma o GGUF mapeado (D-09.2).
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
O app executa `llama-server` (llama.cpp, MIT) como processo filho:
- em loopback, porta aleatória, API key aleatória, sem web UI nem arquivo de log;
- dentro de um Windows Job Object `KILL_ON_JOB_CLOSE`;
- com builds x64 CPU e Vulkan empacotados e CUDA como download opcional;
- com versão fixada e SHA-256 conferido.

Saída estruturada via `response_format: json_schema`. Embeddings em uma segunda instância, sob demanda. (Validado na F0-02: flags `--no-webui`, `--api-key`, `--log-disable` na b12207; RSS de 2,6 a 6,6 GB conforme o modelo; estável em CPU.)

## Alternativas consideradas
- **Bindings in-process (CGO / purego):** um crash do modelo derruba a UI, e o build Windows fica mais complexo.
- **Ollama:** serviço adicional, telemetria/atualização próprias e menor controle sobre schema e processo.
- **Provedor em nuvem:** vedado na V1 (V4 §36).

## Consequências
Isola falhas e OOM e permite trocar o backend (CPU/Vulkan/CUDA) sem recompilar o app. Exige um gerenciador de ciclo de vida (watchdog, health check, timeouts).
