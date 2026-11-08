# RELATÓRIO DA FASE 1 — FUNDAÇÃO (sys-executor)

- **Data:** 08/11/2026
- **Autorização:** expressa do usuário ("Pode iniciar a Fase 1 (Fundação)").
- **Escopo:** F1-01 a F1-07 do plano (§16). Nenhuma etapa de análise jurídica, IA ou SEI foi implementada: elas pertencem às Fases 2 em diante.
- **Ambiente de verificação:** WSL2 (Go 1.27.3, Node 24) e o Windows 11 do usuário (executável real e testes nativos).

## 1. Critérios de saída (PLANO_TESTES_ACEITACAO §12: T01, T02, T07, T12, SEG-DIR-01, SEG-LOG-01)

| Critério | Como foi verificado | Resultado |
|---|---|---|
| **T01** — 1ª escolha PADS/SR; 2ª SEI/PDF | Go: `TestT01OrdemDasEscolhas` (entrada antes do tipo → TAR-002; troca de tipo só antes da entrada; voltar preserva o tipo). Interface: `telas.test.ts` (a 1ª tela só oferece PADS e SR; a entrada aparece depois, com a faixa do tipo) | **OK** |
| **T02** — upload sem credencial nem SEI | Go: `TestT02UploadNaoPassaPeloSEI` (nenhum estado `SEI_*` no histórico; PDF vazio/inválido mantém a tarefa aguardando). Interface: nenhum campo de senha no fluxo de upload | **OK** |
| **T07** — status sem chatbot nem streaming | Interface: linha do tempo e descrição do estado; sem `textarea`/campo de texto; sem métrica real → só indicação de atividade, nunca percentual; com métrica real → barra e "30 de 120 páginas". Go: `TestSemProgressoSemMetrica` | **OK** |
| **T12 / SEG-CRASH-01** — queda → resíduos removidos na abertura, sem retomada | Go, **nativo no Windows**: `TestVarreduraRemoveResiduoDeQueda` mata à força um processo filho com sessão aberta; a varredura preserva a sessão viva e remove a do processo morto; o conteúdo nunca aparece em claro no disco. App real: pasta de "sessão de processo morto" plantada em `%LOCALAPPDATA%\sys-executor\trabalho` foi removida ao abrir o app. Não existe nenhuma API de retomada | **OK** |
| **SEG-DIR-01** — ACL do diretório efêmero | Go, **nativo no Windows**: `TestProtegerDiretorioSomenteUsuario` — DACL protegida (sem herança), **uma única ACE** (controle total do usuário atual), também nos arquivos criados dentro. Conferido com `Get-Acl` na pasta real do WebView2: só `STRIBEIRO\diogo`, `protegida=True` | **OK** |
| **SEG-LOG-01** — nenhum arquivo de log | App real: o logger do Wails descarta tudo; binário `-H windowsgui` (sem console). O WebView2 cria arquivos internos (`LOG`, `*.log`, `Crashpad`) **na pasta própria do app**; após o fechamento normal, a pasta fica **vazia em ~2 s**. Após queda forçada (`taskkill /F`), os arquivos ficam até a próxima abertura, que os apaga (V4 §29.2). Nenhuma pasta criada em `%APPDATA%` | **OK** (ver §3, item 1) |

Também verificados: **SES-CONC-01** (uma tarefa por vez), **SEG-WER-01** (exclusão do WER registrada pelo app real: `HKCU\…\ExcludedApplications\sys-executor.exe = 1`), **GOV-LOCK-01** e **GOV-COV-01** (esqueleto: 124/124 IDs de teste da V4 presentes na matriz), **ERR** (catálogo completo, amigável, sem termos técnicos nem sucesso simulado).

## 2. O que foi construído

