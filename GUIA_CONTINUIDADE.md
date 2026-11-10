# Guia de continuidade — sys-executor

> **Comece por aqui.** Este arquivo é o ponto de entrada para retomar o trabalho em qualquer máquina. Atualizado em 10/11/2026, ao fim da etapa 8 do plano D-11.
>
> **Branch:** `add-mode-apikey-oauth`. O push será feito pelo usuário; não há merge com a `main`.

---

## 1. Estado em uma frase

O aplicativo (Go + Wails 3 + Vue, para Windows) faz o seguinte, da escolha do procedimento até a análise das provas, e para em "Consultando fontes jurídicas" (TAR-004, fim previsto deste escopo):
- recebe os autos **peça por peça**;
- pseudonimiza e passa pela **barreira de privacidade**;
- delimita o objeto com **enquadramento manual**;
- analisa as provas com o **Gemini** (chave) ou o **ChatGPT** (*Sign in with ChatGPT*, plano do usuário).

O usuário testou no Windows, em 09/11/2026, o cadastro de peças com o ChatGPT (`gpt-6-luna`) até esse ponto: autorização, imputações, mídias, resultados, conexão, comportamento e histórico.

Em 10/11/2026 (etapa 8), o bench rodou nos dois provedores com os casos sintéticos, nas entradas monolítica e por peças (`RELATORIO_ETAPA8_BENCH.md`):
- **vazamento zero** nos dois provedores;
- **Gemini dentro dos limiares**, e o 400 da análise está confirmado como corrigido;
- **ChatGPT abaixo dos limiares** de JSON na 1ª tentativa e de resultado. A causa principal foi achada: a mensagem do validador para pessoa sem nome. A correção depende de decisão do usuário.

Falta a etapa 7 (SEI), que depende do mapeamento do usuário.

---

## 2. Leia antes de mexer (nesta ordem)

1. **Este guia.**
2. `docs/planning/PLANO_CHATGPT_PECAS_PRIVACIDADE.md`: plano D-11 aprovado (etapas 0 a 8).
3. `docs/rules/sys-executor_especificacao_consolidada_V4.md`: a especificação. **Prioridade absoluta: as regras jurídicas.** Emendas: §47 (Gemini) e §48 (ChatGPT, peças, enquadramento, barreira).
4. `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`: D-10 e D-11 (decisões E1 a E9).
5. Relatórios por etapa em `docs/planning/`:
   - `RELATORIO_POC_CHATGPT.md`;
   - `RELATORIO_ETAPA2_BARREIRA.md`, `RELATORIO_ETAPA3_MULTIPROVEDOR.md` e `RELATORIO_ETAPA4_CHATGPT.md`;
   - `RELATORIO_ETAPA6_CADASTRO.md` e `RELATORIO_ETAPA8_BENCH.md`;
   - `ADR_010_chatgpt_pecas_barreira.md` (decisões de arquitetura do D-11).
6. `CONTINUIDADE_MIGRACAO_GEMINI.md`: histórico detalhado da migração para o Gemini e as pendências dela.

---

## 3. Regras de trabalho combinadas com o usuário

- **Autorização expressa por etapa ou fase.** Nada de código novo fora do que foi autorizado. As etapas 5 a 8 do plano D-11 já estão autorizadas; as fases F5 a F8 do plano original estão fora do escopo e exigem autorização própria.
- **Dúvida → perguntar** antes de decidir.
- **Commits:**
  - só quando o usuário pedir (ele tem aprovado um commit por etapa);
  - **nunca** com coautor ou atribuição ao Claude/IA.
- **Git:** sem push, merge, rebase, reset, clean ou checkout destrutivo sem autorização. Não apagar alterações do usuário. **Não tocar em `logo-cabecalho-docs.jpeg`**: é um arquivo do usuário, não rastreado.
- **Dados:** só casos **sintéticos**. Nunca autos reais da PMRO, CPF, tokens ou chaves no repositório, em logs, no chat ou em argumentos de linha de comando. A tabela de pseudônimos nunca vai à IA.
- **Máquina do usuário:** **não rodar GPU/Vulkan** (já derrubou a máquina). Use só a CPU.
- **Prompts de referência:** os arquivos golden (`internal/policy/testdata/golden`) **não podem mudar**.
- **Atualizar este guia** e o relatório da etapa ao fim de cada etapa.
- **Documentação de bibliotecas:** consultar pelo Context7 ou pela web antes de usar uma API externa.

