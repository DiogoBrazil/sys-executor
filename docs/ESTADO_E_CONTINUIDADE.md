# ESTADO DO PROJETO E GUIA DE CONTINUIDADE (sys-executor)

> **Para que serve:** quem continuar o trabalho (outra pessoa, outra máquina ou outra sessão de IA) lê este arquivo primeiro. Ele diz o que está pronto, o que falta, em que ordem fazer, como montar o ambiente e quais regras não podem ser quebradas, para não haver regressão nem retrabalho.
>
> **Manter atualizado:** ao fim de cada fase (ou de qualquer mudança relevante de estado), atualize as seções 2, 4 e 5 e a data abaixo, no mesmo commit do relatório da fase.

- **Última atualização:** 08/11/2026, depois da Fase 4 (commit `0d239d1` na branch `development`).
- **Situação em uma linha:** Fases 0 a 3 concluídas; Fase 4 implementada e com CI verde, **aguardando o teste do usuário no Windows**; Fases 5 a 8 por fazer, cada uma só com autorização expressa do usuário.

---

## 1. Ordem de leitura

1. **Este arquivo**, inteiro.
2. `docs/rules/sys-executor_especificacao_consolidada_V4.md`: a especificação (V4). É a fonte da verdade funcional e jurídica, e o hash dela está travado em `policies/V4.lock`.
3. `docs/planning/PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`: arquitetura, fases e tarefas (§16).
4. `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`: decisões do usuário (D-01…D-09), interpretações (I-01…I-05), pendências (P-xx) e decisões técnicas (PT-Fx-yy). **Nada ali pode ser desfeito sem falar com o usuário.**
5. `docs/planning/RELATORIO_FASE0.md` … `RELATORIO_FASE4.md`: o que foi feito, os achados e as medições de cada fase.
6. Conforme a tarefa:
   - `PLANO_TESTES_ACEITACAO.md`: IDs de teste e critérios de saída por fase (§12);
   - `MATRIZ_REQUISITOS_V4.md`: requisito → regra → módulo → teste;
   - `REGRAS_IA_ESTRATEGIA.md`: como as regras chegam à IA;
   - `ADR_00x`: escolhas de arquitetura.

O arquivo `PLANO DE IMPLEMENTAÇÃO — SYS-EXECUTOR.md` na raiz é o plano original, mantido como histórico. O plano vigente é o de `docs/planning/`.

---

## 2. Regras de trabalho que não podem ser quebradas

| Regra | Origem |
|---|---|
| **Nenhuma fase começa sem autorização expressa do usuário.** Dúvida de decisão → perguntar, não supor | Usuário (desde a F0) |
| **Commits sem Claude como coautor** e sem qualquer atribuição a IA (nada de `Co-Authored-By`) | Usuário |
| **Não rodar nada em GPU/Vulkan na máquina do usuário.** Em 08/11/2026 a máquina caiu durante um teste Vulkan (RX 6600 XT). CPU apenas; a GPU real só se valida em outra máquina | D-08 |
| **Nenhum dado real no repositório nem na CI** (autos, nomes, CPFs, números de processo, e-mails vistos em prints). Fixtures são sempre fictícias | Plano de testes |
| **SEI: nenhuma ação de escrita.** Guarda CDP deny-by-default; nunca marcar `chkLiberar`, nunca acionar "Desativar 2FA", nunca recarregar a página de 2FA | V4 §§1.2, 13; F0-08 |
| Credenciais e códigos só em memória (`[]byte`), zerados depois; nunca em disco, log ou evento | V4 §33 |
| Sem logs em disco (app, sidecars, WebView2); diagnóstico só em memória (`SYSEXEC_DIAGNOSTICO=1`) | V4 §33; SEG-LOG-01 |
| Push só na branch `development` (autorizado na F2). Nada em `main` sem o usuário pedir | Usuário |
| Comunicação e textos da interface em **pt-BR** | Usuário |
| Mudança na V4 → atualizar catálogo → `rulescov` → registrar em `policies/CHANGELOG.md` → `-atualizar-lock`. Mudança de regra jurídica passa pelo usuário | GOV-LOCK-01 |

