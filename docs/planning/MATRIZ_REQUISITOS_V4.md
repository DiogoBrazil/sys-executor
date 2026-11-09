# MATRIZ DE REQUISITOS V4 → REGRA → MÓDULO → TESTE

> Base: `docs/rules/sys-executor_especificacao_consolidada_V4.md` (lida integralmente em 08/11/2026) + decisões D-01…D-07 (`REGISTRO_DUVIDAS_DECISOES.md`).
> **Rótulos:** AP = aprovado pelo usuário · IN = interpretação registrada · PT = proposta técnica · AV = a validar em testes.
> **Tipos:** DET determinística · IA política de IA · VAL validador · PRG pergunta · TXT texto fixo · UI · SEG segurança.
> **Status inicial de todas as linhas:** `Planejado`. Os IDs de teste estão em `PLANO_TESTES_ACEITACAO.md`.
> Esta matriz semeou o catálogo `internal/rules/catalog/*.yaml` (Fase 3). Desde então o catálogo é a fonte das regras; a coluna "Módulo / política" é o planejamento, e o campo `enforced_by` do catálogo guarda os símbolos Go reais. O `cmd/rulescov` exige o mesmo conjunto de regras nos dois e prova que todos os IDs de teste da V4 aparecem no catálogo.

## 1. Escopo e natureza do produto

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-ESC-001 | §§1, 21.2, 28.3 | Produzir apenas **minutas** de apoio; nunca decidir, assinar, publicar, homologar, punir, alterar ficha | AP | IA+VAL | nucleo; validate.NoFakeActs | SEG13-07, T09 |
| TRV-ESC-002 | §§1.2, 2.1, 13 | Nenhuma operação destrutiva/de escrita no SEI (bloqueio arquitetural) | AP | SEG | sei (sem métodos de escrita + allowlist `acao`) | SEG13-07, SEG-SEI-01 |
| TRV-ESC-003 | §§12.1-1/2, 16.6 | Não calcular prescrição nem auditar nulidades; relatar argumentos da defesa presentes nos autos | AP | IA+VAL | nucleo; pads/fundamentacao | A16, A17 |
| TRV-ESC-004 | §§21.1, 27.4 | Não confundir PADS com SR (acusado×sindicado; julgamento×relatório); sem homologação/avocação | AP | IA+VAL | domain/pads × domain/sr; validate.ScopeSR | SR34, T25 |
| TRV-ESC-005 | §§26.3, 21.2 | Aviso de revisão humana integral ao disponibilizar a minuta; não bloqueia download; não equivale a finalizar | AP | UI+TXT | frontend Revisao; compose.Aviso | SR34, T09, A31 |

## 2. PADS — escopo (Memorando Disciplinar)

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| PADS-ESC-001 | §§22.1, 30.2 | Memorando Disciplinar inicial é a peça delimitadora; imputações extraídas, apresentadas, confirmadas e **congeladas** (versão auditável) | AP | DET+PRG | workflow.Delimitar; domain/pads.ImputacaoSet | PADS-E01…E05 |
| PADS-ESC-002 | §22.1 | Nunca criar, ampliar, substituir ou trocar artigo de imputação para cobrir conduta diversa | AP | IA+VAL | schema enum imputacao_id; validate.ScopePADS | PADS-E05, VAL-NEG-01 |
| PADS-ESC-003 | §22.2 | Resultados admitidos: procedência, procedência parcial, improcedência, sempre por imputação original | AP | DET+VAL | engine.ResultadoGlobal | PADS-E02, PADS-E04 |
| PADS-ESC-004 | §§22.1, 22.3 | Conduta diversa descoberta na instrução não gera acusação, punição nem encaminhamento automático na minuta | AP | IA+VAL | I-05; validate.ScopePADS | PADS-E01, PADS-E03, PADS-E05 |
| PADS-ESC-005 | §22.1 | Classificação materialmente ambígua de imputação → perguntar, sem inovar | AP | PRG | workflow.Pergunta | PADS-AMB-01 |
| PADS-ESC-006 | §§21.1, 22 | Um acusado por PADS | AP | DET | domain/pads | PADS-1AC-01 |
| PADS-ESC-007 | §24.3 (diferença crítica) | A flexibilidade de fatos novos da SR **não** existe no PADS, nem por autorização informal | AP | VAL | validate.ScopePADS | PADS-E05 |

