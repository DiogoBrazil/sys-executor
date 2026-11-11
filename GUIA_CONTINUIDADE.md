# Guia de continuidade — sys-executor

> **Comece por aqui.** Ponto de entrada para retomar o trabalho em qualquer máquina.
>
> - **Atualizado em:** 11/11/2026 (fim do dia). Estado:
>   - os testes A (SEI real) e B (SR) passaram, com dois ajustes (commit `8596de3`);
>   - a **F5 essencial (biblioteca jurídica)** está implementada e commitada (§5.5);
>   - a **F6 e a F7 do PADS** (cálculo, redação, validação, ajustes, DOCX/PDF, finalizar) estão implementadas e commitadas (§5.7); o usuário **gerou um Julgamento de PADS até o fim** no `.exe`;
>   - o **novo visual da interface** foi refeito por **protótipo aprovado pelo usuário** (canvas de design no claude.ai) e implementado no Vue: cadastro em cartões e modais, trilha horizontal de etapas, imputações em tabela, histórico linha a linha, dosimetria em painéis, robô/engrenagens na espera, minuta com prévia rolável. **Testado e aprovado pelo usuário no `.exe` em 11/11/2026** e commitado (§5.3).
> - **Prioridade do usuário:** Julgamento de PADS completo (feito). O próximo grande bloco é a **SR nas F6 e F7**, que exige autorização (§6.0).
> - **Decisão do usuário:** **sem mais bench**. Os testes são manuais, no `.exe` (§3, §6.2).
> - **Branch:** `add-mode-apikey-oauth`. O push é feito pelo usuário. Não há merge com a `main`.
> - **Em uma linha:** PADS completo de ponta a ponta e com o visual aprovado; faltam as respostas pendentes do usuário sobre a minuta, as decisões abertas do PADS, a mesma cadeia para a SR e o fechamento (§6.0).
> - **Próximos passos:** **§6.0**. **O que não pode regredir:** §6.0, "O que NÃO refazer".

---

## 1. Onde estamos

O aplicativo (Go + Wails 3 + Vue, para Windows) percorre:
1. a escolha do procedimento (PADS ou SR);
2. a obtenção dos autos **peça por peça**, por upload ou **pelo SEI**, com a seleção dos documentos no computador;
3. a pseudonimização e a **barreira de privacidade**;
4. a delimitação com **enquadramento manual**;
5. a análise das provas com o **Gemini** (chave do usuário) ou o **ChatGPT** (*Sign in with ChatGPT*, plano do usuário).

Desde a F5 essencial, a etapa **"Consultando fontes jurídicas"** funciona: recupera da biblioteca local os dispositivos das imputações e a base do julgamento. Desde a parte 1 da F6, o PADS também **calcula a sanção e o comportamento** e mostra a memória de cálculo para conferência. Desde a parte 2 da F6, o PADS também **redige a minuta**. Desde as partes 3 e 4 e a F7, a minuta é **validada** e vai para **"Aguardando sua revisão"**, com **Salvar DOCX**, **Salvar PDF**, **Solicitar ajustes** e **Aprovar e finalizar** (§5.7). A SR ainda para em "Elaborando a minuta" (TAR-004). A interface foi redesenhada por protótipo aprovado (§5.3).

### 1.1 O que já foi testado e o que não foi
| Item | Automatizado | Manual no Windows (usuário) |
|---|---|---|
| ChatGPT no app (login, modelo, PADS até TAR-004) | ✅ | ✅ 09/11/2026, conta Plus, `gpt-6-luna` |
| Cadastro de peças PADS | ✅ | ✅ 09/11/2026, com ChatGPT |
| Cadastro de peças SR | ✅ | ✅ 10/11/2026, ChatGPT, até TAR-004 (ajustes P1 a P3 na Fase 1 da interface) |
| SEI com seleção no computador (etapa 7) | ✅ CAD-SEI-01 (simulado, pdfium real) | ✅ 10/11/2026, processo real, ChatGPT, até TAR-004 (depois dos ajustes do `8596de3`) |
| Novo visual, Fase 1 (base, P1 a P3, itens 1 e 2) | ✅ vitest, galeria | ⚠️ PADS visto pelo usuário nos testes de 11/11; SR não confirmada |
| Telas mais horizontais (Fases 2 e 3 adaptadas aos ajustes de 11/11, §5.3) | ✅ vitest | ✅ robô e minuta aprovados pelo usuário (11/11) |
| Protótipo aprovado (cadastro em cartões e modais, trilha horizontal, imputações em tabela, histórico linha a linha, dosimetria em painéis, parada e fim) | ✅ vitest 43/43, galeria revisada | ✅ 11/11/2026: testado e aprovado pelo usuário no `.exe` (PADS). SR com o visual novo ainda não testada (§6.0) |
| Biblioteca jurídica, F5 essencial (§5.5) | ✅ leitor, acervo, busca, D-01 e etapa no fluxo do PADS | ✅ 11/11/2026, PADS sintético, ChatGPT |
| PADS completo: cálculo, minuta, validação, revisão, DOCX e PDF (§5.7) | ✅ fluxo inteiro com IA simulada; validadores; DOCX e PDF gerados e lidos de volta | ✅ 11/11/2026: minuta gerada até o fim, DOCX e PDF salvos. O usuário **não relatou** o resultado de "Solicitar ajustes", de "Alterar" na revisão nem de "Aprovar e finalizar": confirme com ele. Ajustes de tela pendentes (§6.0) |
| Gemini no app, de ponta a ponta | ✅ | ⚠️ **parcial**: a cota gratuita acabou no teste de 09/11/2026. A conta está no nível pago desde 10/11 (§6.1-C) |
| Bench Gemini `gemini-3.6-flash` (casos sintéticos) | ✅ dentro dos limiares; vazamento 0 | — |
| Bench ChatGPT `gpt-6-luna` | ⚠️ abaixo dos limiares de JSON 1ª e de resultado; vazamento 0 | Bench encerrado por decisão do usuário (§6.2); a qualidade é avaliada nos testes manuais |
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
8. **Interface:** `docs/planning/PLANO_INTERFACE.md` (plano aprovado, 3 fases) e `docs/planning/RELATORIO_INTERFACE.md` (o que cada fase fez).
9. **Biblioteca jurídica:** `docs/planning/RELATORIO_FASE5.md` (o que entrou no recorte essencial e o que ficou adiado).
10. **Cálculo e minuta do PADS:** `docs/planning/RELATORIO_FASE6.md` (partes entregues e o que falta).

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
- **Testes (decisão de 10/11/2026):** **não rodar mais o bench** com IA real. O usuário testa manualmente. Ao terminar uma implementação ou ajuste:
  - rodar só a verificação local, sem rede (§4.4, menos o bench real);
  - gerar o `sys-executor.exe` na pasta de teste (§4.3);
  - passar ao usuário um roteiro curto do que abrir e conferir. Economizar tokens: nada de baterias de testes em sequência.
