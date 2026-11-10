# sys-executor

Aplicativo desktop para Windows x64 que prepara **minutas** de Julgamento de PADS e de Relatório de Sindicância Regular. A IA usada é a que o usuário escolher:
- a **Gemini API** (Google), com a chave do próprio usuário (V4 §47, ADR 009);
- o **ChatGPT**, pelo *Sign in with ChatGPT*, com o plano do próprio usuário (V4 §48, ADR 010).

Não há troca automática entre eles. A especificação é `docs/rules/sys-executor_especificacao_consolidada_V4.md`. O planejamento, as decisões e os relatórios de cada fase ficam em `docs/planning/`. Para retomar o trabalho, comece por `GUIA_CONTINUIDADE.md`.

## Entrada dos autos: peça por peça

- Depois de escolher PADS ou SR, cadastre as peças no assistente:
  - **PADS:** Memorando, com o acusado e os fatos; defesa prévia ou "não apresentada"; depoimentos; alegações finais ou "não apresentadas"; outros documentos;
  - **SR:** Portaria, com sindicados e ofendidos; depoimentos; outros documentos.
- Cada peça aceita um ou vários PDFs. O app junta os arquivos em memória e confere o conteúdo com o tipo declarado. Se houver divergência, ele pergunta e nunca troca o tipo sem avisar.
- **Enquadramento (PADS):** os dispositivos vêm do catálogo curado:
  - RDPM arts. 15 a 17, com a natureza fixa;
  - Estatuto arts. 29 e 32, com a natureza escolhida pelo usuário.
  
  Um dispositivo diferente do citado no Memorando exige confirmação expressa (§22.1).
- O cadastro existe só durante a tarefa (sessão efêmera). Não há rascunho salvo.

## IA: Gemini ou ChatGPT

- **Configuração (uma vez),** em *Configurações de IA*, escolha o serviço usado nas análises:
  - **Gemini API:** cole a chave criada em <https://aistudio.google.com/api-keys>. Ela é conferida no Google (pela listagem de modelos, sem gerar texto), cifrada com DPAPI para a sua conta do Windows e não é exibida de novo;
  - **ChatGPT:** leia o aviso de privacidade e clique em "Entrar com ChatGPT". O login acontece no navegador padrão e exige uma conta **com assinatura** (testado com o Plus). A credencial fica cifrada com DPAPI; o token de acesso, só em memória. "Desconectar" revoga a sessão e apaga a credencial.
  
  Depois, escolha e teste um modelo da lista da chave ou da conta.
- **Em cada tarefa:** antes da primeira etapa de IA, o app pede a autorização de envio, com o serviço, o modelo, as peças, uma estimativa de tokens e o resumo da pseudonimização.
  - No ChatGPT, também pergunta se há dados de **criança menor de 13 anos**. Se houver, o ChatGPT fica bloqueado na tarefa.
  - A autorização vale para um serviço e um modelo: se qualquer um mudar, o app pergunta de novo. Sem autorização, nada é enviado.
- **O que é enviado:** só os blocos de texto de cada etapa. O PDF não é enviado.
  - Nomes e dados pessoais reconhecidos, inclusive os das pessoas cadastradas, são trocados por marcadores e restaurados no computador.
  - A **barreira de privacidade** confere o corpo exato de cada requisição. Um dado pessoal conhecido que sobre bloqueia o envio. Um possível nome não reconhecido vai para a sua revisão, item a item.
  - É melhor esforço: o contexto (local, cargo, datas) pode permitir reidentificação.
  - Toda requisição usa `store=false` e nenhuma ferramenta.
- **Internet:** a análise por IA exige conexão. Sem internet, a tarefa para de forma recuperável ("Tentar novamente"). Extração, OCR, visualização e cálculos continuam funcionando.
- **Termos dos provedores:**
  - **Gemini:** o nível gratuito pode usar o conteúdo para melhorar produtos, com revisão humana. O nível pago (projeto com faturamento ativo) não usa o conteúdo para isso e o registra por tempo limitado, para prevenir abuso;
  - **ChatGPT:** o limite de uso do plano é compartilhado com os outros apps da conta, e o app não tenta contorná-lo. A retenção no modo *Sign in with ChatGPT* não estava documentada em 09/11/2026.
  
  O app não apaga registros dos provedores. A confirmação do usuário não substitui eventual autorização institucional para tratar os dados em serviço externo.