## 3. PADS — estrutura e redação

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| PADS-EST-001 | §3 | Estrutura RELATÓRIO / FUNDAMENTAÇÃO / DECISÃO (numeração compatível com modelos, D-04) | AP | TXT | compose.PADS | EXP-PADS-01 |
| PADS-RED-001 | §3.1 | Relatório: procedimento, acusado, imputações, acusação, defesa prévia, provas, alegações finais, cronologia, sem antecipar culpa | AP | IA | pads/relatorio | INT-PADS-01 |
| PADS-RED-002 | §3.2 | Fundamentação por imputação: elementos, autoria, materialidade, provas favoráveis/desfavoráveis, defesa | AP | IA+VAL | pads/fundamentacao; validate.ScopePADS | INT-PADS-02 |
| PADS-RED-003 | §§3.2, 12.1-4, 16.5 | Prova insuficiente/contraditória → fundamentar; *in dubio pro reo* quando cabível | AP | IA | pads/fundamentacao | A07, P1 |
| PADS-RED-004 | §3.3 | Decisão por imputação; sanção com natureza, dias se cabíveis e fundamentos | AP | TXT+IA | compose.PADS.Decisao | INT-PADS-03 |
| PADS-RED-005 | §3.4 | Providências: improcedência (ciência, publicação BI, arquivamento); procedência (ciência, publicação, recursos, **nota de punição só após irrecorribilidade**, arquivamento) | AP | TXT | compose.Providencias | C13-PROV-01/02 |
| PADS-RED-006 | §3.4 ponto pendente | Proposta de regressão sem afirmar mudança administrativa já efetivada | AP | TXT+VAL | compose; validate.NoFakeActs | EX1, A30 |
| PADS-RED-007 | §10 | Cláusula residencial exata, parametrizada (posto/graduação, matrícula, nome; flexões), **somente** com detenção/prisão; faltante → `[A PREENCHER]` + pergunta | AP | TXT+VAL | compose.ClausulaResidencial; validate.CalcConsistency | C13-CLA-01/02/03 |
| PADS-RED-008 | §15.4 | Justificativa redacional da regressão pelo conjunto, adaptada (BOM/20, INS/9, ÓT/30), só com gatilho satisfeito | AP | TXT | compose.JustRegressao | MP-01…04 |
| PADS-RED-009 | §§3.2, 17.2 | Não afirmar ter visto/ouvido mídia; descrições do usuário usadas naturalmente, sem etiqueta obrigatória | AP | IA+VAL | nucleo; validate.SupportCheck | A11, V1 |
| PADS-RED-010 | D-01 | Não citar o art. 41 § 3º do RDPM como vigente; sem alerta na UI | AP | IA+VAL | legal.NotaVigencia; validate.LegalRefs | LIB-D01-01/02 |

## 4. PADS — conexão e dosimetria

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| PADS-CNX-001 | §§5.1, 8.7; RDPM 58 VI §1º | Conexas: escolher a principal (maior gravidade demonstrada, confirmada); secundárias como agravante; sem punição autônoma nem dupla contagem | AP/IN(I-04) | DET+PRG | engine.ConexaoPrincipal | C13-CNX-01 |
| PADS-CNX-002 | §5.2 | Não conexas: punição individual por transgressão procedente, com análise própria | AP | DET | engine.Sancoes | C13-IND-01, MP-* |
| PADS-CNX-003 | §5.3 | Explicar conexão com base nos autos; perguntar se necessário; não repetir perguntas | AP | PRG | workflow | PRG-01 |
| PADS-DOS-001 | §6.1; RDPM 36 | Havendo punição, examinar os sete fatores I–VII individualmente, indicando ausência de elementos quando for o caso | AP | VAL+IA | validate.RequiredParts | DOS-36-01 |
| PADS-DOS-002 | §6.2; RDPM 37 | Causas de justificação examinadas com fundamentação | AP | IA | pads/circunstancias | DOS-37-01 |
| PADS-DOS-003 | §6.2; RDPM 38, 39 | Atenuantes e agravantes com fundamentação específica, sem presunção | AP | IA+PRG | pads/circunstancias | DOS-38-01 |
| PADS-DOS-004 | §§6.2, 9, 15.6 | Comportamento **anterior**: ÓT/EXC → atenuante 38 I; INS → agravante 39 I; BOM → nenhuma | AP | DET | engine.CircunstanciaComportamento | DOS-CMP-01/02/03 |
| PADS-DOS-005 | §§6.2, 9-5 | Perguntar sobre relevantes serviços (sim/não + justificativa factual); não inventar | AP | PRG | workflow | DOS-38II-01 |
| PADS-DOS-006 | §6.3 | Dias: nenhuma circunstância ou quantidades iguais = 2; mais agravantes = 3; mais atenuantes = 1; sem multiplicar a diferença; repreensão sem dias | AP | DET | engine.Dosimetria | C13-DIA-01/02/03, R9 |
| PADS-DOS-007 | D-05 | Cada alínea reconhecida do art. 39 VII conta como uma agravante; incisos II e III no máximo uma vez cada | AP | DET | engine.Dosimetria | DOS-D05-01…04 |
| PADS-DOS-008 | §6.3; RDPM 58 II b | Validar compatibilidade com art. 58 III e com o limite de detenção ≤ 10 dias | AP/IN(I-03) | VAL | validate.CalcConsistency | DOS-LIM-01 |
| PADS-DOS-009 | §6.3 | Memória da dosimetria visível e revisável pelo usuário | AP | UI | frontend MemoriaCalculo | UI-CALC-01 |