- **Bibliotecas:** consultar a documentação pelo Context7 ou pela web antes de usar uma API externa.

---

## 4. Preparar a máquina nova

### 4.1 Ferramentas (WSL/Linux; o executável é gerado para Windows por compilação cruzada)
- **Go 1.27.3**, a versão do CI e do `gofmt` dele. O `go.mod` pede 1.26.0. O Go do sistema pode ser mais antigo (em 11/11/2026 era o 1.23.1 em `/usr/local/go`): use sempre o do `~/sdk`.
  ```bash
  go install golang.org/dl/go1.27.3@latest && go1.27.3 download
  export PATH=$HOME/sdk/go1.27.3/bin:$PATH      # em todo terminal de trabalho
  ```
- **Node 24** com npm.
- **Wails 3 CLI v3.1.0:** `CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0` (fica em `$(go env GOPATH)/bin`). Sem o `CGO_ENABLED=0`, a instalação falha no WSL por falta de GTK4/WebKit. Só é usado para gerar os bindings (§4.3).

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
| Tesseract (OCR) do teste manual | `componentes\tesseract\` ao lado do `.exe` (`tesseract.exe`, DLLs, `tessdata\por.traineddata` e `osd.traineddata`) | No PowerShell: `winget install --id UB-Mannheim.TesseractOCR --version 5.4.0.20240606 -e --interactive` (marque **Portuguese** em "Additional language data"; a instalação silenciosa não traz o português). Depois copie `C:\Program Files\Tesseract-OCR\.` para `componentes\tesseract\`.<br>Sem ele, página escaneada termina em OCR-001. PDF com texto funciona sem OCR.<br>Não é o binário sem GPL previsto no D-09.7 para o instalador. |
| Typst (PDF da minuta) | `componentes\typst\typst.exe` ao lado do `.exe` (opcional: `componentes\typst\fonts\` com a Liberation Serif) | Baixar o **Typst 0.15.1** (página oficial de versões): `curl -sSLfO https://github.com/typst/typst/releases/download/v0.15.1/typst-x86_64-pc-windows-msvc.zip` e extrair o `typst.exe` para essa pasta; para os testes no WSL, `typst-x86_64-unknown-linux-musl.tar.xz` em `~/.cache/sys-executor-dev/bin/typst/typst`. Sem ele, "Salvar PDF" mostra o erro EXP-002; o DOCX não depende dele. No WSL, os testes usam `SYSEXEC_TYPST` (há um Typst de Linux em `~/.cache/sys-executor-dev/bin/typst/`); sem a variável, o teste do PDF é pulado. |
| Logotipo do cabeçalho dos documentos | `componentes\cabecalho\` ao lado do `.exe` (`logo-pmro.png` ou `logo-cabecalho-docs.jpeg`) | Copiar o arquivo do usuário (`logo-cabecalho-docs.jpeg`, na raiz do repositório, **não rastreado**). Sem a imagem, o cabeçalho sai só com o texto. No WSL: `SYSEXEC_LOGO`. |
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
- **Galeria de telas (só desenvolvimento):** `cd frontend && npm run galeria` abre cada tela com dados sintéticos (`galeria.html#resultado`, `#cadastro-sr`…). As capturas são feitas com o `chrome-headless-shell` do Playwright com `--disable-gpu --disable-features=Vulkan`. Ela não entra no `.exe`.
- **Modo diagnóstico:** mostra a causa técnica, com os dígitos mascarados. No PowerShell: `$env:SYSEXEC_DIAGNOSTICO = "1"; .\sys-executor.exe`.

