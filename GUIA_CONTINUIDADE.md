# Guia de continuidade — sys-executor

> **Comece por aqui.** Ponto de entrada para retomar o trabalho em qualquer máquina.
>
> - **Atualizado em:** 10/11/2026, depois da etapa 7 do plano D-11 (commit `0579327`) e deste guia.
> - **Branch:** `add-mode-apikey-oauth`. O push é feito pelo usuário. Não há merge com a `main`.
> - **Em uma linha:** as etapas 0 a 8 do plano D-11 estão **implementadas**. Para **concluir** o D-11 faltam os testes manuais do usuário (§6.1), uma bateria do bench (§6.2) e algumas decisões dele (§6.3).

---

## 1. Onde estamos

O aplicativo (Go + Wails 3 + Vue, para Windows) percorre:
1. a escolha do procedimento (PADS ou SR);
2. a obtenção dos autos **peça por peça**, por upload ou **pelo SEI**, com a seleção dos documentos no computador;
3. a pseudonimização e a **barreira de privacidade**;
4. a delimitação com **enquadramento manual**;
5. a análise das provas com o **Gemini** (chave do usuário) ou o **ChatGPT** (*Sign in with ChatGPT*, plano do usuário).

O processamento para em **"Consultando fontes jurídicas" (TAR-004)**, que é o fim previsto do escopo D-11. A minuta, o DOCX/PDF e o instalador ficam para as fases F5 a F8, fora deste escopo.

### 1.1 O que já foi testado e o que não foi
| Item | Automatizado | Manual no Windows (usuário) |
|---|---|---|
| ChatGPT no app (login, modelo, PADS até TAR-004) | ✅ | ✅ 09/11/2026, conta Plus, `gpt-6-luna` |
| Cadastro de peças PADS | ✅ | ✅ 09/11/2026, com ChatGPT |
| Cadastro de peças SR | ✅ | ❌ **pendente** (§6.1-B) |
| SEI com seleção no computador (etapa 7) | ✅ CAD-SEI-01 (simulado, pdfium real) | ❌ **pendente** (§6.1-A) |
| Gemini no app, de ponta a ponta | ✅ | ⚠️ **parcial**: a cota gratuita acabou no teste de 09/11/2026. A conta está no nível pago desde 10/11 (§6.1-C) |
| Bench Gemini `gemini-3.6-flash` (casos sintéticos) | ✅ dentro dos limiares; vazamento 0 | — |
| Bench ChatGPT `gpt-6-luna` | ⚠️ abaixo dos limiares de JSON 1ª e de resultado; vazamento 0 | Falta repetir a bateria depois da mudança 1(a) (§6.2) |
| Barreira de privacidade e métrica de vazamento | ✅ 0 nomes em todos os pedidos, nos dois provedores e nas três entradas | — |

---

## 2. Leia antes de mexer (nesta ordem)
1. **Este guia.**
2. `docs/planning/PLANO_CHATGPT_PECAS_PRIVACIDADE.md`: plano D-11 aprovado, etapas 0 a 8. O §4.8 (SEI) foi revisado e o §9 traz os critérios de aceitação.
3. `docs/rules/sys-executor_especificacao_consolidada_V4.md`: a especificação. **Prioridade absoluta: as regras jurídicas.**
   - Emenda 1: §47 (Gemini).
   - Emenda 2: §48 (ChatGPT, peças, enquadramento, barreira), com a **revisão de 10/11/2026 da §48.4 (SEI)**.
4. `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`: D-10 e D-11 (E1 a E9, com o **E2 revisado**).
5. `docs/planning/ADR_010_chatgpt_pecas_barreira.md`: as decisões de arquitetura do D-11.
6. Relatórios das etapas, em `docs/planning/`:
   - `RELATORIO_POC_CHATGPT.md`;
   - `RELATORIO_ETAPA2_BARREIRA.md`, `RELATORIO_ETAPA3_MULTIPROVEDOR.md`, `RELATORIO_ETAPA4_CHATGPT.md`;
   - `RELATORIO_ETAPA6_CADASTRO.md`, `RELATORIO_ETAPA7_SEI.md`, `RELATORIO_ETAPA8_BENCH.md`.
7. `CONTINUIDADE_MIGRACAO_GEMINI.md`: histórico da migração para o Gemini.

---