## 5. PADS — reincidência

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| PADS-REI-001 | §§7.1, 15.5-1 | Somente mesma natureza (L/L, M/M, G/G); a equivalência de comportamento não muda a natureza | AP | DET | engine.Reincidencia | R4, A21, C13-REI-01 |
| PADS-REI-002 | §§7.1, 15.5-2 | Usar a data de **término do cumprimento** da punição anterior × data do **fato** atual | AP | DET | engine.Reincidencia | R1, C13-REI-02 |
| PADS-REI-003 | §15.5-3 | Término estritamente anterior ao fato; posterior impede | AP | DET | engine.Reincidencia | R2, A22 |
| PADS-REI-004 | §§7.1, 15.5-4; RDPM 39 p.ú. | Prazo de 1/2/4 anos; desconsidera somente se decorrido tempo **superior** | AP | DET | engine.DentroPrazoReincidencia | R1, REI-PRZ-01…03 |
| PADS-REI-005 | D-07 | Exatamente 1/2/4 anos ainda conta; a partir do dia seguinte ao aniversário, não | AP | DET | engine.DentroPrazoReincidencia | REI-D07-01…04 |
| PADS-REI-006 | §15.5-5 | Mesma data: reconhecer só com horários provando anterioridade; iguais ou ausentes → não reconhecer | AP | DET+PRG | rules/civil.Comparar | R5, R6, R7, R8, A23, A24 |
| PADS-REI-007 | §15.5 | Data de término desconhecida → perguntar só se necessário; não fornecida → não reconhecer, com registro explicativo | AP | PRG+DET | workflow; engine | R3, A26 |
| PADS-REI-008 | §§7.2, 15.5 | Várias punições anteriores válidas → **uma** agravante | AP | DET | engine.Reincidencia | R9, A25 |
| PADS-REI-009 | §7.2 | Perguntas condicionais: só após fixar a natureza atual; "possui punição anterior de natureza X?" → se sim, data de término DD/MM/AAAA; aproveitar documentos | AP | PRG | workflow.PerguntasReincidencia | PRG-REI-01/02 |
| PADS-REI-010 | §7.2 | Conexas: natureza da principal; não conexas: individual, reaproveitando dados sem repetir perguntas | AP | DET+PRG | engine; workflow | REI-CNX-01 |
| PADS-REI-011 | §15.6 | Não confundir o prazo da reincidência com o "inferior a um ano" da regressão | AP | DET | funções distintas | REI-D07-04, DT-02 |

## 6. PADS — pontuação, comportamento, equivalências, regressão

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| PADS-CMP-001 | §8.1; RDPM 70 | Faixas: EXC > 40; ÓT 30–40; BOM 10–29; INS < 10 | AP | DET | engine.Comportamento | CMP-FX-01…08 |
| PADS-CMP-002 | §8.1 | Pontos e classificação atuais perguntados/confirmados; divergência → alertar e pedir esclarecimento | AP | PRG | workflow | CMP-DIV-01 |
| PADS-CMP-003 | §§8.1, 11, 13 | Dados ausentes → `[A PREENCHER]` e indicação de cálculo não concluído; nunca inventar | AP | TXT+VAL | compose; validate | SEG13-05, CMP-AUS-01 |
| PADS-CMP-004 | §8.2; RDPM 71 | Desconto ordinário: repreensão −1, detenção −2, prisão −4; não presumir acréscimos (+2/ano) | AP | DET | engine.Pontuacao | C13-PTS-01 |
| PADS-CMP-005 | §§8.2, 15.4 | Sem regressão: somar os descontos de todas as sanções aplicadas, sem duplicação (conexas: só a principal) | AP | DET | engine.Pontuacao | MP-05, MP-06 |
| PADS-CMP-006 | §§8.3, 8.8, 15.4 | Regressão no conjunto do mesmo PADS → valor fixo 9/20/30, **sem nenhum desconto ordinário** do PADS | AP | DET+VAL | engine.Pontuacao; validate.CalcConsistency | MP-01…04, A18, A19, A20, C13-REG-01 |
| PADS-CMP-007 | §8.3; RDPM 70 §2º | INSUFICIENTE: desconto ordinário, sem reposição a 9 | AP | DET | engine.Pontuacao | C13-INS-01 |
| PADS-EQV-001 | §§8.4, 15.1 | 2L = 1M; 2M = 1G; 4L = 1G; mistas permitidas; resto não arredondado (cálculo em unidades) | AP/IN(I-01) | DET | engine.Equivalencia | C13-EQV-01, E2, E3, EQV-01…08 |
| PADS-EQV-002 | §15.1 | A mesma infração nunca entra duas vezes em equivalência | AP | DET | engine.Equivalencia | EQV-DUP-01 |
| PADS-EQV-003 | §§8.5, 8.7, 15.1 | Considerar só o histórico **reconhecido em decisão disciplinar** + atuais procedentes; nunca acusações pendentes | AP | DET+PRG | engine; workflow | EQV-PEND-01 |
| PADS-CMP-008 | §§8.5, 15.1, 15.2 | BOM/ÓTIMO: 2 graves equivalentes com datas de cometimento em janela **estritamente inferior a 1 ano** (aniversário exclusivo) | AP | DET | engine.GatilhoBomOtimo; civil.MenosDeUmAno | E1, E2, C13-4M-01, DT-01…05, A27, A28 |
| PADS-CMP-009 | D-06 | A janela precisa conter ≥1 transgressão procedente do PADS atual; histórico sozinho fechando gatilho → alerta + pergunta | AP | DET+PRG | engine.GatilhoBomOtimo | CMP-D06-01…03 |
| PADS-CMP-010 | §15.2 | Demonstrar o subconjunto e a janela usados; não somar fatos dispersos | AP | DET+UI | engine.MemoriaCalculo | DT-05 |
| PADS-CMP-011 | §§12.2-8, 15.2 | 29/02 em ano não bissexto: não impor critério; perguntar quando material | AP | DET+PRG | civil.ErrCasoLimite29Fev | DT-06 |
| PADS-CMP-012 | §§8.6, 15.3 | EXCEPCIONAL: 3ª grave equivalente no período atual, **sem janela**; propor ÓTIMO/30 já na minuta | AP/IN(I-02) | DET | engine.GatilhoExcepcional | EX1, A30, C13-EXC-01 |
| PADS-CMP-013 | §§8.6, 15.3 | Não reutilizar infrações consumidas em regressão excepcional→ótimo anterior; data de ingresso não obrigatória; o usuário confirma o pertencimento ao período | AP | DET+PRG | engine; workflow | EX2, A29, C13-EXC-02 |
| PADS-CMP-014 | §§8.6, 15.3 | Não declarar mudança administrativa efetivada; não alterar ficha/SEI | AP | TXT+VAL | compose; validate.NoFakeActs | A30 |
| PADS-CMP-015 | §8.8 | Memória de cálculo com os 7 itens, visível antes da aprovação | AP | UI+DET | engine.MemoriaCalculo; frontend | UI-CALC-01/02 |
| PADS-CMP-016 | §15.4 limite | Regra de valor fixo restrita ao mesmo PADS; processos distintos não são unificados | AP | DET+PRG | engine | CMP-LIM-01 (P-05) |
| PADS-CMP-017 | P-04 | Saldo negativo: calcular, sinalizar e perguntar se material | IN | DET+PRG | engine | CMP-NEG-01 |
| PADS-PRG-001 | §9 | Ordem de perguntas: imputações → procedência/conexão → natureza → comportamento/pontos → serviços → reincidência → regressão bom/ótimo → excepcional → cálculo → minuta | AP | PRG | workflow.OrdemPerguntas | PRG-ORD-01 |

