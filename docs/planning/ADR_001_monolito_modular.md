# ADR 001 — Monólito modular em Go

- **Status:** Aceita (08/11/2026). Wails v3.1.0 confirmado na F0-01 (RELATORIO_FASE0 §3).
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
Um único executável Wails 3 em Go, com pacotes internos por responsabilidade (`workflow`, `session`, `sei`, `ingest`, `domain`, `rules`, `legal`, `inference`, `orchestrator`, `compose`, `validate`, `export`, `platform/win`). Interfaces existem **somente** onde há substituição real ou necessidade de teste: `OCREngine`, `PDFExtractor`, `InferenceEngine`, `Embedder`, `VectorIndex`, `SEIClient`, `Renderer`. O domínio (`domain`, `rules`) é Go puro, sem I/O.

## Alternativas consideradas
- **Clean/Hexagonal completa:** camadas e portas em todas as áreas, com custo de manutenção sem benefício para um app desktop de usuário único. Rejeitada como dogma.
- **Microsserviços/processos separados para cada módulo:** complexidade de empacotamento e de IPC injustificada. Os únicos processos separados são os sidecars (llama-server, tesseract, typst), por razões técnicas.

## Consequências
Testes simples, build único. A disciplina de dependências (o domínio não importa infraestrutura) é verificada por um lint de imports no CI.