---

## 3. Arquitetura em 1 minuto

- **Aplicativo:** Go + Wails v3 (WebView2) + Vue 3/TypeScript, executável Windows x64 sem CGO e sem console.
- **Tarefa:**
  - cada tarefa é uma máquina de estados com 28 estados (`internal/workflow`);
  - a sessão é efêmera e cifrada (AES-256-GCM, chave só em memória);
  - uma tarefa por vez, sem retomada depois de fechar o app.
- **Etapas do processamento:**
  1. **Obtenção:** SEI pelo Edge (chromedp) ou envio do PDF.
  2. **Extração:** pdfium em WebAssembly e Tesseract para OCR.
  3. **Classificação:** segmentação pelo carimbo do SEI.
  4. **Compatibilidade:** confere se os autos batem com o tipo escolhido.
  5. **Delimitação:** a IA propõe, o usuário confirma e o Go congela.
  6. **Análise:** IA mais confirmações (etapas 5 a 9 do plano §5.4).
  7. **Ainda não implementadas:** consulta à biblioteca (F5), cálculo (o motor da F3 existe, mas não está ligado ao fluxo), redação, validação e revisão (F6 e F7).
- **Princípio da IA:**
  - a IA interpreta e redige, e o Go calcula, confere e impõe as invariantes;
  - o usuário decide e o validador bloqueia;
  - toda citação da IA é conferida literalmente no PDF;
  - as regras chegam à IA pelo catálogo YAML, selecionadas por chave, nunca truncadas.
- **IA local:**
  - `llama-server` (llama.cpp b12207) como sidecar: loopback, chave aleatória, `--offline`, Job Object;
  - modelos GGUF Qwen3.5-4B (padrão) e 2B (básico).

---

## 4. O que está pronto

### 4.1 Por fase

| Fase | Situação | Relatório | Validação |
|---|---|---|---|
| F0 Viabilidade | Concluída | RELATORIO_FASE0 | PoCs em `poc/` (descartáveis) |
| F1 Fundação | Concluída | RELATORIO_FASE1 | T01, T02, T07, T12, SEG-DIR-01, SEG-LOG-01 |
| F2 Ingestão + SEI | Concluída | RELATORIO_FASE2 | Usuário: SEI/RO real (login, 2FA, sigiloso, gerar, baixar) e PDF real de 67 páginas (30 peças = árvore) |
| F3 Motor determinístico | Concluída | RELATORIO_FASE3 | 101 casos da seção 1; `rulescov -estrito` |
| F4 IA local e orquestração | **Implementada; aguarda o teste do usuário no Windows** | RELATORIO_FASE4 | CI verde; benchmark em CPU (4B passa nos limiares); teste real pendente (§5.1) |

### 4.2 Mapa do código