## 7. Sindicância Regular

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| SR-EST-001 | §23.1 | Estrutura fixa I–IV e alíneas a–f, em ordem, sempre presentes | AP | TXT+VAL | compose.SR; validate.ScopeSR | SR02, SR03, EXP-SR-01 |
| SR-EST-002 | §23.2 | Preencher só com certeza; incerto ou divergente → rótulo vazio + aviso na UI | AP | DET+VAL | compose.SR | SR02, SR03 |
| SR-EST-003 | §§23.2, 23.3 | SEI apenas quando comprovado; nunca fictício | AP | VAL | validate.SEIIds | SR04 |
| SR-EST-004 | §23.3-a | Portaria: número, identificação, data; não confundir com sindicância apensada | AP | IA | sr/dados | SR01 |
| SR-EST-005 | §23.3-b | Sindicados: formais + autorizados; vários | AP | DET | domain/sr | SR01 |
| SR-EST-006 | §23.3-c | Ofendido(s) identificados com segurança; não atribuir a todo narrador | AP | IA+VAL | sr/dados | SR02, SR06 |
| SR-EST-007 | §23.3-d | Testemunhas **efetivamente ouvidas como testemunhas** (Termo de Inquirição) | AP | IA | sr/dados | SR04, SR05 |
| SR-EST-008 | §23.3-e | Perícias realizadas, com nº e SEI se houver; teor discutido em III | AP | IA | sr/dados | SR03 |
| SR-EST-009 | §23.3-f | Outras informações: seleção relevante e sucinta, sem inventário nem repetição | AP | IA | sr/dados | SR05, SR07 |
| SR-EST-010 | §§23.2, 27.2-9 | Papel da pessoa **por ato e contexto**; sem duplicação | AP | DET | domain/common.PapeisPorAto | SR06 |
| SR-RED-001 | §23.5 | II – Síntese a partir da portaria/notícia; linguagem preliminar; sem antecipar indício/culpa; fatos novos reservados à III | AP | IA | sr/sintese | SR08 |
| SR-PRV-001 | §24.1 | III organizada por fato e, dentro dele, por pessoa; individualizar condutas; resultados distintos possíveis | AP | IA+VAL | sr/analise | SR09, SR12 |
| SR-PRV-002 | §24.2 | Depoimentos: síntese fiel, transcrição pontual, atribuição precisa, omissão não é ocultação | AP | IA+VAL | sr/analise; SupportCheck | SR32 |
| SR-PRV-003 | §24.2 | Contradições: avaliar relevância; secundária não vira indício | AP | IA | sr/analise | SR14 |
| SR-PRV-004 | §24.2 | Perícia: laudo negativo não prova inexistência; reproduzir limitações | AP | IA | sr/analise | SR15 |
| SR-PRV-005 | §24.2 | Documentos de outros procedimentos incorporados: avaliar pertinência; não importar conclusões | AP | IA | sr/analise | SR16 |
| SR-PRV-006 | §24.2 | Mídia ausente: pedir descrição/transcrição; não presumir | AP | PRG+IA | workflow | SR17 |
| SR-ESC-001 | §24.3 | Fato novo fora da portaria: identificar, mostrar elementos, **perguntar**; incluir em III/IV só se autorizado | AP | PRG+DET+VAL | domain/sr.Autorizacao; validate.ScopeSR | SR19, SR35, S5 |
| SR-ESC-002 | §24.3 | Novo militar (inclusive de outra OPM): perguntar antes de incluir como sindicado; sem alterar portaria | AP | PRG+DET+VAL | domain/sr.Autorizacao | SR20, SR35 |
| SR-ESC-003 | §24.4 | Civis analisados em III; **nunca** sujeitos da IV nem enquadrados no RDPM | AP | DET+VAL | schema enum sujeitos; validate.ScopeSR | SR21 |
| SR-ESC-004 | §24.5 | Situação funcional de cada militar **na data de cada fato**; perguntar só o essencial | AP | DET+PRG+IA | domain/common.SituacaoFuncional | SR22, SR23 |
| SR-RED-002 | §24.6 | Proporcionalidade; nunca omitir prova relevante para encurtar; sem extensão artificial | AP | IA | sr/analise | SR30, SR31 |
| SR-SOL-001 | §25.1 | IV objetiva e fundamentada; sem repetir a prova; sem sanção | AP | IA+VAL | sr/solucao | SR27, SR28 |
| SR-SOL-002 | §25.1 | Conclusão conjunta só se cada conduta foi analisada e o resultado é igual | AP | VAL | validate.ScopeSR | SR10, SR11, SR36, S1 |
| SR-SOL-003 | §25.1 | Conclusão por fato × sindicado | AP | DET | domain/sr.ConclusaoSR | SR12, S2, S3 |
| SR-SOL-004 | §25.2 | Indícios suficientes / ausência / inconclusivo; distinguir ocorrência de autoria | AP | IA+DET | sr/solucao | SR13, SR18, S3, S4 |
| SR-SOL-005 | §25.3 | Enquadramento específico (crime comum, crime militar, transgressão), com diploma e artigo **confirmados** por fonte | AP | IA+VAL | validate.LegalRefs | SR25, SR26 |
| SR-SOL-006 | §25.3 | Dúvida entre normas → alternativas ao usuário antes de concluir | AP | PRG | workflow | SR26 |
| SR-SOL-007 | §25.3 | Não preso à capitulação da portaria para os mesmos fatos | AP | IA | sr/solucao | SR24 |
| SR-SOL-008 | §25.4 | Providências só com necessidade específica; nunca "instaurar PADS" ou "arquivar" automáticos | AP | IA+VAL | validate (lista de fórmulas automáticas) | SR27, SR28, SR29 |
| SR-SOL-009 | §25.5 | Exemplos S1–S5 são ilustrativos; nunca preencher marcadores fictícios | AP | VAL | validate.NoPlaceholderLeak | S1–S5 |
| SR-LAC-001 | §26.1 | Lacuna impeditiva → alertar + diligências concretas; o usuário decide prosseguir; conclusão inconclusiva qualificada | AP | PRG+IA | workflow | SR13, SR33 |
| SR-NAT-001 | §21.1 | SR sem punição, dosimetria, reincidência, comportamento ou pontos | AP | VAL | validate.ScopeSR | SR-NAT-01 |