| Tarefa | Entrega |
|---|---|
| F1-01 Estrutura | Módulo `github.com/DiogoBrazil/sys-executor`; `cmd/sys-executor` (app Wails v3.1.0), `cmd/rulescov`, `cmd/licencas`; `internal/{app,workflow,session,prefs,erros,pastas,platform/win}`; `frontend/` (Vue 3 + TypeScript + Vite); `policies/` |
| F1-02 Sessão efêmera | Pasta aleatória por tarefa em `%LOCALAPPDATA%\sys-executor\trabalho`, DACL só do usuário; conteúdo em AES-256-GCM com chave só em memória e nomes opacos (PT-F1-01); marcador com PID + instante de criação do processo (não confunde PID reaproveitado); encerramento com sobrescrita + remoção (melhor esforço, sem promessa forense); varredura de resíduos na abertura; tipo `Segredo` para credenciais (zerável). Exclusão do WER (P-10) |
| F1-03 FSM + eventos | 28 estados do plano §6, tabela de transições testada (alcançabilidade, terminais sem saída, CANCELADA/FALHA de qualquer estado ativo, proibições de T01/T02/A31); pausa para esclarecimento com retorno à origem; etapas executadas em segundo plano com cancelamento; evento `tarefa:visao` para a interface; progresso só com métrica real |
| F1-04 Telas | Início (duas "capas de autos": PADS em bordô, SR em azul-petróleo), Entrada (SEI/RO ou PDF), Processamento (linha do tempo, estado atual, cancelamento com confirmação, falha com mensagem do catálogo). Faixa do tipo escolhido fixa nas telas seguintes. Fontes OFL empacotadas (funciona offline); CSP restritiva; menu de contexto do navegador desativado |
| F1-05 Preferências | `%LOCALAPPDATA%\sys-executor\preferencias.json`, formato fechado (chaves desconhecidas → padrão), gravação atômica; pesquisa web **desligada** por padrão (T19); estado da GPU com a **sentinela da sonda** (D-09.1): abrir com a sonda "em andamento" marca a GPU como instável |
| F1-06 Catálogo de erros | 40 códigos em 14 categorias (SEI, rede, PDF, OCR, arquivo, modelo, biblioteca, referência, validação, exportação, pesquisa, sessão, tarefa, interno). A interface recebe só título, mensagem e ação (PT-F1-02) |
| F1-07 CI | `.github/workflows/ci.yml`: governança (`rulescov -estrito`, `licencas`), Go no Linux (gofmt, vet, testes com `-race`), Go no Windows (ACL, WER, processos, T12), frontend (tipos, testes, build, `npm audit --omit=dev`) e compilação do executável Windows |

## 3. Pontos de atenção

1. **WebView2 e SEG-LOG-01.** O WebView2 sempre grava arquivos internos do Chromium enquanto o app está aberto, e o Wails não expõe o modo InPrivate. A mitigação adotada (PT-F1-03) mantém tudo numa pasta restrita ao usuário e a esvazia ao fechar e ao abrir. O critério passa a ser lido como "nada resta depois que o app fecha".
2. **CI ainda não rodou no GitHub.** Todos os passos foram executados localmente (Linux e Windows nativo) e passaram, mas o workflow só roda após um `push`, que não foi feito.
3. **Lint.** O CI usa `gofmt` e `go vet`. Uma ferramenta externa de lint (staticcheck/golangci-lint) não foi adicionada: a execução de ferramentas baixadas foi bloqueada no ambiente local e não há como validar a versão compatível com Go 1.27. Pode entrar quando o CI rodar no GitHub.
4. **Licenças.** Verificador próprio (`cmd/licencas`), sem dependências: 3 módulos Go no executável (Wails MIT, go-ole MIT, x/sys BSD-3) e 27 pacotes npm de produção (MIT/BSD/ISC/Apache-2.0; fontes OFL-1.1). Os 5 alertas de `npm audit` são só de dependências de teste (`@vue/test-utils` → `js-beautify` → `glob`/`minimatch`).
5. **Fechar com tarefa ativa** descarta a tarefa sem perguntar (V4 §29.1). Uma confirmação ao fechar a janela pode ser adicionada na Fase 2.

## 4. Próxima fase

**Fase 2 — Ingestão** (plano §16): SEI via chromedp (fluxo mapeado na F0-08), extração com go-pdfium, OCR, segmentação, evidências e mapa de cobertura. Pendências que dependem do usuário antes dela: o valor de `acao_ajax` e a observação local de um PDF consolidado real (P-09).
