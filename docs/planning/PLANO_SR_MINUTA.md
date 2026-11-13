# Plano — ajustes do teste da SR (12/11/2026) e minuta da SR (F6/F7)

- **Pedido do usuário (12/11/2026):** depois do teste da SR no `.exe`, ajustar quatro pontos e concluir a SR até a minuta, "igual está no PADS".
- **Base:** V4 §§21 a 27 (SR), §§31.3 e 32 (revisão e arquivos), §42 (validação); o que já existe para o PADS (§5.7 do guia).
- **Não muda:** o caminho do PADS (cálculo, redação, validação, ajustes, arquivos), o motor determinístico, as regras do PADS, os golden do PADS e as telas aprovadas (só os pontos pedidos).

## Entrega 1 — ajustes do teste (um `.exe` e um roteiro)

### A1. Ofendido: pessoa ou instituição, um ou vários
- A V4 §23.3, c, já admite "pessoa(s) ou **ente** ofendido(s)". O aviso "Só pessoas…" da Fase 1 (P2) foi além da regra e sai.
- Cada ofendido ganha a escolha **Pessoa | Instituição** (`PessoaCadastro.Instituicao`).
  - Instituição: só o nome (ex.: "Administração Pública Militar"); sem posto nem a pergunta de criança.
  - O nome da instituição **não é pseudonimizado**: não é dado pessoal. As pessoas continuam como hoje.
- Vários ofendidos e vários sindicados continuam aceitos (já eram).
- A conferência mostra "Instituição" na tabela de envolvidos.

### A2. Papel do depoente: lista de seleção
- No modal "Depoimentos", o controle segmentado de 3 botões (que quebra em duas linhas) vira um **select** com Testemunha, Declarante e Sindicado (no PADS: Testemunha, Declarante e Acusado, como hoje).
- O `data-teste` é mantido. Não é mudança grande de tela: sem protótipo.

### A3. "Trechos citados" não abre o PDF
- **Confirmação:** sim, o comportamento previsto (UI-SRC-01, desde a fase 2) é abrir o visualizador na página do trecho, com o texto realçado.
- **Diagnóstico até aqui:** a ligação do clique está correta; na galeria, num navegador headless, o clique abre o visualizador. A falha só aparece no `.exe` (WebView2).
- **Correção:**
  - o carregamento do visualizador ganha tratamento de falha visível ("Não foi possível abrir o visualizador", com o motivo no modo diagnóstico), em vez de não mostrar nada;
  - a causa é corrigida a partir do teste pedido ao usuário ("Consultar os autos" abre ou não).

### A4. SR nunca é "inconclusiva" (mudança de regra jurídica: Emenda 3 à V4)
- **Regra do usuário:** ou há indícios suficientes ou não há. Dúvida, falta de informação ou lacuna levam a **ausência de indícios**, nunca a "inconclusivo".
- **No código:**
  - a conclusão da SR passa a ter só "indícios suficientes" e "ausência de indícios" (`domain/sr`, schema da análise, pergunta);
  - lacunas e diligências continuam aparecendo ao usuário (V4 §26.1), e a minuta registra as limitações, sem afirmar que o fato não ocorreu.
- **Governança (V4 §3 do guia):**
  - texto da Emenda 3 abaixo, **aprovado pelo usuário antes**;
  - entrada no `policies/CHANGELOG.md` e `rulescov -atualizar-lock`;
  - regras SR-SOL-004 e SR-LAC-001 (v2) no catálogo;
  - `policies/sr/analise.md` e o **golden `sr-analise.txt`** mudam (só eles; os do PADS não mudam). **Exige o aval do usuário**, porque os golden não mudam sem decisão dele.

#### Texto proposto — Emenda 3 (nova seção 49 e notas [Emenda §49])
> **Emenda 3 (12/11/2026) — seção 49:** por decisão expressa do usuário, a conclusão da Sindicância Regular, para cada fato e sindicado, é **somente** "existência de indícios suficientes" ou "ausência de indícios suficientes". **Não há conclusão inconclusiva.** Quando faltarem elementos essenciais, persistirem dúvidas ou houver limitações graves da prova, a conclusão é de **ausência de indícios suficientes**, com as lacunas e o que permanece sem esclarecimento explicitados na análise, sem afirmar que o fato não ocorreu (§26.1, parte final). As diligências sugeridas continuam a ser apresentadas ao usuário, que decide se prossegue. A seção 49 prevalece sobre as menções a "inconclusivo" nas seções 21.1, 25.1, 25.2 (item 3), 25.5 (S3), 26.1, 27 (SR13 e SR33) e 27.2 (item 5), sinalizadas com **[Emenda §49]**, mantido o texto original para histórico. Nenhuma outra regra da SR ou do PADS foi alterada.