## 8. Transversal — prova, ingestão, rastreabilidade

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-ING-001 | §§2.2, 16.1 | Classificar peças pela função documental (conteúdo + cronologia), não pelo título | AP | IA+DET | ingest/segment | A01, A02, D1, D2 |
| TRV-ING-002 | §16.1 | Classificação incerta e material → perguntar | AP | PRG | workflow | A01-AMB |
| TRV-ING-003 | §§16.3, 12.1-6 | OCR quando necessário; baixa confiança sinalizada; peça essencial ilegível → pedir versão legível | AP | DET+PRG | ingest/ocr | A04, T06 |
| TRV-ING-004 | §39 | Autos extensos em etapas; texto integral preservado por página e peça; resumos só auxiliares | AP | DET | ingest/evidence; orchestrator | T17 |
| TRV-ING-005 | §39 | Cobertura de páginas e peças; nada é ignorado silenciosamente | AP | DET+UI | ingest/evidence.Cobertura | T17, ING-COB-01 |
| TRV-ING-006 | §39 | Busca vetorial temporária dos autos opcional; não obrigatória | AP/AV | PT | ingest/evidence | BEN-RAG-01 |
| TRV-PRV-001 | §§2.2, 16.4 | Relação afirmação → peça, SEI, página, trecho; abrir página e realçar | AP | DET+UI | EvidenceSpan; PDF.js | A05, UI-SRC-01 |
| TRV-PRV-002 | §16.4 | Na minuta: citação simples com SEI se houver; sem SEI → "incluído nos autos"; sem página | AP | IA+VAL | validate.SEIIds | A05, A06 |
| TRV-PRV-003 | §16.2 | Datas divergentes: preservar versões e fontes; resolver fundamentadamente ou perguntar; nunca escolher arbitrariamente | AP | DET+PRG | ingest/Divergencia | A03 |
| TRV-PRV-004 | §§17.1–17.3 | Não baixar/transcrever mídia externa; pedir descrição; sem descrição → ressalva e prova restante | AP | DET+PRG+IA | workflow; nucleo | A08, A09, A10, V2 |
| TRV-PRV-005 | §17.4 | Complementos manuais (PDF, DOCX, imagem, texto); distinguir a origem operacional | AP | DET | ingest | A12 |
| TRV-PRV-006 | §§1.2, 13 | Documentos são dados não confiáveis (prompt injection) | AP | SEG+IA | nucleo; §3 da estratégia | SEG13-04, SEG-INJ-01…05 |
| TRV-PRV-007 | §1.2 | Não fabricar fato, testemunho, data, precedente, citação, prova ou número | AP | IA+VAL | todos os validadores | VAL-NEG-* |
| TRV-PRV-008 | §§16.4, 23.2 | Nenhum SEI fictício | AP | VAL | validate.SEIIds | VAL-NEG-02 |