## 3. Regras de trabalho combinadas com o usuário
- **Autorização expressa por etapa ou fase.** Nada de código novo fora do que foi autorizado.
  - As etapas 0 a 8 do D-11 estão autorizadas e implementadas.
  - Correções decorrentes dos testes manuais e as melhorias do §6.4 cabem no D-11, mas é preciso **perguntar antes**.
  - As fases F5 a F8 exigem autorização própria.
- **Dúvida → perguntar** antes de decidir. Mudança no texto da V4 só com o texto aprovado pelo usuário, seguido de CHANGELOG e `go run ./cmd/rulescov -atualizar-lock`.
- **Commits:**
  - só quando o usuário pedir. Ele tem aprovado um commit por etapa;
  - **nunca** com coautor ou atribuição ao Claude/IA.
- **Git:** sem push (o push é do usuário), merge, rebase, reset, clean ou checkout destrutivo. Não apagar alterações do usuário. **Não tocar em `logo-cabecalho-docs.jpeg`** (e no `:Zone.Identifier`): são arquivos do usuário, não rastreados.
- **Dados:** só casos **sintéticos**. Nunca autos reais da PMRO, CPF, tokens ou chaves no repositório, em logs, no chat ou em argumentos de linha de comando. A tabela de pseudônimos nunca vai à IA.
  - Saídas do `sei-mapa` coladas no chat trazem só a forma dos textos; o usuário as revisa antes.
- **Máquina do usuário:** **não rodar GPU/Vulkan**, que já derrubou a máquina. Use só a CPU; o Chrome de teste roda com `--disable-gpu`.
- **Prompts de referência:** os golden (`internal/policy/testdata/golden`) **não podem mudar**.
- **Ao fim de cada etapa:** atualizar este guia e o relatório da etapa.
- **Bibliotecas:** consultar a documentação pelo Context7 ou pela web antes de usar uma API externa.

---

## 4. Preparar a máquina nova

### 4.1 Ferramentas (WSL/Linux; o executável é gerado para Windows por compilação cruzada)
- **Go 1.27.3**, a versão do CI e do `gofmt` dele. O `go.mod` pede 1.26.0.
  ```bash
  go install golang.org/dl/go1.27.3@latest && go1.27.3 download
  export PATH=$HOME/sdk/go1.27.3/bin:$PATH      # em todo terminal de trabalho
  ```
- **Node 24** com npm.
- **Wails 3 CLI v3.1.0:** `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0` (fica em `$(go env GOPATH)/bin`).

```bash
git clone <repo> && cd sys-executor && git checkout add-mode-apikey-oauth
cd frontend && npm ci && npm run build && cd ..
go build $(go list ./... | grep -v '/cmd/sys-executor$')   # tudo, menos o app
GOOS=windows go vet ./...                                   # inclui o app
go test ./internal/... ./cmd/bench/...
```
- `go build ./...` e `go test ./cmd/sys-executor` **não compilam no Linux**, porque o app exige GTK4/WebKit. Não é defeito.