| Pasta / pacote | Responsabilidade | Fase |
|---|---|---|
| `cmd/sys-executor` | `main.go`: liga tudo (pipeline, SEI, IA, serviços Wails); `faxina.go`: limpeza do WebView2 por relançamento | F1–F4 |
| `cmd/rulescov` | GOV-LOCK-01 (hash da V4) e GOV-COV-01 (catálogo × matriz × V4 × código × testes) | F1, F3 |
| `cmd/licencas` | Licenças das dependências (só permissivas) | F1 |
| `cmd/autos-mapa` | Diagnóstico de estrutura de PDF real **sem texto** (para o usuário rodar) | F2 |
| `cmd/bench` | Benchmark de modelos sobre os casos sintéticos | F4 |
| `internal/workflow` | FSM, perguntas com campos, respostas revisáveis, parada recuperável, visão para a UI | F1, F4 |
| `internal/session`, `pastas`, `prefs`, `platform/win`, `erros` | Sessão cifrada, pastas, preferências mínimas, ACL/WER/Job Object/memória, catálogo de erros | F1, F4 |
| `internal/ingest/{pdf,ocr,autos,segment}` | Extração, OCR, índice FTS5 em memória, datas, segmentação, compatibilidade, mídias | F2 |
| `internal/sei` | Cliente SEI (chromedp + guarda) | F2 |
| `internal/rules/{civil,engine,catalog}` | Datas civis, motor (reincidência, equivalências, pontuação, dosimetria, memória de cálculo), catálogo YAML de 185 regras | F3, F4 |
| `internal/domain/{pads,sr}` | Invariantes de escopo (ImputacaoSet congelado, um acusado, autorizações da SR…) | F3 |
| `internal/inference` | GGUF, catálogo de modelos, hardware, orçamento, recomendação, gerenciador (download/importação), sidecar, GPU segura | F4 |
| `internal/policy` + `policies/` | Compilação dos prompts por etapa (núcleo + regras + pack), golden files, orçamento *fail closed* | F4 |
| `internal/analise` | Etapas de IA: classificação residual, delimitação, extração, análise; conferência literal e validações | F4 |
| `internal/pipeline` | Etapas por estado (verificar, extrair, classificar, compatibilidade, delimitar, analisar), DAG de invalidação, painel | F2, F4 |
| `internal/app` | Serviços Wails (tarefa, fontes/PDF, preferências, modelos) e `MotorLocal` | F1–F4 |
| `internal/testutil/*` | PDF sintético, simulador do SEI, llama-server falso, GGUF sintético, casos com gabarito | F2–F4 |
| `frontend/src` | Telas Início, Entrada, Processamento, Modelos; cartão de pergunta com formulário; painéis de autos e de análise; visualizador PDF.js | F1–F4 |
| `testdata/autos` | Fixtures fictícias (`.typ` → `.pdf` com Typst 0.15.1) | F2 |

### 4.3 Onde a tarefa pára hoje

1. **Sem modelo de IA instalado:** a tarefa pára na delimitação com MOD-001, em parada recuperável. O usuário instala o modelo pela tela Modelos e clica em "Tentar novamente".
2. **Com modelo instalado:** a tarefa percorre a delimitação, as confirmações, a leitura das peças, as mídias, a análise e as confirmações da etapa 9. Depois pára em `CONSULTANDO_FONTES`, com TAR-004 ("próximas etapas ainda não disponíveis").
3. **O que o usuário ainda pode fazer nessa parada:** consultar os autos e a análise, alterar respostas e cancelar.

---

## 5. O que falta — passos até concluir

### 5.1 Fechar a Fase 4 (antes da Fase 5)

1. **Teste do usuário no Windows** com o pacote `C:\Users\diogo\Documents\sys-executor-teste` (app + `llama\cpu` + `componentes\tesseract`; o roteiro está no RELATORIO_FASE4 §5):
   - importar o modelo de `%LOCALAPPDATA%\sys-executor-dev\models\Qwen3.5-4B-Q4_K_M.gguf`, ou baixá-lo pela tela Modelos;
   - processar um PADS real e conferir delimitação, "Ver no PDF", perguntas e "Alterar";
   - anotar os tempos.
2. Corrigir o que o teste mostrar (como na F2, por rodadas) e registrar no RELATORIO_FASE4.
3. **Decisão do usuário pendente:** manter o Qwen3.5-2B como perfil básico, com aviso (recomendado), ou retirá-lo do catálogo.
4. Marcar a F4 como concluída no plano, no relatório e neste arquivo, e pedir autorização para a Fase 5.

### 5.2 Pendências herdadas de fases anteriores (não esquecer)