## 9. Transversal — fluxo, interface, sessão

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-UI-001 | §30.1 | 1º escolher PADS/SR; 2º SEI/upload; pipeline compartilhado | AP | UI | frontend Inicio/Entrada | T01 |
| TRV-UI-002 | §30.1-4 | Upload não pede credencial nem abre SEI | AP | UI+SEG | workflow | T02 |
| TRV-UI-003 | §§30.3, 35 | SEI sem internet → aviso + upload, preservando a escolha | AP | UI | workflow | T03 |
| TRV-UI-004 | §30.2 | Compatibilidade: Memorando (PADS) / Portaria (SR) pelo conteúdo; portaria anexa não muda o PADS; nunca trocar silenciosamente | AP | IA+PRG | workflow.Compatibilidade | T04, T05, T06 |
| TRV-UI-005 | §31.1 | Sem chatbot principal; status claros; sem streaming de tokens; percentual só se mensurável | AP | UI | frontend Processamento | T07 |
| TRV-UI-006 | §§31.2, 26.2, 9 | Perguntas pontuais, só o essencial, sem repetir; retomar a mesma tarefa | AP | PRG | workflow DAG | T08, PRG-01 |
| TRV-UI-007 | §31.3 | Prévia completa (fiel ao PDF); DOCX+PDF; "Aprovar e finalizar" / "Solicitar ajustes" | AP | UI | frontend Revisao | T09 |
| TRV-UI-008 | §31.3 | Ajuste preserva partes válidas e decisões confirmadas; revalida; sem histórico permanente de chat | AP | DET+IA | workflow.Ajuste | T10 |
| TRV-UI-009 | §§1.1-8, 19-A31 | "Aguardando revisão" até comando inequívoco; downloads permitidos antes | AP | DET | workflow FSM | A31, SEG13-06 |
| TRV-SES-001 | §29.1 | Tarefa efêmera, sem histórico nem retomada | AP | DET | session; workflow | T12 |
| TRV-SES-002 | §29.1 | Finalizar → apagar todos os artefatos controlados; preservar exportações, biblioteca, modelos e preferências | AP | DET | session.Finalizar | T11 |
| TRV-SES-003 | §29.2 | Temporários mínimos, com ACL do usuário Windows e proteção adicional (cifra AES-256-GCM com chave só em memória, PT-F1-01) | AP/PT | SEG | session; platform/win | SEG-DIR-01 |
| TRV-SES-004 | §29.2 | Na inicialização, remover resíduos de crash sem restaurar | AP | SEG | session.VarrerResiduos | T12, SEG-CRASH-01 |
| TRV-SES-005 | §§29.2, 1.1-9 | Não prometer apagamento forense | AP | UI | textos | DOC-01 |
| TRV-SES-006 | §§12.2-4, 43 | Concorrência: uma tarefa ativa por instância | PT | DET | workflow | SES-CONC-01 |

## 10. Transversal — segurança

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-SEG-001 | §§2.1, 33 | Credenciais solicitadas a cada tarefa, só em memória, nunca persistidas; limpeza como melhor esforço | AP | SEG | session.Credenciais | T13, SEG-CRED-01 |
| TRV-SEG-002 | §§2.1, 13 | OTP/2FA e reautenticação legítimos; sem contorno | AP | SEG+UI | sei | SEG13-01, SEG13-02 |
| TRV-SEG-003 | §§1.2, 33 | Sem módulo de logs persistentes; diagnóstico em memória sem dados processuais | AP | SEG | app logger; cliente HTTP da IA sem log | SEG-LOG-01 |
| TRV-SEG-004 | §33 | Erros abrangentes, claros e amigáveis; sem segredos; sem simular sucesso | AP | UI | catálogo de erros | ERR-01…15 |
| TRV-SEG-005 | §§2.2, 33, 41 | Nada sigiloso em buscas públicas | AP | SEG | legal/web.Sanitizador | T20, SEG-WEB-01 |
| TRV-SEG-006 | §36.2 |  **REVOGADA (§47, D-10)** — Sem envio de autos a provedores externos de IA | AP | SEG | inference (somente loopback) | SEG-NET-01 |
| TRV-SEG-007 | PT |  **REVOGADA (§47, D-10)** — Sidecar em loopback, porta aleatória, API key, Job Object | PT | SEG | inference.Sidecar | SEG-SID-01 |
| TRV-SEG-008 | PT | Cache do WebView2/Edge e dumps mitigados | PT/AV | SEG | platform/win | SEG-CACHE-01, SEG-WER-01 |
| TRV-SEG-009 | §47.4 | Chave da Gemini API cifrada com DPAPI; nunca na interface, eventos, preferências, erros ou logs; validada sem gerar texto; substituível e removível | AP | SEG | credencial; app.IAService | SEG-KEY-01 |
| TRV-SEG-010 | §47.3 | Confirmação de envio à Gemini API em cada tarefa, antes da 1ª etapa de IA; sem autorização nada é enviado | AP | SEG+UI | pipeline (envio_ia) | SEG-ENV-01 |
| TRV-SEG-011 | §47.3 | Pseudonimização de dados pessoais antes do envio e restauração exata (melhor esforço) | AP | SEG | pseudonimo | SEG-PSEUDO-01 |
| TRV-SEG-012 | §47.3 | Envio só a generativelanguage.googleapis.com; store=false; sem ferramentas, Files API nem cache no provedor | AP | SEG | inference.MotorGemini | SEG-NET-01 |

