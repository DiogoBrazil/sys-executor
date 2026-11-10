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

## 6. Teste real de 10/11/2026 e ajuste da revisão de nomes
- **Resultado do teste A:** login, 2FA, download e separação em 17 documentos corretos; sugestões de tipo assertivas; 6 documentos escolhidos chegaram à confirmação de envio.
- **Problema:** a lista "Possíveis nomes não reconhecidos" trouxe 18 itens, dos quais 11 não eram pessoas (termos do processo, qualificação, lugares, marca de veículo, ruído de OCR).
- **Falha encontrada na análise:** um pedaço de nome conhecido (ex.: só os sobrenomes do acusado) não era ocultado e também não ia à revisão (`conhecida()` tratava o pedaço como decidido). Corrigida.
- **Ajuste aprovado pelo usuário (itens 1 a 8; o item 9, lista permanente em Configurações, ficou para depois):**
  1. pedaços de nome conhecido são ocultados como variante da mesma pessoa: trechos contínuos, pares na ordem e palavra solta com inicial maiúscula (≥ 3 letras, fora das paradas; não vale para o que o usuário digitou). Pedaço comum a duas pessoas ganha número próprio. A palavra solta não entra na prova de resíduo (instruções e schema podem tê-la);
  2. âncoras de parente na qualificação (pai, mãe, genitor(a), filiação, cônjuge, companheiro(a), esposo(a), convivente), com a pontuação que o OCR deixar; "casado(a) com", "união estável com";
  3. valor de campos da qualificação (escolaridade, estado civil, religião, profissão, naturalidade, veículo…) não forma candidato;
  4. lugares com forma de nome (municípios de RO, capitais, UFs) só quando o trecho inteiro é o lugar;
  5. novas palavras de parada (termos disciplinares, qualificação, instituições, marcas de veículo, UNISP…), sem sobrenomes comuns;
  6. ruído de OCR (sem vogal; mesma letra 3 vezes) vira ponto de corte; sem regra de tamanho (ANA DIAS continua) nem de consoantes (Schmidt continua);
  7. "X e Y" vira dois itens quando os dois lados têm duas palavras de nome; a detecção com âncora também separa;
  8. "Não é dado pessoal" libera só a frase exata.
  - Também: o candidato passou de 6 para 10 palavras (a sobra solta se perdia).
- **Não mudou:** texto da V4, revisão item a item, golden, motor. "SO CNE AOC" continua na lista (as três palavras têm vogal; regra de tamanho deixaria passar nomes curtos).
- **Verificação:** `filtros_test.go` (casos sintéticos com a forma dos 18 itens), testes Go, `-race` em pseudonimo e pipeline, `rulescov -estrito`, bench falso idêntico ao anterior (vazamento 0).

## 7. Reteste de 10/11/2026 e segunda rodada de ajustes
- **Revisão de nomes:** caiu de 18 para 2 itens (um título em maiúsculas e uma pessoa real sem âncora).
- **Ajustes aprovados (A1, A2, B1 a B4 e a tela de depoimentos):**
  - **A1:** conectores minúsculos (*e, da, de…*) não contam como "linha com minúscula" nem como "trecho com minúscula": "NECESSIDADE e PROPORCIONALIDADE DA REAÇÃO" é título.
  - **A2:** o "Ver no PDF" da revisão acha o trecho com qualquer espaçamento entre as palavras.
  - **B1:** quando o Memorando cita só o artigo, a imputação pareia com o dispositivo do cadastro do mesmo diploma e artigo, desde que haja um só candidato e uma só imputação nessa situação. O bloco "Enquadramento do cadastro sem correspondência" deixa de repetir a imputação, e as mensagens "Inciso não confirmado" e "Nenhum dispositivo…" viram uma nota: "O Memorando cita só o art. N; o inciso X é o informado no seu cadastro. Confira." Com ambiguidade, o comportamento anterior continua.
  - **B2:** o aviso "Natureza da transgressão não declarada" não aparece no formulário de delimitação: a natureza vem do dispositivo escolhido (RDPM: a do artigo; Estatuto: campo obrigatório próprio).
  - **B3:** se o acusado lido no Memorando for a mesma pessoa do cadastro, o campo vem com o posto e o nome do cadastro. A comparação ignora "Mat.", números e palavras grudadas ("DACRUZ" = "DA CRUZ").
  - **B4:** o fato aparece pela posição ("fato 1"), e não pelo ID interno ("FATO-06").
  - **Depoimentos (`Montagem.vue`):** posto e nome do depoente vêm direto. O seletor "Já cadastrada em outra parte?" só aparece quando há outra pessoa com nome para reaproveitar, e não lista "(pessoa sem nome)". Depoimento salvo antes de ter nome volta com os campos em branco.
- **Verificação:** testes novos em `pipeline/conferencia_cadastro_test.go`, `pseudonimo/filtros_test.go` e vitest (32/32); testes Go, `-race` em pseudonimo, pipeline e analise; `rulescov -estrito`; bench falso igual ao anterior; golden sem diff.