### 4.2 O que NÃO está no repositório (refazer em cada máquina)
| Item | Onde fica | Como refazer |
|---|---|---|
| Chave de desenvolvimento do Gemini (bench) | `~/.config/sys-executor/gemini-dev.key` (0600) | No terminal do usuário, **nunca no chat**:<br>`install -d -m 700 ~/.config/sys-executor`<br>`read -rs K; printf %s "$K" > ~/.config/sys-executor/gemini-dev.key; unset K`<br>`chmod 600 ~/.config/sys-executor/gemini-dev.key`<br>Usar com `SYSEXEC_GEMINI_API_KEY="$(tr -d '[:space:]' < ~/.config/sys-executor/gemini-dev.key)"`.<br>Validar sem gerar texto: `go run ./cmd/bench -provedor gemini -modelos`. |
| Conta ChatGPT do bench | `~/.config/sys-executor/chatgpt-dev.json` (0600) | `go run ./cmd/bench -conectar-chatgpt` abre o navegador do Windows e dá 5 min para entrar.<br>`-estado-chatgpt` e `-desconectar-chatgpt`. |
| Credenciais do app (Windows) | `%LOCALAPPDATA%\sys-executor\credenciais\` (`gemini.dpapi`, `chatgpt.dpapi`, DPAPI) | **Não migram.** Em *Configurações de IA*, cadastre a chave do Gemini e conecte o ChatGPT, que exige conta **com assinatura**; conta gratuita recebe `invalid_grant`. |
| Tesseract (OCR) do teste manual | `componentes\tesseract\` ao lado do `.exe` (`tesseract.exe`, DLLs, `tessdata\por.traineddata` e `osd.traineddata`) | Copiar a pasta de uma máquina que já a tenha (`~/.cache/sys-executor-dev/bin/win-tesseract/`) ou instalar o Tesseract 5.4 da UB Mannheim e copiar a pasta.<br>Sem ele, página escaneada termina em OCR-001. PDF com texto funciona sem OCR.<br>Não é o binário sem GPL previsto no D-09.7 para o instalador. |
| Chrome de desenvolvimento (só para `poc/sei-mapa`, `go test .`) | `~/.cache/sys-executor-dev/bin/chrome/chrome-linux64/chrome` ou `SEIMAPA_CHROME` | Opcional. Sem ele, o teste é pulado. |
| PDFs sintéticos do teste manual | pasta de teste no Windows | `SAIDA_PDF=/mnt/c/Users/<usuário>/Documents/testes-sys-executor go test -count=1 -run TestGerarPDFsSinteticos ./internal/testutil/pdfsint`<br>Gera `pecas-pads-sintetico/`, `pecas-sr-sintetico/` e os consolidados `*-texto.pdf`. |

**Faturamento do Gemini.** O nível é pago (pré-pago) desde 10/11/2026.
- A conta de faturamento usa primeiro os créditos do GCP, mas exige saldo pré-pago acima de zero.
- O perfil de pagamentos pertence a **outra conta Google do usuário**, do tipo pessoa física, que não aceita outros usuários. Para comprar créditos, entre no AI Studio **com essa outra conta**.
- No nível pago, prompts e respostas não são usados para melhorar os produtos do Google.

### 4.3 Gerar e testar o executável
```bash
GOOS=windows $(go env GOPATH)/bin/wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor  # se mudar serviços em internal/app
cd frontend && npm run build && cd ..
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o /mnt/c/Users/<usuário>/Documents/testes-sys-executor/sys-executor.exe ./cmd/sys-executor
```
- **Feche o app antes de substituir o `.exe`.** Confira com `tasklist.exe | grep -i sys-executor`.
- Pasta de teste usada até aqui: `C:\Users\diogo\Documents\testes-sys-executor\`, com o `.exe`, `componentes\tesseract\`, os PDFs sintéticos e o `sei-mapa.exe`.
- **Modo diagnóstico:** mostra a causa técnica, com os dígitos mascarados. No PowerShell: `$env:SYSEXEC_DIAGNOSTICO = "1"; .\sys-executor.exe`.

### 4.4 Verificação completa (antes de cada commit)
```bash
gofmt -l ./internal ./cmd                                   # vazio
GOOS=windows go vet ./...
go vet ./internal/... ./cmd/rulescov/... ./cmd/licencas/... ./cmd/bench/...
go test ./internal/... ./cmd/bench/...                      # -race nos pacotes alterados
cd frontend && npx vitest run && npm run build && cd ..     # vitest 31/31; vue-tsc + vite
go run ./cmd/rulescov -estrito                              # 202 regras; 124/124 IDs da V4
go run ./cmd/bench -provedor falso -entrada memoria,monolitica,pecas   # vazamento 0, sem rede
(cd poc/sei-mapa && go test .)                              # só se mexer no sei-mapa
go run ./cmd/licencas                                       # 45 módulos Go, 40 npm
git diff --stat -- internal/policy/testdata                 # golden: vazio
```

---

## 5. O que já foi feito

### 5.1 Antes do plano D-11
- Fases F0 a F4 do plano original: base, SEI, ingestão e OCR, motor determinístico, catálogo de regras e etapas de IA com schemas e validadores (`RELATORIO_FASE0..4.md`).
- D-10 (V4 §47): a IA local foi removida, e a análise passou a usar a Gemini API com a chave do usuário.

### 5.2 Plano D-11 (09 e 10/11/2026)
| Etapa | Commit | Conteúdo | Situação |
|---|---|---|---|
| 0 | `51af5e5`, `c0b2e57` | Plano, D-11, Emenda 2 (V4 §48), `V4.lock` | Concluída |
| 1 | `e0dfdeb` | PoC do *Sign in with ChatGPT* em Go (`poc/chatgpt`) | Concluída |
| 2 | `ab14035` | Barreira de privacidade: camada 1 no pedido, camada 2 nos bytes (IA-014), revisão item a item; correção do 400 do Gemini | Concluída; **400 confirmado como corrigido** no bench de 10/11 |
| 3 | `0c5335c` | Dois provedores sem troca automática; autorização presa a provedor e modelo; preferências v3 | Concluída |
| 4 | `e53a815`, `78bc80e` | ChatGPT no app (OAuth com PKCE, DPAPI, Responses API com SSE, `store:false`), bloqueio de dados de criança (IA-021) | Testada pelo usuário |
| 5 | `76df133` | Catálogo de enquadramento (130 dispositivos) e formulário PADS-ESC-001 com a §22.1 | Concluída; **curadoria a revisar** (§6.3) |
| 6 | `e6f40d6` | Cadastro peça por peça, junção em memória, peças declaradas conferidas pelo conteúdo, assistente `Montagem.vue` | PADS testado pelo usuário; **SR pendente** |
| — | `f001fdd` | Guia e gerador versionado dos PDFs sintéticos | — |
| 8 | `4ec155d` | Bench:<br>• `-provedor gemini\|chatgpt\|falso`, `-entrada memoria,monolitica,pecas`, `-max-chamadas`, `-amostras`;<br>• conta ChatGPT própria;<br>• métrica de vazamento sobre o corpo exato.<br>ADR 010, README, plano de testes §9, `sei-mapa` ampliado. | Concluída. Gemini dentro dos limiares; ChatGPT abaixo (§6.2) |
| 7 | `0579327` | SEI com seleção no computador (opção A, E2 revisado):<br>• o PDF de todos os documentos é separado pelo carimbo;<br>• os documentos vão ao cadastro, e o usuário escolhe os de cada peça.<br>V4 §48.4 revisada; TRV-ING-008; CAD-SEI-01.<br>Na extração, pessoa sem nome é descartada, sem recusa. | Implementada; **teste real pendente** (§6.1-A) |
| — | (este commit) | Este guia reescrito para a continuidade | — |

### 5.3 Onde está cada coisa (código)
- **Provedores de IA:**
  - Gemini: `internal/inference/gemini*.go`;
  - ChatGPT: `internal/inference/chatgpt*.go` e `internal/chatgpt/` (conta e OAuth);
  - seletor: `internal/app/fonte_ia.go`;
  - serviço da interface: `internal/app/ia*.go`.
- **Privacidade:** `internal/pseudonimo/` (mapa, barreira, motor) e `internal/pipeline/{envio,privacidade}.go`.
- **Cadastro de peças:**
  - `internal/workflow/cadastro.go`;
  - `internal/ingest/pdf` (`Juntador`);
  - `internal/ingest/segment/declaradas.go`;
  - `internal/pipeline/cadastro.go`;
  - `internal/app/cadastro.go`;
  - `frontend/src/views/Montagem.vue`, `components/ArquivosPeca.vue`, `components/EditorEnquadramento.vue`.
- **SEI → cadastro (etapa 7):**
  - `internal/pipeline/sei.go` (`separarSEI`, `sugestaoPeca`);
  - `internal/workflow/documentos_sei.go`;
  - `internal/ingest/pdf` (`Separador`) e `internal/ingest/autos` (`LerTextoNativo`);
  - na interface, "Documento do SEI" em `ArquivosPeca.vue`.
  - O cliente e a guarda do SEI (`internal/sei/`) **não mudaram**: continuam gerando "Todos os documentos".
- **Enquadramento:** `internal/rules/enquadramento/` e `internal/pipeline/{enquadramento,delimitar}.go`.
- **Validadores das etapas de IA:** `internal/analise/` (na extração, pessoa sem nome é descartada em `extracao.go`).
- **Bench:**
  - `cmd/bench/` (`provedores.go`, `vazamento.go`, `entrada.go`);
  - casos sintéticos em `internal/testutil/casos`, com `Pessoas`/`Identificadores`.
- **Mapeamento do SEI (observação):** `poc/sei-mapa/`, com o comando `mapear` e as linhas `[post-gerar]`. O roteiro está no `LEIA-ME.md`.
- **Governança:**
  - `internal/rules/catalog/*.yaml`;
  - `docs/planning/MATRIZ_REQUISITOS_V4.md`;
  - `docs/planning/PLANO_TESTES_ACEITACAO.md`;
  - `policies/CHANGELOG.md`;
  - `policies/V4.lock`.

---

## 6. O que falta para concluir o D-11 (em ordem)

### 6.1 Testes manuais do usuário no Windows (bloqueiam a conclusão)
Antes, gere o `.exe` (§4.3) e os PDFs sintéticos e copie o Tesseract (§4.2). Em *Configurações de IA*, conecte o ChatGPT e cadastre a chave do Gemini.

**A. SEI real (etapa 7).** Use um processo de teste ou já encerrado.
1. Escolha PADS (ou SR) e depois **"Buscar no SEI"**. Faça o login e o 2FA e informe o processo.
2. O app gera e baixa "Todos os documentos", como antes, e **abre o cadastro** com o aviso "separado em N documento(s)".
3. Em cada peça, anexe os documentos em **"Documento do SEI"**. Os de tipo sugerido igual ao da peça aparecem primeiro.
4. Na conferência, veja a contagem de escolhidos e não escolhidos. Conclua.
5. **Conferir:**
   - N é igual ao número de documentos da árvore do processo;
   - títulos e números SEI estão corretos;
   - a sugestão de tipo é razoável;
   - só os documentos escolhidos aparecem nos autos;
   - o processamento segue até TAR-004.
6. **Se algo divergir,** descreva sem dados reais: quantos documentos eram esperados e quantos apareceram, e a forma dos títulos. O ajuste é em `segment.Segmentar` ou em `autos.LerCarimbos`, **nunca na guarda**.

**B. SR com upload** (`pecas-sr-sintetico/`): Portaria com sindicados e ofendidos (ou "sem ofendido"), depoimentos e conferência, até TAR-004. Conferir os alertas de sindicados divergentes e a pergunta de conexão.

**C. Gemini no app, de ponta a ponta** (`pecas-pads-sintetico/`): escolha a Gemini em *Configurações de IA*. Faça a confirmação de envio, a delimitação e a análise até TAR-004. Ele já roda no nível pago, sem a cota de 20 por dia.

**D. Opcional:** repetir um caso com o ChatGPT `gpt-6.1-sol` (decisão do usuário: "talvez, num teste manual").

### 6.2 Bench pendente (só casos sintéticos)
- **Repetir a bateria completa do ChatGPT** depois da mudança 1(a) (pessoa sem nome descartada). Na conferência rápida do pads-01, o JSON 1ª ficou em 7/8. Registre o resultado no `RELATORIO_ETAPA8_BENCH.md`.
  ```bash
  S=/tmp/bench && mkdir -p $S
  go run ./cmd/bench -provedor chatgpt -modelo gpt-6-luna -entrada monolitica,pecas -max-chamadas 120 -saida $S/chatgpt.md
  SYSEXEC_GEMINI_API_KEY="$(tr -d '[:space:]' < ~/.config/sys-executor/gemini-dev.key)" \
    go run ./cmd/bench -provedor gemini -modelo gemini-3.6-flash -entrada monolitica,pecas -max-chamadas 120 -saida $S/gemini.md
  ```
  Cada bateria completa usa cerca de 95 chamadas no ChatGPT e 75 no Gemini. No ChatGPT, o limite de 5 h do Plus é compartilhado com os outros apps.
- **Limiares (plano de testes §9), para cada provedor:**
  - delimitação ≥ 90%;
  - JSON 1ª ≥ 90%;
  - resultado ≥ 75%;
  - SEG-INJ igual ao caso limpo em 5/5;
  - 0 violações e vazamento 0.
- **O ChatGPT `gpt-6-luna` ficou abaixo no resultado** (11/18 a 12/18). É julgamento do modelo: o pads-01 sai IMPROCEDENTE. O usuário decidiu não trocar de modelo por enquanto.
- **Proposta ainda não decidida:** repetir cada caso 3 vezes e usar a mediana antes de afirmar qualidade.

### 6.3 Decisões pendentes do usuário
1. **Revisar a curadoria do catálogo de enquadramento** (`docs/planning/CATALOGO_ENQUADRAMENTO_REVISAO.md`) e registrar a aprovação no `policies/CHANGELOG.md`.
2. **Interrogatório do acusado no PADS** (plano §8.1). Hoje vai em "Outros documentos", inclusive na sugestão do SEI. Tipo próprio?
3. **Imagens (JPG/PNG) como peça:** hoje só PDF. Converter, se o usuário quiser.
4. **ChatGPT abaixo do limiar:** aceitar como está (o usuário escolhe o modelo) ou investigar mais (outro modelo ou `-pensamento high`).

### 6.4 Melhorias pequenas identificadas (dentro do D-11; perguntar antes)
- **Conexão por fato:** usar o agrupamento por fato do cadastro para pré-selecionar os grupos na pergunta de conexão. O motor não muda e o usuário confirma.
- **Conferência da SR:** cruzar os ofendidos do cadastro com a leitura da Portaria. Os sindicados já são cruzados.
- **Pânico nos workers da extração:** `recover` → INT-001 (problema conhecido 2).

### 6.5 Critérios de conclusão do D-11 (plano §9) e situação
| Critério | Situação |
|---|---|
| OAuth oficial em Go; escopos; tokens com DPAPI e fora do Vue e de logs; Responses API com `store:false`/`stream:true`; sucesso só com `completed`; modelos dinâmicos | ✅ Feito e testado |
| Gemini preservado, sem fallback; nenhuma IA local | ✅ Bench ok; ⚠️ falta o teste manual completo no app (§6.1-C) |
| Cadastro PADS/SR peça por peça | ✅ PADS; ⚠️ SR manual pendente (§6.1-B) |
| SEI só com os documentos escolhidos, sem download manual (V4 §48.4 revisada) | ✅ Implementado; ⚠️ teste real pendente (§6.1-A) |
| Enquadramento por catálogo, divergência da §22.1 confirmada, conexão pelo motor | ✅ Feito; ⚠️ curadoria a revisar (§6.3-1) |
| Barreira bloqueia identificador conhecido; mapa nunca sai do computador; restauração determinística | ✅ Feito; vazamento 0 no bench |
| Regras jurídicas, cálculos e golden inalterados; `rulescov -estrito` verde | ✅ |

**O D-11 estará concluído quando** os três ⚠️ de teste manual passarem, a curadoria for aprovada e o usuário decidir sobre o ChatGPT abaixo do limiar (§6.3-4). Ao concluir:
- registrar a conclusão no `REGISTRO_DUVIDAS_DECISOES.md` (histórico) e neste guia;
- o usuário decide o merge com a `main`.

### 6.6 Depois do D-11 (cada fase exige autorização própria)
- **F5:** biblioteca jurídica. Decidir antes entre busca lexical e embeddings (V4 §47.7).
- **F6/F7:** composição da minuta, DOCX/PDF (Typst) e revisão humana. O motor de cálculo existe, mas o pipeline ainda não o chama. Precisa dos materiais institucionais (D-04).
- **F8:** instalador (certificado, WiX, Tesseract sem GPL, Liberation Serif).

---

## 7. Como retomar com o Claude
Na nova sessão, cole algo como:

> Projeto sys-executor, branch `add-mode-apikey-oauth`. Leia o `GUIA_CONTINUIDADE.md` e os documentos que ele indica antes de qualquer ação. Siga as regras do §3. Prepare esta máquina pelo §4 e rode a verificação completa (§4.4). Depois me diga o que falta configurar. Em seguida, vamos pelo §6: <o que quer fazer agora, por exemplo, "acabei o teste A do §6.1; o resultado foi…">.

---

## 8. Problemas conhecidos
1. `go build ./...` e `go test ./cmd/sys-executor` não compilam no Linux (GTK). Use o vet do Windows (§4.1).
2. Os pânicos dentro dos workers da extração, fora de `executarSeguro`, ainda derrubam o processo (§6.4).
3. Uma tarefa completa usa dezenas de chamadas de IA. O ChatGPT Plus tem limite de 5 h compartilhado com os outros apps da conta.
4. Dez commits antigos da branch, anteriores à regra, têm `Co-Authored-By`. O usuário foi informado; nada foi reescrito.
5. O `gofmt` do CI já acusou `internal/ingest/segment/carimbo_test.go`. Com o Go 1.27.3 está limpo. Confira no próximo CI.
6. O backend ainda tem `ReceberUpload` (PDF único) só para os testes legados. A interface não oferece esse caminho (E6).
7. **Carimbo do SEI no texto:** o nome do documento na árvore, impresso no carimbo de cada página, faz parte do texto da página. Ele segue com as evidências e passa pela pseudonimização (já era assim desde a Fase 2).
8. Pelo SEI, o processo inteiro é baixado e separado, o que é mais lento em processos grandes. Os documentos não escolhidos ficam só na sessão cifrada até o fim da tarefa.
