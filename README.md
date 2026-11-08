# sys-executor

Aplicativo desktop para Windows x64 que prepara **minutas** de Julgamento de PADS e de Relatório de Sindicância Regular, com IA 100% local. A especificação é `docs/rules/sys-executor_especificacao_consolidada_V4.md`. O planejamento, as decisões e os relatórios de cada fase ficam em `docs/planning/`.

## Estrutura

```
cmd/sys-executor/   aplicativo (Wails v3 + Vue 3)
cmd/rulescov/       governança da V4 (V4.lock e cobertura dos IDs de teste)
cmd/licencas/       verificação de licenças das dependências do produto
internal/           app (serviços da interface), workflow (tarefa), session (sessão efêmera),
                    prefs, erros (catálogo), pastas, platform/win
frontend/           interface (Vue 3 + TypeScript + Vite)
policies/           V4.lock e registro de revisões da V4
poc/                provas de conceito da Fase 0 (descartáveis)
```

## Desenvolvimento

Requisitos: Go 1.27, Node 24. O produto é só Windows; no Linux/WSL compila-se para Windows.

```bash
# Interface
cd frontend && npm ci && npm run build && npm test && cd ..

# Testes Go (Linux) e verificações de governança
go test -race ./internal/... ./cmd/rulescov/... ./cmd/licencas/...
go run ./cmd/rulescov -estrito
go run ./cmd/licencas

# Bindings da interface (após mudar serviços em internal/app)
GOOS=windows wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor

# Executável Windows (sem console, sem CGO)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
```

Executáveis gerados no WSL devem ser copiados para uma pasta do Windows antes de executar (não abrem a partir de `\\wsl.localhost\…`).

## Dados em disco

Tudo fica em `%LOCALAPPDATA%\sys-executor`: `trabalho\` (sessões efêmeras, apagadas ao final e na próxima abertura), `webview\` (esvaziada ao fechar e ao abrir) e `preferencias.json` (só configurações). Não há logs nem histórico de tarefas.