### 4.4 Verificação completa (antes de cada commit)
```bash
gofmt -l ./internal ./cmd                                   # vazio
GOOS=windows go vet ./...
go vet ./internal/... ./cmd/rulescov/... ./cmd/licencas/... ./cmd/bench/...
go test ./internal/... ./cmd/bench/...                      # -race nos pacotes alterados; com SYSEXEC_TYPST=<typst> o PDF é gerado de verdade (senão o teste é pulado)
cd frontend && npx vitest run && npm run build && cd ..     # vitest 37/37; vue-tsc + vite
go run ./cmd/rulescov -estrito                              # 202 regras; 124/124 IDs da V4
go run ./cmd/bench -provedor falso -entrada memoria,monolitica,pecas   # vazamento 0, sem rede
(cd poc/sei-mapa && go test .)                              # só se mexer no sei-mapa
go run ./cmd/licencas                                       # 45 módulos Go, 40 npm
go run ./cmd/bibgen -conferir                               # acervo da biblioteca em dia com docs/law_library
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
| 7 | `0579327` | SEI com seleção no computador (opção A, E2 revisado):<br>• o PDF de todos os documentos é separado pelo carimbo;<br>• os documentos vão ao cadastro, e o usuário escolhe os de cada peça.<br>V4 §48.4 revisada; TRV-ING-008; CAD-SEI-01.<br>Na extração, pessoa sem nome é descartada, sem recusa. | ✅ Testada pelo usuário (SEI real, 10/11) |
| — | `3bb0c6f` | Guia reescrito para a continuidade | — |
| — | `8596de3` | **Ajustes do teste real** (ver `RELATORIO_ETAPA7_SEI.md` §6 e §7):<br>• revisão de nomes: pedaços de nome conhecido ocultados, parentes da qualificação como âncora, filtros de termos, lugares e ruído de OCR. Caiu de 18 para 1 item no processo real;<br>• delimitação: pareamento pelo artigo, sem aviso de natureza, acusado com posto, "fato 1";<br>• tela de depoimentos. | ✅ Testado pelo usuário (PADS pelo SEI e SR) |

### 5.3 Novo visual da interface (pedido do usuário, 10/11/2026)
- **Plano aprovado:** `docs/planning/PLANO_INTERFACE.md`, em 3 fases. **Relatório:** `docs/planning/RELATORIO_INTERFACE.md`.
- **Decisões do usuário:**
  - identidade "capa dos autos refinada" (aba bordô no PADS e azul-petróleo na SR), **sem tema escuro**;
  - no processamento, **etapas à esquerda + abas** (Análise, Suas respostas, Autos) e barra "Confirmar" fixa;
  - ajustes funcionais P1 a P3 e itens 1 e 2 na Fase 1;
  - **galeria local** de telas para conferir o visual, sem Superdesign.
- **Restrições:** sem dependência npm nova (continua com 40), `data-teste` preservados, sem mudança em regra jurídica, golden ou V4.

| Fase | Conteúdo | Situação |
|---|---|---|
| 1 | Tokens e estilos globais (`frontend/src/estilo.css`), `components/ui/{Icone,Recolhivel,Abas}.vue`, `AvisoErro` como callout, `--acento` pelo tipo no `App.vue`; galeria (`frontend/galeria.html`, `src/galeria/`, `vite.galeria.config.ts`, `npm run galeria`).<br>Ajustes:<br>• **P1:** conclusão da SR com o sindicado no título (`analisar.go`);<br>• **P2:** aviso de ofendido institucional (`Montagem.vue`);<br>• **P3:** "Usar a mesma" para depoente já cadastrado, e avisos na conferência (sindicado com outro papel, pessoa repetida);<br>• **item 1:** acusado com posto também sem leitura da IA (`cadastro.go`);<br>• **item 2:** grupo vazio escondido (`CartaoPergunta.vue`). | ✅ Implementada e commitada (commit "feat(interface): fase 1…", 10/11/2026). `.exe` gerado. **Teste do usuário pendente** |
| 2 | Processamento:<br>• lombada de etapas à esquerda (sticky);<br>• pergunta no centro com aba da etapa, "Trechos citados" recolhidos e agrupados por peça, fundamentos em lista com "Mostrar mais", Lacuna/Diligência com ícone;<br>• barra Confirmar fixa e rolagem até o 1º erro;<br>• abas Análise / Suas respostas / Autos;<br>• `PainelAnalise` com `Recolhivel` e selos; `PainelAutos` com ícones;<br>• modal do PDF animado. | ✅ Substituída pelo redesenho aprovado (abaixo), 11/11/2026; testada e aprovada pelo usuário; commitada |
| 3 | Cadastro (stepper com situação de cada passo, cartões, "Como funciona?" recolhido, barra Anterior/Próximo fixa, conferência em tabela), `ArquivosPeca`, `EditorEnquadramento`, `CampoBusca`, Início, Entrada, `PainelSEI`, Configurações de IA e `FaixaTipo` | ✅ Cadastro, `ArquivosPeca`, `EditorEnquadramento` e Entrada no redesenho aprovado (abaixo), 11/11/2026. ❌ Início, `PainelSEI`, Configurações de IA e `FaixaTipo` não foram pedidos (pergunte, §6.0) |

**Redesenho de 11/11/2026 (pedido do usuário; protótipo aprovado e implementado; testado e aprovado no `.exe`).** Substitui as Fases 2 e 3 do plano onde elas conflitam. Detalhes: `RELATORIO_INTERFACE.md`, seções "Fases 2 e 3…" e "Protótipo aprovado".
- **Como foi feito:** primeiro uma rodada direto no código (aprovados dela: **robô/engrenagens da espera** e **minuta com prévia rolável e tela cheia**). Depois **protótipo de 10 telas** num canvas de design privado do usuário no claude.ai ("Telas do julgamento – protótipo"), com dados sintéticos, revisado em três rodadas e aprovado; então implementado no Vue.
- **Entrada** (`Entrada.vue`): trilha "Tipo › Origem › Processamento", dois cartões com botão, nota de cifra; "Escolher outro caminho" e "Trocar o tipo de minuta" no rodapé.
- **Cadastro** (`Montagem.vue`): título "Cadastre as peças do **processo**" (SR: "do **procedimento**"); barra de progresso; **cartões das peças** (✓ verde, "Pendente", "Opcional") e cartão "Conferir e processar"; **cada peça num modal** (`Janela`) com Anterior/Continuar; depoimentos e outros documentos em **linhas**; conferência em tabelas com o painel "Concluir e processar"; **loader ao anexar** (`ArquivosPeca`, modos completo e compacto).
- **Processamento** (`Processamento.vue`): **trilha horizontal** de etapas (✓ desenhado, arco girando quando o app trabalha, "sua vez", "parou") com Análise, Suas respostas (gavetas), Autos lidos (modal), Consultar os autos e Cancelar tarefa na mesma faixa; **espera** com `ui/Figura.vue` (robô nos estados de IA `DELIMITANDO`, `ANALISANDO`, `REDIGINDO`, `AJUSTANDO`, `VALIDANDO`; engrenagens nos demais); **cartão de parada** com "O que fazer" e código; **tela final** centralizada.
- **Perguntas** (`CartaoPergunta.vue` + `CampoPergunta.vue`): campos com borda mais forte e 44 px; escolhas em **controle segmentado** (`.segmentado`) ou cartões (`.escolhas`); explicação longa com "Ler a explicação completa" (menos `envio_ia`). Formatos especiais, escolhidos pelos dados do Go:
  - `confirmar_delimitacao` → **tabela das imputações** (colunas pelos prefixos `conduta_`, `diploma_`/`disp_rdpm_`/`disp_est_`/`natureza_`, `data_`, `incluir_`; o resto e os alertas numa linha abaixo; acusado ao lado do título);
  - grupos com título "Prefixo n" (3 ou mais, sem conteúdo próprio, ex.: `historico` h1…h5) → **linhas acrescentadas uma a uma**;
  - campo solto de que dependem campos de grupo (ex.: `tem`) → vem **antes** dos grupos;
  - grupos `s_*` → **painéis da dosimetria** (sanção, dias com −/+, comportamento proposto a partir do item "propostos" da memória) e grupos `m1…m7` → **memória de cálculo em modal**.
- **Minuta** (`MinutaPrevia.vue`): prévia rolável e "Ler em tela cheia" (Esc); aviso, pendências, arquivos, ajustes e finalizar na coluna ao lado.
- **PDF** (`VisualizadorFontes.vue`): loader (`ui/Carregando.vue`) ao abrir os autos e a cada página.
- **Componentes base novos:** `ui/Janela.vue` (modal/gaveta com o conteúdo **no DOM**, `v-show`), `ui/Figura.vue`, `ui/Carregando.vue`, `components/CampoPergunta.vue`.
- **Não redesenhadas** (fora do pedido): Início, `PainelSEI`, Configurações de IA e `FaixaTipo`.
- Sem pacote npm novo (40); sem mudança em Go, golden ou V4; `data-teste` preservados; vitest 43/43.

**Capturas da galeria** (eu as reviso antes de entregar o `.exe`):
1. `cd frontend && npx vite build -c vite.galeria.config.ts --outDir <scratch>/galeria`;
2. `npx vite preview -c vite.galeria.config.ts --outDir <scratch>/galeria` (porta 9311);
3. `$(ls -d ~/.cache/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-linux64/chrome-headless-shell | tail -1) --headless --disable-gpu --disable-features=Vulkan --no-sandbox --hide-scrollbars --virtual-time-budget=3000 --window-size=1440,<altura> --screenshot=<arq>.png "http://127.0.0.1:9311/galeria.html#<cenario>"`.

Os cenários estão em `src/galeria/cenarios.ts`, com a altura de cada um. O número da pasta do Playwright muda com as atualizações (era 1248, passou a 1254): use o `ls -d … | tail -1`. Cenários com `clicar` abrem o modal ou a gaveta e desligam as animações, porque a captura com tempo virtual congela as transições de CSS no meio (o modal aparece "esmaecido" na imagem, mas está aberto). Em outra máquina, instale o navegador com `npx playwright install chromium-headless-shell`, se ele faltar. **Nunca use GPU** (§3).

### 5.4 Onde está cada coisa (código)
- **Interface (frontend):**
  - tokens e estilos globais: `frontend/src/estilo.css`;
  - componentes base: `src/components/ui/` (Icone, Recolhivel, Abas, **Janela**, **Figura**, **Carregando**);
  - campo de pergunta (todos os tipos): `src/components/CampoPergunta.vue`; formatos da pergunta: `CartaoPergunta.vue` (§5.3);
  - telas: `src/views/`; perguntas genéricas: `src/components/CartaoPergunta.vue`; cadastro: `src/views/Montagem.vue`;
  - galeria: `frontend/galeria.html`, `src/galeria/`, `vite.galeria.config.ts`;
  - testes de tela: `src/telas.test.ts` (dependem dos `data-teste`).
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
- **Biblioteca jurídica (F5):**
  - `internal/legal/` (`norma.go`, `acervo.go`, `biblioteca.go`, `referencia.go`, `vigencia.go`);
  - dados embutidos em `internal/legal/dados/` (`manifesto.yaml`, `acervo.json.gz`, `notas_vigencia.yaml`, `tesauro.yaml`);
  - gerador do acervo: `cmd/bibgen`;
  - etapa do pipeline: `internal/pipeline/fontes.go`.
- **Cálculo oficial do PADS (F6, parte 1):** `internal/pipeline/calculo.go`, sobre o motor de `internal/rules/engine`.
- **Minuta do PADS (F6, parte 2):** trechos da IA em `internal/analise/redacao.go`; modelo do documento e textos fixos em `internal/minuta/`; etapa em `internal/pipeline/redigir.go`; tela em `frontend/src/components/MinutaPrevia.vue`.
- **Validação e ajustes da minuta (F6, partes 3 e 4):** verificação de suporte e classificação do pedido em `internal/analise/validacao.go`; validadores determinísticos em `internal/minuta/validar.go`; etapas em `internal/pipeline/validar.go`; comandos `SolicitarAjuste` e `Aprovar` em `internal/workflow/gerenciador.go`.
- **Arquivos da minuta (F7):** `internal/documento/` (`docx.go`, `typst.go`, `documento.go`); ligação em `internal/pipeline/documento.go` e `internal/app/tarefa.go` (`SalvarMinuta`).
- **Governança:**
  - `internal/rules/catalog/*.yaml`;
  - `docs/planning/MATRIZ_REQUISITOS_V4.md`;
  - `docs/planning/PLANO_TESTES_ACEITACAO.md`;
  - `policies/CHANGELOG.md`;
  - `policies/V4.lock`.

### 5.5 Fase 5 — biblioteca jurídica, recorte essencial (11/11/2026)
- **Relatório:** `docs/planning/RELATORIO_FASE5.md`.
- **Decisões do usuário:** busca **somente lexical** (V4 §47.7); recorte essencial agora; PADS primeiro.
- **Feito:**
  - leitor de normas (artigo, §, inciso, alínea; redação histórica × atual; partes de um documento composto);
  - acervo inicial de `docs/law_library` **embutido no `.exe`** (14 documentos; 2 escaneados lidos por OCR);
  - busca por referência exata e por palavras (FTS5), com tesauro;
  - nota de vigência do art. 41, § 3º do RDPM (D-01);
  - etapa "Consultando fontes jurídicas" no PADS, com a seção "Fontes jurídicas" na aba Análise.
- **Adiado** (para depois da F7): tela de gestão do acervo, tela de duplicidade RDPM × Coletânea e pesquisa web.
- **Ao mudar um PDF de `docs/law_library` ou o manifesto:** rode `go run ./cmd/bibgen`. Os dois originais escaneados precisam do Tesseract só se mudarem:
  ```bash
  T=/mnt/c/Users/<usuário>/Documents/testes-sys-executor/componentes/tesseract
  go run ./cmd/bibgen -tesseract $T/tesseract.exe -tessdata "$(wslpath -w $T/tessdata)"
  ```
- **Situação:** implementada, commitada e **testada pelo usuário** em 11/11/2026 (PADS sintético, ChatGPT, até "Calculando"). Do teste saíram dois ajustes: o hífen de fim de linha do PDF e o texto do aviso de parada.

### 5.6 Modelos de minuta (`docs/draft_templates/`)
- O usuário criou `docs/draft_templates/PADS` e `docs/draft_templates/SR` com julgamentos e relatórios já feitos, para servirem de **referência de forma e estilo** na F6 e na F7 (D-04).
- **Não entram em commit** (decisão do usuário). A pasta está no `.gitignore`. **Ela não existe na outra máquina:** o usuário vai recriá-la lá e pôr os modelos. Se a pasta não existir, nada quebra (nenhum código a lê).
- **Atenção:** os dois modelos de PADS trazem nomes completos, números de processo e códigos do SEI (só a matrícula está mascarada). Trate-os como dados reais: nada deles vai para o código, para fixtures, para o chat ou para a IA do aplicativo. Os modelos de SR ainda não foram lidos.

---

### 5.7 Fases 6 e 7 do PADS — cálculo, minuta, validação, revisão e arquivos (11/11/2026)
- **Relatórios:** `docs/planning/RELATORIO_FASE6.md` e `docs/planning/RELATORIO_FASE7.md`.
- **Cadeia completa do PADS:** autos → análise → fontes → cálculo oficial (conferido pelo usuário) → redação por trechos → validação → revisão → DOCX/PDF → finalizar.
- **Validação (V4 §42):**
  - a IA confere, trecho a trecho, se as provas citadas sustentam cada parágrafo e se a fundamentação não contraria o resultado confirmado;
  - o que não se sustenta é reescrito, até 2 vezes por trecho; se persistir, vira item de "O que conferir ou completar";
  - o aplicativo confere a minuta inteira contra as respostas e o cálculo (seções, resultado de cada imputação, punição, dias, pontos, cláusula de residência, nota de punição, proibições de forma, evidências);
  - falha dessa conferência não bloqueia a revisão: aparece em destaque (VAL-001).
- **Revisão (V4 §31.3):** pedido de ajuste em texto livre. A IA classifica o pedido e reescreve **só os trechos atingidos**. Pedido que mexe em decisão confirmada ou em texto fixo não é aplicado, e o usuário é orientado. O pedido fica só na memória da tarefa.
- **Arquivos (V4 §32):** DOCX por escritor próprio e PDF pelo Typst, do mesmo documento. Times New Roman 12, A4, margens 2/2/3/1,5 cm, recuo de 2,5 cm, cabeçalho com logotipo e as duas linhas, rodapé com a página. `[A PREENCHER]` sai realçado. Sem autor, sem data de geração e sem assinatura.
- **Finalizar (V4 §29.1):** "Aprovar e finalizar" pede confirmação ("Pode finalizar"), avisa se nada foi salvo e apaga a sessão.
- **Não feito:**
  - prévia do PDF dentro do aplicativo (a prévia é a folha em HTML; o PDF é conferido no arquivo salvo);
  - Liberation Serif empacotada (reserva da Times New Roman);
  - tudo isso para a SR.
- **Situação:** implementado, verificado localmente, **commitado** e usado pelo usuário até o DOCX e o PDF (11/11/2026). Ajustes de tela pendentes (§6.0, passo 1).

## 6. O que falta

### 6.0 Próximos passos (em ordem)
**Antes de tudo, na máquina nova:** prepare-a pelo §4 (inclusive o `typst.exe` e o logotipo em `componentes\`, §4.2), rode a verificação do §4.4 e gere o `.exe`. Tudo deve estar verde **antes** de qualquer alteração; se não estiver, o problema é do ambiente, não do código.

1. ✅ **Ajustes de tela do PADS — concluídos** (11/11/2026): protótipo aprovado, implementado, testado e aprovado pelo usuário no `.exe`, commitado (§5.3). **Não refaça.** Ajuste visual novo: pergunte ao usuário e, se for mudança grande, faça antes um protótipo no canvas de design (o usuário aprovou esse fluxo; o canvas dele é "Telas do julgamento – protótipo").
   **Pergunte ao usuário (ainda sem resposta):**
   - o que achou do texto da IA (relatório, fundamentação, art. 36) e do que apareceu em "O que conferir ou completar";
   - do DOCX no Word (cabeçalho, logotipo, margens, fonte) e do PDF;
   - se usou "Solicitar ajustes", "Alterar" durante a revisão e "Aprovar e finalizar" e se funcionaram;
   - se quer o novo visual também em Início, Configurações de IA, `PainelSEI` e `FaixaTipo` (hoje no visual da Fase 1).
   **Teste ainda não feito no `.exe` com o visual novo:** a **SR** (cadastro com Portaria e envolvidos no modal; processamento até TAR-004) e o caminho **pelo SEI** (documentos do SEI dentro dos modais). Peça ao usuário um teste rápido da SR com `pecas-sr-sintetico\`.
2. **Decisões do PADS que estão abertas:**
   - gênero do acusado nos textos fixos (hoje "o(a) acusado(a)"; perguntar no cadastro?);
   - logotipo: embutir no `.exe` (o arquivo entra no repositório) ou manter em `componentes\cabecalho\`; e qual imagem (a V4 §32.1 cita o brasão da PMRO, `logo-pmro.png`; o arquivo fornecido é o do Governo do Estado);
   - prévia do PDF dentro do aplicativo (hoje a prévia é a folha em HTML);
   - Lei nº 2.484/2011 na cláusula de residência: está citada, pela redação da V4 §10, e aguarda a confirmação dele;
   - autoridade, nº do PADS, local e assinatura saem `[A PREENCHER]`, sem pergunta (decisão dele; reabrir só se ele pedir).
3. **SR nas F6 e F7** (exige autorização; é o maior bloco que falta). A SR hoje para em "Elaborando a minuta" (TAR-004). Para chegar ao Relatório de Sindicância Regular:
   - ler os modelos de `docs/draft_templates/SR` (o usuário vai recriar a pasta; **confirme com ele se pode ler** e trate como dados reais, §5.6);
   - `internal/minuta/sr.go`: `ComporSR`, com a estrutura da V4 (§§21 a 25) e os textos fixos;
   - trechos da IA para a SR em `internal/analise/redacao.go` (por fato e por sindicado), com as mesmas conferências;
   - `internal/pipeline/redigir.go`: hoje `redigir`, `validar` e `ajustar` devolvem TAR-004 quando o tipo não é PADS, e o `redator` é só do PADS. Generalize sem mudar o caminho do PADS;
   - `internal/minuta/validar.go`: `ValidarSR`;
   - `internal/documento`: já aceita a SR (título e nome de arquivo); falta o estilo das alíneas, se houver;
   - a SR não tem cálculo (V4 §21): vai de "Consultando fontes" direto a "Elaborando a minuta".
4. **Visual das telas restantes** (Início, Configurações de IA, `PainelSEI`, `FaixaTipo`): só se o usuário pedir (passo 1).
5. **Restante da F5** (§5.5): tela de gestão do acervo, duplicidade RDPM × Coletânea, pesquisa web.
6. **Teste C, Gemini de ponta a ponta** (§6.1-C), agora até o DOCX/PDF.
7. **Fechar o D-11:** decisões do §6.3 e registro da conclusão (§6.5).
8. **F8, entrega:** instalador (WiX/NSIS, certificado), componentes empacotados (Tesseract, Typst, Liberation Serif, logotipo), manual do usuário, testes de segurança e de desempenho, relatório de riscos e aceite (plano §16).
9. **Merge com a `main`:** decisão e ação do usuário.

#### O que NÃO refazer e o que não pode regredir
- **Não reescreva o que está pronto:** biblioteca (`internal/legal`), cálculo (`internal/pipeline/calculo.go`), redação, validação, ajustes e arquivos do PADS (§5.4 e §5.7). Mexa só no que o usuário pedir.
- **Motor determinístico** (`internal/rules/engine`) e **golden** (`internal/policy/testdata/golden`): não mudam. Os prompts de `policies/` também não mudaram nas F5 a F7: a tarefa de cada chamada vai no pedido (`Chamada.Tarefa`), não no arquivo de política.
- **A IA nunca escreve** a decisão, a punição, os dias, os pontos, a cláusula de residência, as providências nem o fecho: isso é de `internal/minuta` (plano §5.5).
- **Toda chamada de IA** passa por `p.comRevisao(...)` e por `prepararIA` (autorização de envio e barreira de privacidade). Etapa nova que chame a IA segue o mesmo molde de `redigir`, `validar` e `ajustar`.
- **Cuidado com maiúsculas nas instruções à IA:** duas palavras seguidas com inicial maiúscula ou em caixa alta (ex.: "Fora de REDACAO") são tomadas pela barreira por possível nome de pessoa e param a tarefa com a pergunta de revisão. O teste do fluxo do PADS apanha isso.
- **A minuta só aparece depois da validação** (V4 §42.1) e some quando uma resposta é alterada; não publique a minuta em outra etapa.
- **Nada da tarefa é gravado fora da sessão**, salvo o arquivo que o usuário manda salvar. Sem logs, sem histórico de pedidos de ajuste.
- **Anotações `// Regra:`** nos testes precisam de IDs que existem em `internal/rules/catalog` (o `rulescov -estrito` acusa).
- **Teste parado?** `SYSEXEC_TESTE_PRAZO=30s go test ./internal/pipeline/` encurta a espera e mostra em que pergunta a tarefa ficou.
- **Interface aprovada (§5.3):** não redesenhe telas aprovadas. Mantenha:
  - todos os `data-teste` (os testes de tela dependem deles) e o conteúdo dos modais **no DOM** (`Janela` usa `v-show`; os testes encontram campos dentro de modais fechados);
  - `CampoPergunta.vue` como o único lugar que desenha um campo; os formatos do `CartaoPergunta.vue` dependem dos ids e títulos que o Go manda (`confirmar_delimitacao`, prefixos `incluir_`/`conduta_`/`diploma_`/`disp_*`/`natureza_`/`data_`, grupos `s_*` e `m1…m7`, títulos "Transgressão anterior n"). **Se mudar esses nomes no Go, ajuste o front junto**, senão a tela cai no formato comum;
  - "processo" no PADS e "procedimento" na SR;
  - sem pacote npm novo; ícones e figuras em SVG próprio (IconScout descartado: licença e dependência);
  - o bloco de testes "Telas do protótipo aprovado" e os cenários da galeria.
- Os testes de fluxo que provam o PADS inteiro: `TestFluxoPADSComIA` e `TestRevisaoNoMeioDaEtapaENenhumMarcadorNoEstado` (`internal/pipeline/ia_test.go`), `internal/minuta/minuta_test.go`, `internal/documento/documento_test.go` e o bloco "F6 — minuta" de `frontend/src/telas.test.ts`. Se algum quebrar, é regressão.

### 6.1 Testes manuais do usuário no Windows (bloqueiam a conclusão)
Antes, gere o `.exe` (§4.3) e os PDFs sintéticos e copie o Tesseract (§4.2). Em *Configurações de IA*, conecte o ChatGPT e cadastre a chave do Gemini.

**A. SEI real (etapa 7).** ✅ **Concluído** em 10/11/2026: login, download, separação (17 documentos) e sugestões corretas. A revisão de nomes trouxe muitos falsos positivos → ajuste no `internal/pseudonimo` (relatório da etapa 7, §6 e §7). ✅ Reteste: 1 item (pessoa real) na revisão, delimitação sem repetições, processamento até TAR-004 (fim previsto do D-11).

Use um processo de teste ou já encerrado.
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

**B. SR com upload** (`pecas-sr-sintetico/`). ✅ **Concluído** em 10/11/2026: Portaria, sindicados, ofendido, depoimentos e conferência, até TAR-004, sem nenhum item na revisão de nomes.
- Achados, todos corrigidos na Fase 1 da interface:
  - **P1:** duas conclusões com o mesmo título;
  - **P2:** "Administração Pública" cadastrada como ofendido pessoa;
  - **P3:** sindicado cadastrado de novo como depoente declarante.
- Não testados: a pergunta de conexão (só havia um fato) e o alerta de sindicados divergentes (os nomes batiam).

**C. Gemini no app, de ponta a ponta** (`pecas-pads-sintetico/`), ❌ **pendente**: escolha a Gemini em *Configurações de IA*. Faça a confirmação de envio, a delimitação e a análise até TAR-004. Ele já roda no nível pago, sem a cota de 20 por dia.

**D. Opcional:** repetir um caso com o ChatGPT `gpt-6.1-sol` (decisão do usuário: "talvez, num teste manual").

### 6.2 Bench — encerrado (decisão do usuário, 10/11/2026)
> O usuário decidiu não rodar mais o bench, já feito nas duas máquinas. Os limiares abaixo ficam como referência; a avaliação segue pelos testes manuais (§6.1). O `-provedor falso` continua na verificação local, porque não usa rede.

- ~~**Repetir a bateria completa do ChatGPT**~~ (não será feito) depois da mudança 1(a) (pessoa sem nome descartada). Na conferência rápida do pads-01, o JSON 1ª ficou em 7/8. Registre o resultado no `RELATORIO_ETAPA8_BENCH.md`.
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
| Cadastro PADS/SR peça por peça | ✅ PADS e SR testados pelo usuário |
| SEI só com os documentos escolhidos, sem download manual (V4 §48.4 revisada) | ✅ Testado com processo real |
| Enquadramento por catálogo, divergência da §22.1 confirmada, conexão pelo motor | ✅ Feito; ⚠️ curadoria a revisar (§6.3-1) |
| Barreira bloqueia identificador conhecido; mapa nunca sai do computador; restauração determinística | ✅ Feito; vazamento 0 no bench |
| Regras jurídicas, cálculos e golden inalterados; `rulescov -estrito` verde | ✅ |

**O D-11 estará concluído quando** o teste C (Gemini) passar, a curadoria for aprovada e o usuário decidir sobre o ChatGPT abaixo do limiar (§6.3-4). Ao concluir:
- registrar a conclusão no `REGISTRO_DUVIDAS_DECISOES.md` (histórico) e neste guia;
- o usuário decide o merge com a `main`.

### 6.6 Depois do D-11 (cada fase exige autorização própria)
- **F5:** biblioteca jurídica. Recorte essencial **feito** (§5.5), com busca lexical. Faltam a gestão do acervo, a tela de duplicidade e a pesquisa web.
- **F6/F7:** **feitas para o PADS** (§5.7). Falta a SR (DocModel próprio, redação por fato e pessoa, validadores, mesmos arquivos).
- **F8:** instalador (certificado, WiX, Tesseract sem GPL, Liberation Serif).

---

## 7. Como retomar com o Claude
Na nova sessão, cole algo como:

> Projeto sys-executor, branch `add-mode-apikey-oauth` (já fiz `git pull`). Antes de qualquer ação, leia o `GUIA_CONTINUIDADE.md` (comece pelo §6.0, inclusive "O que NÃO refazer e o que não pode regredir"; depois o §3 e o §4), e `docs/planning/RELATORIO_INTERFACE.md`, `RELATORIO_FASE6.md` e `RELATORIO_FASE7.md`. Siga as regras do §3: sem bench, testes manuais no `.exe`, um `.exe` e um roteiro curto a cada ajuste, commit sem coautor só quando eu pedir, nada de GPU, só dados sintéticos, não toque em `logo-cabecalho-docs.jpeg` nem inclua `docs/draft_templates` em commit, não reescreva o que já está pronto, pergunte em caso de dúvida. Prepare esta máquina pelo §4 (Typst, Tesseract e logotipo em `componentes\`, §4.2), rode a verificação do §4.4 e gere o `.exe`. Depois me faça as perguntas do §6.0, passo 1, e siga a ordem do §6.0.

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
9. **PADS, limites conhecidos da minuta:** "o(a) acusado(a)" nos textos fixos; pedido de ajuste que mistura redação e decisão não é aplicado; o texto da IA depende do provedor e do modelo escolhidos; a prévia na tela é HTML, não o PDF.
10. **Typst e logotipo são componentes ao lado do `.exe`** (§4.2). Sem o Typst, "Salvar PDF" dá EXP-002; sem o logotipo, o cabeçalho sai só com o texto.
11. **Arquivos do usuário não rastreados na raiz:** `logo-cabecalho-docs.jpeg` e, na máquina de 11/11/2026, uma cópia dele com o nome `│` (criada por engano num comando de cópia). Nenhum dos dois entra em commit; não apague sem o usuário pedir.
12. **Tela final:** depois de finalizar, o app não sabe quais arquivos foram salvos (a sessão é apagada); por isso o bloco diz "Onde você os gravou", e não "DOCX e PDF salvos".
13. **"Ir para a conferência"** fica sempre clicável, mesmo com peça pendente: a conferência mostra o que falta e o Go valida ao concluir.