| Item | Origem | Quando |
|---|---|---|
| Ler a **árvore de documentos do SEI** (nome e tipo, sem conteúdo) e comparar com as peças do PDF, para apontar mídias e arquivos fora do PDF (ex.: o vídeo do PDF real) | RELATORIO_FASE2 §4 (registrada para a F4, **não implementada**) | Junto da F5 ou da F6, com aval do usuário |
| SR-EST-010 (papel da pessoa por ato) e SR-ESC-004 (situação funcional na data de cada fato) | RELATORIO_FASE3 §4 | F6 (redação da SR) |
| Ligar o **motor de cálculo** (F3) ao estado `CALCULANDO`, com as respostas da etapa 9 (comportamento, pontos, histórico, reincidência, conexão, serviços, circunstâncias confirmadas), e usar `pads.VerificarCalculo` | Plano §5.4 etapa 10 | F5/F6 (o estado vem logo depois de `CONSULTANDO_FONTES`) |
| Caso de 29/02 e saldo negativo **materiais**: perguntar ao usuário (o motor já devolve a pendência) | P-03, P-04, PT-F3-01 | Ao ligar o cálculo |
| Autorização de fato/militar novo da SR **revisável** (hoje é definitiva) | PT-F4-08 | F6, se o usuário quiser |
| Tesseract recompilado sem libtiff/jbig/libarchive/OpenSSL (sem GPL/LGPL) e com strip | D-09.7 | F8 |
| Liberation Serif (OFL) empacotada como fallback de fonte | D-09.8 | F7 |
| Validação real de **GPU** (sonda + priorização, T14) e de máquina com **8 GB** (T16) | D-08; RELATORIO_FASE4 | Outra máquina; F8 (matriz de instalação) |
| Instalador: `llama\cpu` (e `llama\vulkan` opcional) ao lado do executável; `componentes\tesseract` | PT-F4; F2 | F8 |

### 5.3 Fases restantes (plano §16; critérios em PLANO_TESTES_ACEITACAO §12)

Todas exigem autorização expressa. A ordem recomendada é esta:

1. **Fase 5 — Biblioteca jurídica** (critério: seção 5 do plano de testes)
   - **F5-01:** parser de normas (artigo, parágrafo, inciso, alínea), com o RDPM-RO como referência.
   - **F5-02:** bootstrap dos 14 PDFs de `docs/law_library`: OCR onde preciso; duplicidade RDPM-RO × Coletânea, que o usuário decide (P-11).
   - **F5-03:** FTS5 + vetores (Qwen3-Embedding-0.6B, já no catálogo de modelos), com busca híbrida.
     - Biblioteca inicial **pré-indexada no build** (D-09.9).
     - Instrução da consulta em inglês, chunk com cabeçalho hierárquico, tesauro e held-out (D-09.10).
   - **F5-04:** notas de vigência. **RDPM art. 41 § 3º considerado revogado** (D-01), filtrado e bloqueado.
   - **F5-05:** gestão da biblioteca: incluir, substituir, excluir, e bloquear o documento em uso (BIB-002).
   - **F5-06:** pesquisa web opcional, desligada por padrão, com sanitizador (nada dos autos sai).
   - **Ligação:** implementar a etapa de `CONSULTANDO_FONTES` no pipeline e o enquadramento da SR com fonte confirmada (SR-SOL-005, PT-F3-05).
2. **Marco M1** (PADS-E01…E05 de ponta a ponta com minuta esqueleto validada): exige o cálculo ligado e a composição mínima da F6.
3. **Fase 6 — Composição e validação** (critério: seções 2, 3 e 6; T09, T10, T24)
   - DocModel PADS e SR, e textos fixos parametrizados (plano §5.5).
   - Redação por slots (packs `redacao`, já com regras no catálogo).
   - Validadores do §2.6 da estratégia: ScopePADS, ScopeSR, EvidenceRefs, SEIIds, LegalRefs, CalcConsistency, UserDecisions, RequiredParts, NoFakeActs, SupportCheck.
   - Laço de correção e ajustes que preservam os slots.
4. **Fase 7 — Documentos e revisão** (critério: seção 10, T11, SR34). **Exige D-04/P-01:** materiais institucionais do usuário (modelos, cabeçalho, brasão).
   - DOCX + Typst institucionais, prévia, downloads, aviso de revisão.
   - "Aprovar e finalizar" com limpeza.
5. **Fase 8 — Endurecimento e entrega** (critério: seções 8, 9 e 11 + riscos residuais)
   - Segurança e desempenho 8/16/32 GB.
   - Instaladores NSIS/MSI. Pendências P-06 (certificado Authenticode) e P-07 (taxa OSMF do WiX).
   - Matriz de instalação, manual pt-BR e relatório de riscos.

### 5.4 Ao iniciar qualquer fase

