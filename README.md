# sys-executor

Aplicativo desktop para Windows x64 que prepara **minutas** de Julgamento de PADS e de Relatório de Sindicância Regular, com IA pela **Gemini API** (Google), usando a chave do próprio usuário (emenda V4 §47, ADR 009). A especificação é `docs/rules/sys-executor_especificacao_consolidada_V4.md`. O planejamento, as decisões e os relatórios de cada fase ficam em `docs/planning/`.

## IA: Gemini API

- **Configuração (uma vez):** em *Configurações de IA*, cole a chave criada em <https://aistudio.google.com/api-keys>. Ela é conferida no Google (listagem de modelos, sem gerar texto) antes de ser guardada, cifrada com DPAPI para a sua conta do Windows, e não é exibida de novo. Depois, escolha e teste um modelo da lista.
- **Em cada tarefa:** antes da primeira etapa de IA, o app pede a autorização de envio, com o modelo, as peças, uma estimativa de tokens e o resumo da pseudonimização. Sem autorização, nada é enviado.
- **O que é enviado:** só os blocos de texto de cada etapa (o PDF não é enviado), com nomes e dados pessoais reconhecidos trocados por marcadores. É melhor esforço: nomes sem posto, papel processual ou rótulo podem passar. Toda requisição usa `store=false` e nenhuma ferramenta.
- **Internet:** a análise por IA exige conexão. Sem internet, a tarefa para de forma recuperável ("Tentar novamente"); extração, OCR, visualização e cálculos seguem funcionando.
- **Termos do provedor:** pelos termos da Gemini API, o nível gratuito pode usar o conteúdo para melhorar produtos, com revisão humana; o nível pago (projeto com faturamento ativo) não usa o conteúdo para isso e o registra por tempo limitado para prevenir abuso. O app não apaga registros do provedor. A confirmação do usuário não substitui eventual autorização institucional para tratar os dados em serviço externo.

## Estrutura

```
cmd/sys-executor/   aplicativo (Wails v3 + Vue 3)
cmd/rulescov/       governança da V4 (V4.lock e cobertura dos IDs de teste)
cmd/licencas/       verificação de licenças das dependências do produto
cmd/autos-mapa/     diagnóstico de estrutura de PDF real (sem texto)
cmd/bench/          benchmark dos modelos Gemini (só casos sintéticos)
internal/           app, workflow, session, prefs, erros, pastas, platform/win; ingest/* (PDF, OCR,
                    segmentação), sei; rules/* (motor e catálogo), domain/*; inference, policy,
                    analise (etapas de IA); credencial (chave DPAPI); pseudonimo;
                    pipeline (etapas da tarefa)
frontend/           interface (Vue 3 + TypeScript + Vite)
policies/           packs de instruções da IA, V4.lock e registro de revisões da V4
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

# Benchmark e teste de integração com a Gemini API (chave de desenvolvimento; SÓ casos sintéticos)
SYSEXEC_GEMINI_API_KEY=… go run ./cmd/bench -modelo gemini-… [-pseudonimizar=false] [-saida rel.md]
SYSEXEC_GEMINI_API_KEY=… SYSEXEC_GEMINI_MODELO=gemini-… go test -run Integracao -v ./internal/analise/

# Bindings da interface (após mudar serviços em internal/app)
GOOS=windows wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor

# Executável Windows (sem console, sem CGO)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
```

Executáveis gerados no WSL devem ser copiados para uma pasta do Windows antes de executar (não abrem a partir de `\\wsl.localhost\…`).

## Dados em disco

Tudo fica em `%LOCALAPPDATA%\sys-executor`: `trabalho\` (sessões efêmeras, apagadas ao final e na próxima abertura), `webview\` (esvaziada ao fechar e ao abrir), `credenciais\gemini.dpapi` (chave da Gemini API cifrada com DPAPI; removível na tela de configurações) e `preferencias.json` (só configurações, inclusive o modelo escolhido; nunca a chave). Não há logs nem histórico de tarefas. A pasta `modelos\` de versões anteriores (GGUF da IA local) não é mais usada e pode ser apagada manualmente.