## Entrega 2 — minuta da SR (F6 e F7 da SR)

### Fluxo
- Análise → confirmações → fontes (com a **proposta de enquadramento** das conclusões com indícios e a confirmação do usuário) → redação → validação → revisão → DOCX/PDF → finalizar. A SR **não tem cálculo** (V4 §21.1).

### Biblioteca (decisão do usuário de 12/11/2026, feita)
- A biblioteca passa a ler normas em **Markdown** (`internal/legal/markdown.go`).
- O `RDPM-RO.pdf` foi substituído pelo `RDPM RO - Decreto_13255_2007.md`, que tem o mesmo texto: foi conferido palavra por palavra, e os 100 itens do catálogo também foram conferidos.
- Entraram no acervo, com as siglas: **CP**, **CPM**, CPP, CPPM, CTB, Estatuto da Advocacia, ECA, Lei Maria da Penha, Lei de Abuso de Autoridade, Lei dos Juizados Especiais, Estatuto do Desarmamento e Lei de Tortura.
- Os originais em Markdown não têm página: a passagem cita o dispositivo. As rubricas ("Peculato") entram na busca, que pode ser restrita a alguns documentos (`BuscarEm`).

### Enquadramento (V4 §25.3; regra do usuário de 12/11/2026)
- O enquadramento existe só na conclusão **"indícios suficientes"**, por fato e sindicado. Pode haver mais de um.
- **Proposto pelo aplicativo, conferido pelo usuário:** a IA propõe, o Go confere cada dispositivo na biblioteca e o usuário confirma, troca, tira ou acrescenta. Nenhum artigo entra sem estar na biblioteca (V4 §25.3: "não inventar artigo").
- **As três categorias e onde cada uma pode estar:**
  - **transgressão disciplinar:** um inciso dos arts. 15, 16 ou 17 do RDPM, ou um inciso dos arts. 29 ou 32 do Estatuto. Vem do catálogo de enquadramento já usado no PADS, com o texto da biblioteca;
  - **crime militar:** um tipo do **CPM** ou de **outra lei penal**. Na lei penal comum, só nas hipóteses do **art. 9º, inciso II, do CPM**, e a alínea do art. 9º, II, é indicada e citada ("c/c art. 9º, inciso II, alínea e, do CPM");
  - **crime comum:** um tipo do **CP** ou de outra lei penal (não do CPM).
- **Dúvida entre normas:** a IA pode trazer alternativas, e o usuário escolhe antes de a minuta ser redigida (V4 §25.3).
- **Data do fato:** a biblioteca traz a redação vigente hoje. Por isso, "O que conferir" pede a conferência da redação na data do fato (V4 §25.3, "Data e condição funcional").
- **Na tela:** uma pergunta por conclusão com indícios, no estilo das perguntas compactas. Ela mostra cada dispositivo proposto com o texto da biblioteca e o fundamento curto. Para trocar ou acrescentar:
  - transgressão: o seletor do catálogo do PADS;
  - crime: artigo e diploma, conferidos na biblioteca na hora.
- **Prompt novo:** etapa `enquadramento`, com o prompt em `policies/sr/enquadramento.md` e os golden novos `sr-enquadramento.txt` e `pads-enquadramento.txt` (no PADS a etapa não é usada). Os golden existentes não mudam. As regras SR-SOL-005 e SR-SOL-007 passaram a valer também nessa etapa.

### Estrutura (V4 §23.1): o aplicativo compõe, a IA redige só o que é texto de análise
- Cabeçalho (logotipo e as duas linhas) e título "RELATÓRIO DE SINDICÂNCIA REGULAR".
- **I – DADOS DA SINDICÂNCIA**, composto pelo aplicativo a partir do cadastro e da leitura das peças:
  - a. Portaria: a identificação informada;
  - b. Sindicados: posto/graduação e nome (SEI do termo, se houver);
  - c. Ofendido: pessoas e instituições; com "não há ofendido individualizado", o rótulo fica vazio;
  - d. Testemunhas ouvidas: depoimentos com papel "testemunha" (SEI, se houver);
  - e. Perícias realizadas: laudos encontrados na leitura (número, se houver); sem laudo, só o rótulo;
  - f. Outras informações: termos de declarações (declarantes); os demais documentos ficam para a seção III (V4 §23.3, f, seletiva).
  - Dado incerto → só o rótulo, com aviso em "O que conferir" (SR02, SR03).