## Estrutura

```
cmd/sys-executor/   aplicativo (Wails v3 + Vue 3)
cmd/rulescov/       governança da V4 (V4.lock e cobertura dos IDs de teste)
cmd/licencas/       verificação de licenças das dependências do produto
cmd/autos-mapa/     diagnóstico de estrutura de PDF real (sem texto)
cmd/bench/          benchmark dos provedores (Gemini, ChatGPT, falso) e métrica de vazamento (só casos sintéticos)
internal/           app, workflow, session, prefs, erros, pastas, platform/win; ingest/* (PDF, OCR,
                    segmentação), sei; rules/* (motor e catálogo), domain/*; inference, policy,
                    analise (etapas de IA); credencial (DPAPI); chatgpt (Sign in with ChatGPT);
                    pseudonimo (marcadores e barreira); rules/enquadramento; pipeline (etapas da tarefa)
frontend/           interface (Vue 3 + TypeScript + Vite)
policies/           packs de instruções da IA, V4.lock e registro de revisões da V4
poc/                provas de conceito (descartáveis): Fase 0, chatgpt (SIWC), sei-mapa
```

## Desenvolvimento

Requisitos: Go 1.27, Node 24. O produto é só Windows; no Linux/WSL compila-se para Windows.

```bash
# Interface
cd frontend && npm ci && npm run build && npm test && cd ..

# Testes Go (Linux) e verificações de governança
go test -race ./internal/... ./cmd/rulescov/... ./cmd/licencas/... ./cmd/bench/...
go run ./cmd/rulescov -estrito
go run ./cmd/licencas

# Benchmark (SÓ casos sintéticos). Sem rede: métrica de vazamento e comparação das entradas
go run ./cmd/bench -provedor falso -entrada memoria,monolitica,pecas
# Gemini (chave de desenvolvimento em arquivo 0600, nunca em argumento)
SYSEXEC_GEMINI_API_KEY="$(tr -d '[:space:]' < ~/.config/sys-executor/gemini-dev.key)" \
  go run ./cmd/bench -provedor gemini -modelo gemini-… [-entrada monolitica,pecas] [-max-chamadas 18] [-saida rel.md]
# ChatGPT (conta do bench: credencial de desenvolvimento em ~/.config/sys-executor/chatgpt-dev.json, 0600)
go run ./cmd/bench -conectar-chatgpt          # abre o navegador; -estado-chatgpt, -desconectar-chatgpt
go run ./cmd/bench -provedor chatgpt -modelos
go run ./cmd/bench -provedor chatgpt -modelo <slug> [-pensamento low] [-entrada monolitica,pecas] [-max-chamadas 120]
# Teste de integração com a Gemini API
SYSEXEC_GEMINI_API_KEY=… SYSEXEC_GEMINI_MODELO=gemini-… go test -run Integracao -v ./internal/analise/

# Bindings da interface (após mudar serviços em internal/app)
GOOS=windows wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor

# Executável Windows (sem console, sem CGO)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
```

Executáveis gerados no WSL devem ser copiados para uma pasta do Windows antes de executar (não abrem a partir de `\\wsl.localhost\…`).

## Dados em disco

Tudo fica em `%LOCALAPPDATA%\sys-executor`:
- `trabalho\`: sessões efêmeras, com o cadastro e as peças cifrados; apagada ao final e na próxima abertura;
- `webview\`: esvaziada ao fechar e ao abrir;
- `credenciais\gemini.dpapi`: chave da Gemini API, cifrada com DPAPI;
- `credenciais\chatgpt.dpapi`: conta ChatGPT, cifrada com DPAPI, sem o token de acesso;
- `preferencias.json`: só configurações (serviço e modelos escolhidos); nunca chave ou token.

As duas credenciais são removíveis na tela de configurações e não migram para outro computador. Não há logs nem histórico de tarefas. A pasta `modelos\` de versões anteriores (GGUF da IA local) não é mais usada e pode ser apagada manualmente.
