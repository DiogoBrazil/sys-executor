# Relatório — etapa 7 do plano D-11: SEI com seleção dos documentos no computador

- **Data:** 10/11/2026. **Branch:** `add-mode-apikey-oauth`.
- **Autorização:** etapas 5 a 8 já autorizadas. Depois do mapeamento real, o usuário escolheu a **opção A** ("Pode continuar com 1, 2 e 3") e aprovou a revisão da V4 §48.4 ("Aprovo o texto").
- **Dados:** testes automatizados só com dados sintéticos. O mapeamento real registrou só a **forma** da tela, sem títulos nem números.

## 1. O que o mapeamento real mostrou (`poc/sei-mapa`, 10/11/2026)
- A tela "Gerar Arquivo PDF do Processo" (`procedimento_gerar_pdf`) **não lista os documentos**. Com "Apenas selecionados", a `selDocumentosApenas` começa vazia.
- A escolha é feita por uma lupa (`infraLupaSelect`), que abre **`documento_selecionar`** num modal. Essa tela tem uma tabela com `chkInfraItem#`, filtros e os botões "Pesquisar" e "Transportar".
- Depois de "Transportar", a *select* recebe as opções `<número> - <título>`. O POST leva `rdoTipo=A`, `selDocumentosApenas` (os ids) e `hdnDocumentosApenas`, no formato da lupa: `id±texto¥id±texto`.
- **Consequência:** "Apenas selecionados" exige `documento_selecionar`, que está **fora** da lista permitida da guarda. A V4 §48.4 não admite ampliar essa lista.

## 2. Decisão (D-11, E2 revisado)
- **Opções apresentadas:**
  - **(A)** gerar "Todos" e selecionar no app;
  - **(B)** liberar `documento_selecionar` na guarda;
  - **(C)** imitar a lupa a partir da árvore.
- **Escolhida: A.** O SEI continua no fluxo validado desde a Fase 2, sem nenhuma ação nova.
- **Texto da V4 §48.4 revisado e aprovado.** Também foram ajustados a linha 1 da §48.2, o CHANGELOG e o `V4.lock`.

## 3. O que mudou
| Camada | Mudança |
|---|---|
| `ingest/autos` | `LerTextoNativo`: lê só o texto nativo e os carimbos, **sem OCR**. O carimbo do SEI é texto mesmo em página escaneada. O OCR fica para os documentos escolhidos. |
| `ingest/pdf` | `Separador`/`Separar`: divide o PDF em memória por faixas de páginas, numa única instância do pdfium. Faixa fora do documento é recusada. |
| `pipeline/sei.go` | Depois do download, `separarSEI` segmenta pelo carimbo (`segment.Segmentar`), separa um PDF por documento e sugere o tipo de peça pelo título (`sugestaoPeca`).<br>No PADS, o interrogatório do acusado vai em "Outros documentos" (plano §8.1, pendente).<br>Os documentos vão ao cadastro (`RECEBENDO_UPLOAD`). **O PDF completo não é gravado na sessão.** |
| `workflow` | `DocumentoSEI` (id, nº SEI, título, páginas, sugestão, papel, peça em que foi usado).<br>`Contexto.ReceberDocumentosSEI`: grava cada documento cifrado na sessão e prepara o cadastro.<br>`Gerenciador.AnexarDocumentosSEI`: anexa à peça com `Origem: "SEI"` e preenche o nº SEI da peça quando ela tem um só documento. Documento repetido → ARQ-004.<br>Transição nova: `SEI_BAIXANDO → RECEBENDO_UPLOAD`.<br>Visão com `documentosSei`. |
| `app` | `TarefaService.AnexarDocumentosSEI` e bindings regenerados. |
| Interface | `ArquivosPeca.vue`: "Documento do SEI" com os documentos ainda não escolhidos; os de tipo sugerido igual ao da peça aparecem primeiro, e a escolha é sempre do usuário.<br>`Montagem.vue`: aviso de que os não escolhidos não entram nos autos nem vão à IA; contagem de escolhidos e não escolhidos na conferência. |
| Catálogo e governança | TRV-ING-008 (nova), matriz, CAD-SEI-01 no plano de testes, V4 §48.4 revisada, CHANGELOG, D-11 E2 revisado, `V4.lock`. |

- **Sem mudança:**
  - a guarda (`internal/sei/guarda.go`) e o cliente SEI, que continua gerando "Todos os documentos";
  - o simulador `seisim`;
  - o motor, as políticas e os golden.

**Títulos.** O título lido do SEI fica só no nome local do arquivo do cadastro. O título da peça enviado à IA continua sendo o declarado no cadastro (`tituloDeclarado`). Uma nuance já existente desde a Fase 2: o **carimbo impresso em cada página**, com o nome do documento na árvore, faz parte do texto da própria página. Ele segue com as evidências e passa pela pseudonimização como o resto do texto.

## 4. Verificação
| Verificação | Resultado |
|---|---|
| `TestSEIComOTPEReautenticacao` (CAD-SEI-01, pdfium real, `sei-sintetico.pdf`) | O PDF é separado em 7 documentos, com as sugestões Memorando, defesa e alegações. Três são anexados às peças. O mesmo documento numa 2ª peça → erro. Só as páginas dos escolhidos entram nos autos (3 peças), e a tarefa chega à delimitação. Nenhum segredo nas visões. |
| `TestSepararPorFaixas`, `TestLerTextoNativoSemOCR` | Verdes |
| Vitest | 31/31 (novo: SEI com sugestão primeiro, anexar e lista atualizada); `vue-tsc` e `vite build` verdes |
| `gofmt`, `go vet` (Linux e Windows), `go test ./internal/... ./cmd/bench/...` | Verdes; `-race` em workflow, pipeline, pdf, autos e analise |
| `rulescov -estrito` | 202 regras; 124/124 IDs da V4 |
| `cmd/licencas` | 45 módulos Go e 40 npm |
| Golden | Sem diff |

## 5. Teste manual pendente (usuário, SEI real)
- **Executável novo:** `C:\Users\diogo\Documents\testes-sys-executor\sys-executor.exe`.
- **Roteiro:**
  1. PADS ou SR → "Buscar no SEI";
  2. login, 2FA e processo de teste;
  3. o app gera e baixa "Todos" (como antes) e abre o cadastro com o aviso "separado em N documento(s)";
  4. em cada peça, anexe por "Documento do SEI";
  5. confira a sugestão de tipo;
  6. conclua.
- **O que conferir:**
  - o número de documentos corresponde ao da árvore;
  - os títulos e números aparecem corretos;
  - os documentos não escolhidos não aparecem nos autos;
  - o processamento segue até TAR-004.