- **II – DA SÍNTESE DOS FATOS EM APURAÇÃO:** IA, a partir da Portaria e dos documentos de origem, sem antecipar conclusão (V4 §23.5).
- **III – ANÁLISE DAS PROVAS:** IA, **um trecho por fato**, examinando cada sindicado dentro do fato, preso às conclusões confirmadas pelo usuário (V4 §§24.1 e 24.2); cada parágrafo com evidências.
- **IV – SOLUÇÃO:** composta pelo **aplicativo** a partir das conclusões e enquadramentos confirmados (modelos S1 a S4 da V4 §25.5):
  - conclusão conjunta quando todos os sindicados de um fato têm o mesmo resultado; individual quando divergem; por fato quando há vários;
  - só militares sindicados como sujeitos;
  - **sem providências por padrão** (V4 §25.4); o usuário pode pedi-las em "Solicitar ajustes".
- Fecho: local e data `[A PREENCHER]`; encarregado `[A PREENCHER]`; espaço de assinatura sem assinatura.

### Validação, revisão e arquivos (como no PADS)
- Verificação de suporte, trecho a trecho, e reescrita até 2 vezes.
- Validadores determinísticos da SR (`minuta.ValidarSR`):
  - as quatro seções e as seis alíneas, na ordem;
  - cada par fato × sindicado com o resultado confirmado;
  - indícios com enquadramento; ausência sem enquadramento;
  - nenhum civil nem militar não autorizado como sujeito da IV;
  - nenhuma providência que não tenha sido pedida;
  - proibições de forma (página, mídia, marcador, identificador interno).
- "Solicitar ajustes", "Alterar", DOCX/PDF e "Aprovar e finalizar": os mesmos do PADS, com o título e o nome de arquivo da SR.

### Implementação (generalizar sem mudar o caminho do PADS)
- `internal/minuta/sr.go` (`ComporSR`) e `ValidarSR` em `validar.go`.
- `internal/analise/redacao.go`: trechos da SR (síntese; análise por fato).
- `internal/pipeline/redigir.go`, `validar.go` e o ajuste: o redator passa a atender os dois tipos; o PADS segue igual.
- `internal/pipeline/fontes.go`: na SR, o texto dos dispositivos de transgressão indiciados.
- Interface: o enquadramento na pergunta da conclusão; a minuta usa o mesmo `MinutaPrevia.vue`.
- Testes: um fluxo completo da SR com IA simulada (como `TestFluxoPADSComIA`); `ValidarSR` com adulterações; DOCX/PDF da SR; testes de tela dos ajustes A1 a A3.

### Modelos de referência
- Os relatórios de `docs/draft_templates/SR` servem de **forma e estilo** (V4 §27.1). São tratados como dados reais: nada deles vai para o código, fixtures, chat ou IA do app. **Só são lidos com o aval do usuário.**

## Verificação e entrega
- Verificação local do §4.4 (sem bench) em cada entrega, `.exe` na pasta de teste e roteiro curto.
- Ao fim: guia, `RELATORIO_FASE6.md`/`RELATORIO_FASE7.md` (seção SR) e este plano atualizados. Commit só quando o usuário pedir.

## Situação (12/11/2026, noite)
- **Entregas 1 e 2 implementadas e verificadas localmente:**
  - `go test`, `go vet` (Linux e Windows) e `gofmt` sem pendências;
  - vitest 44/44 e `vue-tsc`;
  - `rulescov -estrito` e `bibgen -conferir` sem falhas;
  - DOCX e PDF da SR gerados de verdade.
- **`.exe` de 19:20** na pasta de teste. Aguarda o teste do usuário e, depois, o commit.
- **Diferenças em relação ao que estava planejado:**
  - o enquadramento dos crimes vem da biblioteca, e não é digitado livremente, porque o usuário incluiu o CP e o CPM em Markdown;
  - a validação da SR fica em `validar_sr.go`, e não no `validar.go`;
  - a redação da SR fica em `analise/redacao_sr.go` e `pipeline/redigir_sr.go`, e o PADS ficou igual.
- **Não feito:**
  - o tratamento de falha visível ao abrir o visualizador (A3): a causa foi achada e corrigida no CSS, então ele deixou de ser necessário;
  - a tela de enquadramento na galeria.