## 11. Transversal — IA (Gemini API desde a emenda §47; regras da IA local revogadas)

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-IA-001 | §36 |  **REVOGADA (§47, D-10)** — IA local prioritária; GGUF; llama.cpp/llama-server como candidatos; sem acoplar regras ao modelo | AP | PT | inference (porta InferenceEngine) | F0-02 |
| TRV-IA-002 | §36.5 | Motor determinístico em Go para regras formalizáveis | AP | DET | rules/engine | todos os DET |
| TRV-IA-003 | §37 |  **REVOGADA (§47, D-10)** — Vários modelos instalados; um gerador por tarefa; embeddings separados | AP | DET | inference.Gerenciador | MOD-01 |
| TRV-IA-004 | §37 |  **REVOGADA (§47, D-10)** — Detecção de hardware e seleção automática entre modelos testados; seleção manual avançada | AP | DET+UI | inference.Hardware | T14, MOD-02 |
| TRV-IA-005 | §37 |  **REVOGADA (§47, D-10)** — Primeiro uso: download orientado + importação de GGUF; integridade; listar/remover | AP | DET+UI | inference.Gerenciador | T27, MOD-03/04 |
| TRV-IA-006 | §38.1 |  **REVOGADA (§47, D-10)** — GPU priorizada; GPU incompatível → CPU se viável | AP/AV | DET | inference | T14 |
| TRV-IA-007 | §38.2 |  **REVOGADA (§47, D-10)** — Q4_K_M como referência; nenhum modelo definitivo antes de teste | AP/AV | PT | catálogo de modelos | BEN-01 |
| TRV-IA-008 | §38.3 |  **REVOGADA (§47, D-10)** — Contexto adaptativo pela memória **disponível** | AP/AV | DET | inference.Orcamento | T16, BEN-02 |
| TRV-IA-009 | §38.4 |  **REVOGADA (§47, D-10)** — Máquina insuficiente → mensagem amigável, alternativa ou impossibilidade; nunca degradar em silêncio | AP | UI | inference | T15 |
| TRV-IA-010 | §§37, 31.1 | Sem streaming na UI | AP | UI | frontend | T07 |
| TRV-IA-011 | §§47.1, 47.5 | Gemini API (SDK oficial, chave do usuário); sem modelo fixo: o usuário lista, testa e escolhe; um modelo por tarefa; sem fallback local | AP | PT+UI | inference.MotorGemini; app.IAService | T27, IA-MOD-01 |
| TRV-IA-012 | §§47.6, 38.4 | Resposta só vale íntegra; cortada, bloqueada ou vazia nunca vira resultado; falhas → parada recuperável | AP | DET | inference.MotorGemini | T15, IA-RESP-01 |
| TRV-IA-013 | §§47.2, 38 | Nenhum modelo roda no computador; sem requisitos de GPU/RAM para a IA; janela de trabalho mantém o fatiamento | AP | PT | inference | T14, T16 |

## 12. Transversal — biblioteca jurídica e pesquisa web

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-LIB-001 | §§29.3, 40.1 | Biblioteca permanente, separada dos autos temporários | AP | DET | legal × session | LIB-SEP-01 |
| TRV-LIB-002 | §40.2 | Importar `docs/law_library` no preparo inicial; ampliação manual | AP | DET | legal.Bootstrap | T28 |
| TRV-LIB-003 | §40.3 | Guardar originais + metadados; preservar artigo, §, inciso, alínea e localização | AP | DET | legal.ParserNormas | LIB-PARSE-01…05 |
| TRV-LIB-004 | §§18.1, 40.3 | Busca exata + semântica; RAG não prova vigência; conferir a fonte antes de citar | AP | DET+VAL | legal.Busca; validate.LegalRefs | A13, LIB-HYB-01 |
| TRV-LIB-005 | §40.4 | Embeddings locais separados; trocar gerador não reindexa; trocar embeddings pode reindexar sem perder originais | AP | DET | legal.Indice | T21, LIB-REIDX-01 |
| TRV-LIB-006 | §40.5 | 5 categorias automáticas com correção manual; busca atravessa categorias | AP | DET+UI | legal | LIB-CAT-01 |
| TRV-LIB-007 | §40.6 | Duplicidade/versão → manter ambas ou substituir; nunca em silêncio; índices atualizados; versão identificada na citação | AP | DET+UI | legal.Versoes | T22, LIB-DUP-RDPM-01 |
| TRV-LIB-008 | §40.7 | Exclusão com confirmação; remove texto e vetores; bloqueia se usado na tarefa ativa; preserva modelos | AP | DET+UI | legal.Excluir | T23 |
| TRV-LIB-009 | §§18.3, 40.6 | Atualização do corpus só a pedido expresso; nada automático ao iniciar | AP | DET | legal | A14, A15 |
| TRV-LIB-010 | §41 | Pesquisa web **desligada por padrão**; preferência persistente | AP | DET+UI | config; legal/web | T19 |
| TRV-LIB-011 | §41 | Ligada: automática e condicional, fontes oficiais, sem dados sigilosos, sem incorporar ao corpus, conferir antes de citar | AP | SEG+DET | legal/web | T20, A13 |
| TRV-LIB-012 | D-01 | Nota de vigência curada para o art. 41 § 3º do RDPM; filtrar da redação; sem alerta | AP | DET | legal.NotaVigencia | LIB-D01-01/02 |