---

## 4. Preparar a máquina nova

### 4.1 Ferramentas
- **Go:** 1.26 ou superior (`go.mod` pede 1.26.0; o CI e esta máquina usam 1.27.x).
- **Node** 24 com npm.
- **Wails 3 CLI** v3.1.0: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0` (fica em `$(go env GOPATH)/bin`).
- Ambiente de desenvolvimento: **WSL** (Linux). O executável é gerado para Windows por compilação cruzada.

```bash
git clone <repo> && cd sys-executor && git checkout add-mode-apikey-oauth
cd frontend && npm ci && npm run build && cd ..
go build ./... && go test ./internal/...
```

Observações:
- `go build ./...` e `go test ./cmd/sys-executor` **não compilam no Linux**, porque o pacote do app exige GTK4/WebKit. Não é defeito: o app é para Windows.
  - Para compilar no Linux: `go build $(go list ./... | grep -v '/cmd/sys-executor$')`.
  - Para conferir esse pacote: `GOOS=windows go vet ./...`.
- **Versão do Go:** se o Go do PATH for anterior ao 1.27, instale o 1.27.3 com `go install golang.org/dl/go1.27.3@latest && go1.27.3 download` e use `export PATH=$HOME/sdk/go1.27.3/bin:$PATH`. É a versão do CI e do `gofmt` dele.

### 4.2 O que NÃO está no repositório (refazer na máquina nova)
- **Chave de desenvolvimento do Gemini:**
  - guarde-a em `~/.config/sys-executor/gemini-dev.key`, com permissão 600;
  - use sempre `SYSEXEC_GEMINI_API_KEY="$(tr -d '[:space:]' < ~/.config/sys-executor/gemini-dev.key)"`;
  - **nível pago (pré-pago)** desde 10/11/2026. A conta de faturamento usa os créditos do GCP primeiro e exige saldo pré-pago acima de zero;
  - o perfil de pagamentos é de **outra conta Google do usuário**, do tipo pessoa física, que não aceita outros usuários. Para comprar créditos, entre no AI Studio com essa outra conta. Os créditos valem para a conta de faturamento.
- **Conta ChatGPT do bench** (WSL), separada da do app:
  - `go run ./cmd/bench -conectar-chatgpt` abre o navegador do Windows;
  - a credencial de desenvolvimento fica em `~/.config/sys-executor/chatgpt-dev.json` (0600). Nunca no repositório;
  - `-estado-chatgpt` mostra o estado e `-desconectar-chatgpt` revoga a sessão e apaga a credencial.
- **Credenciais do app no Windows:**
  - ficam em `%LOCALAPPDATA%\sys-executor\credenciais\`: `gemini.dpapi` e `chatgpt.dpapi`, cifradas com DPAPI;
  - **não migram entre máquinas.** Em outra máquina, cadastre a chave do Gemini de novo e reconecte o ChatGPT em *Configurações de IA*;
  - o ChatGPT só funciona com conta **com assinatura** (testado com Plus). Conta gratuita recebe `invalid_grant`.
- **Tesseract (OCR):** precisa de `componentes\tesseract\tesseract.exe` e `tessdata\por.traineddata` ao lado do `.exe`. Sem ele, página digitalizada termina em OCR-001. PDFs só com texto funcionam sem OCR.
  - Para teste, a cópia de desenvolvimento fica em `~/.cache/sys-executor-dev/bin/win-tesseract/` e vai para `componentes\tesseract\` com `cp -r`. Não é o binário sem GPL do D-09.7.
- **PDFs sintéticos para o teste manual.** O gerador agora está versionado:
  ```bash
  SAIDA_PDF=/mnt/c/Users/<usuário>/Documents/testes-sys-executor go test -run TestGerarPDFsSinteticos ./internal/testutil/pdfsint
  ```
  Ele cria `pecas-pads-sintetico/` (Memorando, 2 termos de inquirição, alegações finais) e `pecas-sr-sintetico/` (Portaria, termo de inquirição, termo de declarações), além dos PDFs consolidados `*-texto.pdf`.

### 4.3 Gerar e testar o executável
```bash
# bindings (depois de mudar métodos dos serviços em internal/app)
GOOS=windows $(go env GOPATH)/bin/wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor
# interface + executável
cd frontend && npm run build && cd ..
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
```
- **Feche o app antes de copiar** o `.exe` novo; confira com `tasklist.exe | grep -i sys-executor`.
- Pasta de teste usada até aqui: `C:\Users\diogo\Documents\testes-sys-executor\`. Ajuste o caminho na máquina nova.
- **Modo diagnóstico** (mostra a causa técnica, com os dígitos mascarados): no PowerShell, `$env:SYSEXEC_DIAGNOSTICO = "1"; .\sys-executor.exe`.

### 4.4 Verificação completa (antes de cada commit)
```bash
gofmt -l ./internal ./cmd                 # deve ficar vazio
GOOS=windows go vet ./...
go test ./internal/... ./cmd/bench/...    # e -race nos pacotes alterados
cd frontend && npx vitest run && npm run build && cd ..   # vitest 30/30; vue-tsc + vite
go run ./cmd/rulescov -estrito            # catálogo × V4 × matriz × testes
go run ./cmd/bench -provedor falso -entrada memoria,monolitica,pecas   # vazamento 0, sem rede (sai 3 se reprovar)
(cd poc/sei-mapa && go test .)            # se mexer no sei-mapa (usa o Chrome de desenvolvimento, sem GPU)
go run ./cmd/licencas
git diff --stat -- internal/policy/testdata   # golden: deve ficar vazio
```

---

## 5. O que já foi feito

### 5.1 Antes do plano D-11
- Fases F0 a F4 do plano original: base, SEI, ingestão e OCR, motor determinístico, catálogo de regras, etapas de IA com schemas e validadores. Ver `RELATORIO_FASE0..4.md`.
- D-10 (V4 §47): a IA local foi removida, e a análise passou a usar a **Gemini API** com a chave do usuário. Ver `CONTINUIDADE_MIGRACAO_GEMINI.md`.

### 5.2 Plano D-11 (etapas de 09 e 10/11/2026)

| Etapa | Commit | Conteúdo | Situação |
|---|---|---|---|
| 0 | `51af5e5`, `c0b2e57` | Plano, D-11, Emenda 2 (V4 §48), `V4.lock` | Concluída |
| 1 | `e0dfdeb` | PoC do *Sign in with ChatGPT* em Go (`poc/chatgpt`) | Concluída; conta Plus ok |
| 2 | `ab14035` | Barreira de privacidade: camada 1 no pedido e camada 2 nos bytes enviados (IA-014), revisão item a item; correções do 400 do Gemini (`maxItems`) e da classificação da cota | Concluída; testada no Windows |
| 3 | `0c5335c` | Dois provedores sem troca automática; autorização presa a provedor e modelo; preferências v3; IA-015 a IA-020 | Concluída |
| 4 | `e53a815`, `78bc80e` | ChatGPT no app (OAuth com PKCE, DPAPI, Responses API com SSE, `store:false`), IA-021 (criança) | **Testada pelo usuário** (Plus, PADS até TAR-004) |
| 5 | `76df133` | Catálogo de enquadramento com 130 dispositivos (RDPM arts. 15–17 com natureza fixa; Estatuto arts. 29 e 32 com natureza escolhida); formulário PADS-ESC-001 com divergência e confirmação da §22.1 | Concluída; **curadoria a revisar pelo usuário** |
| 6 | `e6f40d6` | Cadastro peça por peça (PADS/SR), junção dos PDFs em memória, peças declaradas conferidas pelo conteúdo, semente de pseudônimos, conferência com a delimitação, assistente `Montagem.vue` | **Testada pelo usuário** com ChatGPT até TAR-004 |
| — | `f001fdd` | Este guia e o gerador versionado de PDFs sintéticos | — |
| 8 | (a commitar) | Bench:<br>• `-provedor gemini\|chatgpt\|falso`, `-entrada`, `-max-chamadas`, `-amostras`;<br>• login do ChatGPT próprio;<br>• métrica de vazamento sobre o corpo exato.<br>ADR 010, README, plano de testes §9, `sei-mapa` ampliado para a etapa 7. | Concluída. Gemini dentro dos limiares; ChatGPT abaixo (§6.2) |

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
  - `frontend/src/views/Montagem.vue`, `components/ArquivosPeca.vue` e `components/EditorEnquadramento.vue`.
- **Enquadramento:** `internal/rules/enquadramento/` (o YAML embutido) e `internal/pipeline/{enquadramento,delimitar}.go`.
- **Bench:** `cmd/bench/`:
  - `provedores.go`: os três provedores e a conta de desenvolvimento;
  - `vazamento.go`: a métrica;
  - `entrada.go`: memória, monolítica e por peças;
  - os casos sintéticos, com `Pessoas`/`Identificadores`, ficam em `internal/testutil/casos`.
- **Governança:**
  - `internal/rules/catalog/*.yaml`;
  - `docs/planning/MATRIZ_REQUISITOS_V4.md`;
  - `docs/planning/PLANO_TESTES_ACEITACAO.md`;
  - `policies/CHANGELOG.md`;
  - `policies/V4.lock`.

---

## 6. O que falta — próximos passos (em ordem sugerida)

### 6.1 Etapa 7 do D-11: SEI pelos documentos selecionados (autorizada, mas **bloqueada pelo usuário**)

**Pré-requisito do usuário.** Rodar `poc/sei-mapa` no SEI real para mapear a tela **"Gerar Arquivo PDF do Processo"** (`procedimento_gerar_pdf`, já liberada na guarda) com "Apenas selecionados".
- O `sei-mapa` foi **ampliado em 10/11/2026**:
  - comando `mapear`;
  - linhas `[post-gerar]`;
  - textos reduzidos à forma (`Aa`, `AA`, `#`), com só os tipos documentais em claro.
- O roteiro está em `poc/sei-mapa/LEIA-ME.md`, seção "Etapa 7".
- O executável fica em `C:\Users\diogo\Documents\testes-sys-executor\sei-mapa.exe`.
- Da F0, já se sabe que a tela tem `selDocumentosApenas` (*select*) e o oculto `hdnDocumentosApenas`. Falta a forma das opções e o separador da lista.

Depois do mapeamento, a implementação prevista (plano §4.8):
1. `jsDocumentos` em `internal/sei/telas.go` lê a tabela (nº SEI, título, tipo) da própria tela de gerar. Isso quebra o invariante "só booleanos": registrar no ADR e no catálogo. Os títulos ficam só no app e não vão à IA.
2. Um `PedidoSelecao` com a lista em `workflow/pedido.go`. O usuário marca os documentos; o app marca as caixas ("Apenas selecionados"), gera, baixa e **fecha o navegador**.
3. O PDF baixado é separado pelo carimbo (`autos.LerCarimbos`) em arquivos virtuais por nº SEI, que entram no **cadastro** (`ArquivoEntrada{Origem: "SEI"}`) com o tipo pré-preenchido pelo título, para o usuário conferir no assistente.
4. Atualizar o simulador `internal/testutil/seisim` (`rdoTipo=A`, tabela, PDF parcial) e os testes.
5. Não ampliar as ações permitidas na guarda (`internal/sei/guarda.go`).

### 6.2 Depois da etapa 8: pendências de decisão (detalhes em `RELATORIO_ETAPA8_BENCH.md` §4)
1. **Mensagem do validador para pessoa sem nome** (`internal/analise/extracao.go:223`). Hoje, um nome vazio é recusado com a mensagem "não tem trecho literal". O ChatGPT corrige o trecho, nunca o nome, e repete o erro. A proposta é separar as duas mensagens. **Exige autorização**: muda o validador, não as políticas.
2. **ChatGPT `gpt-6-luna` abaixo dos limiares** (JSON 1ª 74%, resultado 67%). Repetir a bateria depois do item 1; se preciso, testar `-pensamento high` ou outro modelo da conta.
3. **Variação entre rodadas:** repetir cada caso 3 vezes e usar a mediana antes de afirmar qualidade.
4. **Comandos das rodadas reais** (só casos sintéticos):
   ```bash
   SYSEXEC_GEMINI_API_KEY="$(tr -d '[:space:]' < ~/.config/sys-executor/gemini-dev.key)" \
     go run ./cmd/bench -provedor gemini -modelo gemini-3.6-flash -entrada monolitica,pecas -max-chamadas 120 -saida rel.md
   go run ./cmd/bench -provedor chatgpt -modelo gpt-6-luna -entrada monolitica,pecas -max-chamadas 120 -saida rel.md
   ```

### 6.3 Pendências de revisão e decisão do usuário
1. **Revisar a curadoria do catálogo de enquadramento** em `docs/planning/CATALOGO_ENQUADRAMENTO_REVISAO.md`. Registrar a aprovação no CHANGELOG.
2. **Interrogatório do acusado no PADS** (plano §8.1): hoje vai em "Outros documentos", e o assistente não deixa o acusado ser testemunha ou declarante. Falta decidir se terá tipo próprio.
3. **Imagens (JPG/PNG) como peça:** hoje só PDF; a conversão fica para depois, se o usuário quiser.

### 6.4 Melhorias pequenas já identificadas (dentro do escopo D-11)
- **Conexão por fato:** usar o agrupamento por fato do cadastro para pré-selecionar os grupos na pergunta de conexão que já existe. O motor não muda e o usuário confirma.
- **Conferência da SR:** os ofendidos do cadastro ainda não são cruzados com a leitura da Portaria; os sindicados já são.
- **Teste manual da SR no Windows** com `pecas-sr-sintetico/`. A PADS já foi testada.
- ~~Confirmar o 400 do Gemini~~: **confirmado** no bench de 10/11/2026 (75 chamadas, 0 falhas).

### 6.5 Fora do escopo D-11 (exigem autorização própria)
- **F5:** biblioteca jurídica (busca lexical × embeddings, V4 §47.7).
- **F6/F7:** composição da minuta, DOCX/PDF e revisão. O motor de cálculo existe, mas o pipeline ainda não o chama; a tarefa para em TAR-004.
- **F8:** instalador.

---

## 7. Problemas conhecidos
1. `go build ./...` e `go test ./cmd/sys-executor` não compilam no Linux (GTK). Use o vet do Windows (§4.1).
2. Os pânicos dentro dos workers da extração (fora de `executarSeguro`) ainda derrubam o processo. A melhoria sugerida é um `recover`, convertendo em INT-001.
3. No nível gratuito, o Gemini permite 20 requisições por dia, e uma tarefa completa usa dezenas de chamadas. Desde 10/11/2026 a conta está no nível pago. O bench completo nas duas entradas usa cerca de 75 chamadas no Gemini e 104 no ChatGPT (limite de 5 h do Plus, compartilhado).
4. Dez commits antigos da branch, anteriores a esta regra, têm a linha `Co-Authored-By`. O usuário foi informado e nada foi reescrito.
5. O `gofmt` do CI já acusou `internal/ingest/segment/carimbo_test.go`, arquivo anterior à migração. Com o Go 1.27.3 local ele está limpo; provavelmente é diferença de versão do formatador. Confira no próximo CI.
6. O backend ainda tem `ReceberUpload` (PDF único) para os testes legados e o SEI atual. A interface não oferece mais esse caminho (decisão E6).
