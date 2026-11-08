# PoCs da Fase 0 (descartáveis, sem dados reais)

Resultados e decisões: `docs/planning/RELATORIO_FASE0.md`. Artefatos pesados (modelos, binários, fixtures geradas, logs) ficam em `~/.cache/sys-executor-dev/` (fora do git, com SHA256SUMS).

| Pasta | Tarefa | Como rodar |
|---|---|---|
| `fixtures/` | Fixture sintética PADS-01 (Typst) + gabarito | `typst compile pads01.typ pads01.pdf` (`--input filler=N` para autos longos) |
| `wails/` | F0-01 Wails 3.1 + Vue | `GOOS=windows wails3 generate bindings && (cd frontend && npm i && npm run build) && GOOS=windows go build -o wails-poc.exe .`; no Windows, de pasta local: `SYSEXEC_SELFTEST=1 wails-poc.exe` |
| `llm/` | F0-02/F0-03 sidecar + avaliação de modelos | `go run . -server <llama-server> -model <gguf> -fixtures <dir com pads01-p*.txt> [-reasoning on]` |
| `sidecar-win/` | F0-02 Job Object (Windows) | `GOOS=windows go build -o sidecar-poc.exe .`; `sidecar-poc.exe -server llama-server.exe -model m.gguf [-memlimit MB] [-hold]` |
| `pdfium/` | F0-04 go-pdfium wasm | `go run . <arquivo.pdf> [workers]` |
| `docmodel/` | F0-06 DocModel → DOCX + `.typ` | `go run . -out <dir>` e `typst compile <dir>/minuta.typ` |
| `rag/` | F0-07 SQLite FTS5 + embeddings | `go run . index -lib docs/law_library -db lib.db -server <llama-server> -model <emb.gguf>`; `go run . eval -db lib.db … -queries consultas.json` |
| `sei-mapa/` | F0-08 mapeamento SEI (usuário executa) | ver `sei-mapa/LEIA-ME.md` |

**GPU:** não rodar Vulkan/CUDA nesta máquina sem decisão do usuário (relatório §2).