## 13. Transversal — validação, documentos, instalação, offline

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-VAL-001 | §42.1 | Validação automática obrigatória antes da revisão e após cada ajuste | AP | VAL | validate; workflow | T09, T10 |
| TRV-VAL-002 | §42.1-1 | Fidelidade de fatos, datas, depoimentos e fontes | AP | VAL | SupportCheck, DatesPersons | VAL-NEG-03 |
| TRV-VAL-003 | §42.1-2 | Referências jurídicas verificáveis; nenhuma inventada | AP | VAL | LegalRefs | T24, VAL-NEG-04 |
| TRV-VAL-004 | §42.1-3 | Regras PADS/SR específicas | AP | VAL | ScopePADS/ScopeSR | VAL-NEG-01/05/06 |
| TRV-VAL-005 | §42.1-4 | Coerência prova × fundamento × conclusão | AP | VAL+IA | SupportCheck | VAL-NEG-07 |
| TRV-VAL-006 | §42.1-5 | Cálculos conferidos | AP | VAL | CalcConsistency | VAL-NEG-08 |
| TRV-VAL-007 | §42.1-6 | Seções, campos e avisos obrigatórios | AP | VAL | RequiredParts | VAL-NEG-09 |
| TRV-VAL-008 | §42.1-7 | Coerência com o resultado confirmado pelo usuário | AP | VAL | UserDecisions | VAL-NEG-10 |
| TRV-VAL-009 | §42.2 | Correção automática por releitura; depois pendência ao usuário; limitação não impeditiva declarada; nunca "validado" genérico | AP | DET | workflow.LacoCorrecao | T24 |
| TRV-VAL-010 | §42.2-2 | Não alterar decisão confirmada em silêncio | AP | VAL | UserDecisions | T10 |
| TRV-VAL-014 | D-01 | Bloquear citação do art. 41 § 3º como vigente | AP | VAL | LegalRefs | LIB-D01-02 |
| TRV-EXP-001 | §32.1 | Padrão único; Times New Roman 12; `logo-pmro.png`; cabeçalho e título específicos | AP | TXT | export | T25 (D-04) |
| TRV-EXP-002 | §32.2 | Parâmetros de margem/recuo como proposta, ajustáveis pelos modelos | PT | TXT | export template | EXP-LAY-01 |
| TRV-EXP-003 | §§11, 32.1 | Nenhuma assinatura, publicação, homologação ou validação SEI fictícia | AP | VAL | NoFakeActs | VAL-NEG-11 |
| TRV-EXP-004 | §32.2 | Nome de arquivo seguro, com tipo + número do processo e sem dados pessoais | AP | DET | export.NomeArquivo | EXP-NOME-01 |
| TRV-EXP-005 | D-03 | DOCX (OOXML) e PDF (Typst) a partir do mesmo DocModel | AP | DET | export | EXP-EQ-01 |
| TRV-INS-001 | §34 | Instaladores `.exe` e `.msi`; x64 (D-02) | AP | PT | build/windows | INS-01…06 |
| TRV-INS-002 | §34 | Dependências instaladas automaticamente; intervenção manual só se indispensável | AP | PT | instalador + primeiro uso | INS-02 |
| TRV-INS-003 | §§34, 47.2 | Instalador leve: sem mecanismo de inferência, pesos de modelo nem chave de API | AP | PT | build | INS-03 |
| TRV-INS-004 | §34 | Atualização só por novo instalador | AP | PT | — | T26 |
| TRV-OFF-001 | §35 |  **REVOGADA (§47, D-10)** — Upload + modelos + biblioteca instalados → tudo offline | AP | DET | todos | T18 |
| TRV-OFF-002 | §§35, 47.3 | Sem consulta externa silenciosa: pesquisa web desligada por padrão; IA só após a confirmação de envio da tarefa | AP | SEG | legal/web; pipeline (envio_ia) | T19, SEG-NET-01 |
| TRV-OFF-003 | §47.6 | Sem internet/chave/cota/modelo na etapa de IA: mensagem clara, parada recuperável com estado preservado; funções sem IA offline | AP | UI | pipeline; workflow | T18, OFF-IA-01 |

## 14. Governança das regras

| ID | Origem V4 | Requisito | Rót. | Tipo | Módulo / política | Testes |
|---|---|---|---|---|---|---|
| TRV-GOV-001 | §§28.4, 46 | Não perder regra por compressão; cobertura V4 total | AP | DET | cmd/rulescov | GOV-COV-01 |
| TRV-GOV-002 | §§44, 14.3 | Conflito normativo demonstrado → usuário decide; aprovação funcional ≠ comprovação de vigência | AP | Processo | REGISTRO_DUVIDAS | GOV-CONF-01 |
| TRV-GOV-003 | PT | V4 com hash fixado; mudança exige revisão registrada | PT | DET | policies/V4.lock | GOV-LOCK-01 |
| TRV-GOV-004 | PT | Prompts compilados com golden files; completude por etapa | PT | DET | policies | POL-COMP-01…n |