1. Confirmar a autorização do usuário.
2. Ler a seção da fase no plano, os IDs de teste dela e as regras do catálogo (`internal/rules/catalog/*.yaml`, campo `modulo`).
3. Implementar com testes primeiro nos pacotes puros e, depois, ligar ao pipeline e à UI.
4. Testar com componentes reais quando houver (modelo em CPU, Tesseract, Edge) e registrar medições.
5. Rodar a "definição de pronto" (§7).
6. Escrever `RELATORIO_FASEn.md`, as decisões PT-Fn-xx no registro, a saída da fase no plano e atualizar **este arquivo**.
7. Commits sem coautor, push na `development`, CI verde. Relatar ao usuário e pedir autorização para a próxima fase.

---

## 6. Montar o ambiente em outra máquina

### 6.1 Ferramentas

- **Go 1.27.x** (go.mod declara 1.26.0; CI usa `1.27.x`) e **Node 24**.
- `gh` (GitHub CLI) autenticado.
- Wails CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`. O binário fica em `$(go env GOPATH)/bin`; ponha esse caminho no `PATH`.
- Desenvolvimento usado até aqui: **WSL2 (Ubuntu) no Windows 11**. O produto é só Windows; no Linux compila-se com `GOOS=windows`. O pacote `cmd/sys-executor` não compila para Linux (exige GTK), e a CI não o testa no Linux.

### 6.2 Primeira vez

```bash
git clone <repo> && cd sys-executor && git checkout development
cd frontend && npm ci && npm run build && npm test && cd ..
go test ./internal/... ./cmd/rulescov/... ./cmd/licencas/...
go run ./cmd/rulescov -estrito && go run ./cmd/licencas
```

### 6.3 Artefatos externos (fora do Git; baixar e conferir o SHA-256)

Na máquina original eles ficam em `~/.cache/sys-executor-dev/` (`bin/`, `models/`). Recrie a mesma estrutura.

| Artefato | Origem | SHA-256 |
|---|---|---|
| llama.cpp b12207 Linux (testes reais no WSL) | `https://github.com/ggml-org/llama.cpp/releases/download/b12207/llama-b12207-bin-ubuntu-x64.tar.gz` → extrair em `bin/linux/llama-b12207/` | `3dbcc99e866441ba2fd541561c90d09581ed10f4c56d0cca2461c0b1f311efe7` |
| llama.cpp b12207 Windows CPU (vai para `llama\cpu\` ao lado do exe) | `…/b12207/llama-b12207-bin-win-cpu-x64.zip` → `bin/win-cpu/` | `24447c546dca02bc449c7369eba07608cf5b170a59042ef388712654356d0d4c` |
| llama.cpp b12207 Windows Vulkan (opcional; **nunca rodar na máquina do usuário**, D-08) | `…/b12207/llama-b12207-bin-win-vulkan-x64.zip` | `c8c84eff8b7eae83602d3cf2dd6defe16c47d1011b4a2514905c1817e3a909ca` |
| Modelos GGUF (Qwen3.5-4B, Qwen3.5-2B, Qwen3-Embedding-0.6B) | URLs e SHA-256 em `internal/inference/catalogo_modelos.json` (Hugging Face, fixadas em commit) → `models/` | ver o catálogo |
| Tesseract 5.4.0.20240606 (UB Mannheim) — teste; a versão do produto será recompilada (D-09.7) | `tesseract-ocr-w64-setup-5.4.0.20240606.exe` → runtime mínimo em `componentes\tesseract\` (exe + DLLs + `tessdata\`) | `c885fff6998e0608ba4bb8ab51436e1c6775c2bafc2559a19b423e18678b60c9` |
| tessdata_best `por` e `osd` | repositório `tesseract-ocr/tessdata_best` | `711de9dbb8052067bd42f16b9119967f30bada80d57e2ef24f65d09f531adb04` (por) · `9cf5d576fcc47564f11265841e5ca839001e7e6f38ff7f7aacf46d15a96b00ff` (osd) |
| Typst 0.15.1 (só para regenerar fixtures `testdata/autos/*.typ`) | GitHub typst/typst | linux `a6d077d0a95eed5a2eba715b2dae06be954f624ccbf85758a03f389ded33118c` · windows `19ce3551153c2fe7ee9fa2f95208310c8f4d3209fedb699e0333faf8913f6736` |

### 6.4 Pacote de teste no Windows (como o usuário testa)

```
C:\Users\<usuário>\Documents\sys-executor-teste\
  sys-executor.exe            GOOS=windows … -ldflags "-H windowsgui -s -w" ./cmd/sys-executor
  llama\cpu\llama-server.exe  + DLLs do zip win-cpu (+ LICENSE*)
  componentes\tesseract\      tesseract.exe + DLLs + tessdata\por, osd
  autos-mapa.exe              (diagnóstico de estrutura de PDF, opcional)
```

- **Executar o exe:** copie para uma pasta do Windows antes. Ele não abre a partir de `\\wsl.localhost\…`.
- **Executável em uso:** se a cópia falhar porque o app está aberto, gere com outro nome e peça ao usuário para fechar o app.
- **Dados do app:** ficam em `%LOCALAPPDATA%\sys-executor\`, com `trabalho\`, `webview\`, `modelos\` e `preferencias.json`. Não há logs.

### 6.5 Variáveis de ambiente

| Variável | Uso |
|---|---|
| `SYSEXEC_DIAGNOSTICO=1` | Mostra a causa técnica mascarada dos erros + "Copiar diagnóstico" |
| `SYSEXEC_SEI_VISIVEL=1` | Mostra a janela do Edge do SEI |
| `SYSEXEC_SEI_URL` | Aponta o SEI para o simulador (`http://127.0.0.1:…`) ou outro https |
| `SYSEXEC_NAVEGADOR` | Caminho do Edge/Chrome (testes) |
| `SYSEXEC_TESSERACT`, `SYSEXEC_TESSDATA` | Tesseract fora de `componentes\tesseract` |
| `SYSEXEC_LLAMA_SERVER`, `SYSEXEC_LLAMA_VULKAN` | llama-server fora de `llama\cpu` / `llama\vulkan` |
| `SYSEXEC_MODELO_TESTE` | GGUF para os testes de integração reais (`internal/inference`, `internal/analise`) |
| `SYSEXEC_BASE` | Base de dados do app (testes) |
| `SYSEXEC_LLAMA_FALSO` | Uso interno dos testes (llama-server falso) |

Exemplos (WSL, CPU):

```bash
L=~/.cache/sys-executor-dev
SYSEXEC_LLAMA_SERVER=$L/bin/linux/llama-b12207/llama-server SYSEXEC_MODELO_TESTE=$L/models/Qwen3.5-4B-Q4_K_M.gguf \
  go test -run Integracao -v -timeout 30m ./internal/inference/ ./internal/analise/
go run ./cmd/bench -servidor $L/bin/linux/llama-b12207/llama-server -modelo $L/models/Qwen3.5-4B-Q4_K_M.gguf -saida /tmp/bench.md
```

---

## 7. Definição de pronto (rodar antes de cada commit relevante)

```bash
set -o pipefail                               # senão um "| tail" esconde falha
gofmt -l internal cmd policies                # vazio
go vet ./internal/... ./cmd/rulescov/... ./cmd/licencas/... ./cmd/bench/...
GOOS=windows go vet ./cmd/sys-executor/ ./internal/...
go test -race -timeout 25m ./internal/... ./cmd/rulescov/... ./cmd/licencas/...
go run ./cmd/rulescov -estrito && go run ./cmd/licencas
(cd frontend && npx vue-tsc --noEmit && npm test && npm run build)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o /tmp/sys-executor.exe ./cmd/sys-executor
```

Procedimentos:

| Situação | O que fazer |
|---|---|
| Mudou serviço em `internal/app` | `GOOS=windows wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor` e commitar `frontend/bindings`. O aviso sobre `Middleware`/`http.Handler` é antigo e esperado |
| Mudou regra do catálogo ou pack em `policies/` | `go test ./internal/policy -atualizar`, **revisar o diff** dos golden em `internal/policy/testdata/golden` e commitar junto |
| Mudou a matriz | Catálogo e matriz precisam ter o mesmo conjunto de IDs (`rulescov`) |
| Teste novo de regra | Anotar `// Cobre: <caso do plano>` e/ou `// Regra: <ID do catálogo>` |
| Erro novo | Código novo **no fim do grupo** em `internal/erros/catalogo.go`; nunca reutilizar código |
| Dependência Go nova | Conferir licença (`cmd/licencas`) e o `go.mod` inteiro depois do `go get` |

---

## 8. Convenções do código

- **Nomes em pt-BR**, comentários explicando o porquê e a origem (V4 §x, D-0x, PT-Fx-yy).
- **Serviços Wails nunca devolvem `error`:** devolvem `Resposta{visao, erro}` com o erro do catálogo, para a causa técnica não chegar ao JavaScript (PT-F1-02).
- **Erros** sempre do catálogo (`erros.Novo` ou `erros.Com`). Erro de etapa com o próximo estado igual ao atual é parada recuperável (PT-F4-09).
- **Saída da IA:**
  - decodificação estrita (`DisallowUnknownFields`);
  - todo enum também conferido no Go, sem confiar só no schema;
  - todo trecho conferido literalmente (`analise.LocalizarComOmissao`);
  - nunca aceitar o que não confere.
- **Regras para a IA** vêm do catálogo (`instrucao_ia`, `etapas`, `condicao`), nunca escritas soltas no código. Os packs têm só a tarefa da etapa e exemplos de formato.
- Fim de linha **LF** em tudo (`.gitattributes`), porque o hash do `policy_bundle` depende disso.
- **Fixtures** fictícias; textos de teste sem dados reais.
- **UI:**
  - estilo da "capa dos autos" (tokens em `frontend/src/estilo.css`);
  - `data-teste` para os testes;
  - acessibilidade com foco visível e `aria-live`.

---

## 9. Armadilhas já encontradas (não repetir)

| Armadilha | Como evitar |
|---|---|
| `go test … \| tail` com `&& git commit` commitou com teste falhando | `set -o pipefail` |
| `go get` de uma versão mais velha rebaixou o go-pdfium | Usar a versão já exigida e conferir o `go.mod` |
| CRLF no checkout do Windows mudou os golden files e o hash do bundle | `.gitattributes` com `eol=lf` (já feito) |
| App com `-H windowsgui` quebrava o wazero/pdfium ao escrever em stdout | Stdout/Stderr do wazero em `io.Discard` (já feito; há teste) |
| `wails3` não está no PATH | `$(go env GOPATH)/bin/wails3` |
| Executável aberto a partir de `\\wsl.localhost` não roda | Copiar para `C:\…` |
| Vulkan derrubou a máquina do usuário | Nada de GPU nela (D-08) |
| O modelo real cita frases pulando linhas, mantém o hífen de quebra, troca inciso/alínea/diploma e **inventa artigo de lei** | Já tratado no Go (PT-F4-03 a 06); reteste a cada troca de modelo com `cmd/bench` |
| O 2B gera "mídias" falsas e é influenciado por injeção | Só perfil básico; validador obrigatório |
| Senha do SEI/RO é um campo mascarado (`pwdSenha` visível sem `name`) e a geração usa `optTodos` | Ver `internal/sei/telas.go` e RELATORIO_FASE2 |
| Em JS, `2 << 30` estoura 32 bits | Usar `2 * 2 ** 30` |

---

## 10. Medições de referência (para detectar regressão)

- **Ingestão:** 806 páginas extraídas em 0,67 s na F0 e autos sintéticos de 800 páginas processados em 0,4 s na F2 (pdfium-wasm). O PDF real de 67 páginas rendeu 30 peças.
- **IA em CPU (Ryzen 5 5600X, 6 núcleos):**
  - Qwen3.5-4B: 8,8 a 11 tokens/s, pico de 6,5 GB a 8.192 de contexto, 2 a 5 min por caso sintético;
  - Qwen3.5-2B: cerca de 20 tokens/s e 4,2 GB.
- **Benchmark F4 (4B):** delimitação 7/7, JSON na 1ª tentativa 40/41, resultado conforme o gabarito 8/9, injeção 5/5, violações aceitas 0. Os limiares estão em PLANO_TESTES_ACEITACAO §9.
