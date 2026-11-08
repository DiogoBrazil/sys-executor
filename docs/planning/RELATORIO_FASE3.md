# RELATÓRIO DA FASE 3 — MOTOR DETERMINÍSTICO (sys-executor)

- **Data:** 08/11/2026
- **Autorização:** expressa do usuário ("Pode começar, esta autorizado.").
- **Escopo:** F3-01 a F3-08 (plano §16): datas civis, reincidência, equivalências e janelas, comportamento e pontuação, dosimetria, invariantes de escopo PADS/SR, memória de cálculo, catálogo de regras em YAML e `rulescov` completo.
- **Situação:** **concluída** (08/11/2026). Tudo em Go puro, sem IA e sem interface nova. O motor ainda não está ligado ao fluxo da tarefa: as etapas de cálculo são das Fases 4–6.

## 1. Critério de saída (PLANO_TESTES_ACEITACAO §12: "Seção 1 inteira verde + GOV-COV-01 para as regras DET")

| Verificação | Resultado |
|---|---|
| **Seção 1 do plano de testes** (motor determinístico): os **101 casos** têm teste anotado (`// Cobre: <caso>`) e passam | **OK** — `rulescov`: "101 casos da seção 1; 0 sem teste anotado" |
| **GOV-COV-01 (catálogo)**: as 185 regras da matriz estão no catálogo; os 124 IDs de teste da V4 são citados por alguma regra | **OK** — 0 sem regra; catálogo e matriz com o mesmo conjunto |
| **GOV-COV-01 (regras DET)**: as **33 regras determinísticas do motor** têm `enforced_by`; os **56** símbolos citados existem no código; cada regra garantida tem teste anotado (`// Regra:` ou `// Cobre:`) | **OK** — `rulescov -estrito` sai com 0 |
| **GOV-LOCK-01**: V4 inalterada | **OK** |
| Contra-teste do `rulescov` | Um repositório sintético com cada tipo de defeito (ID da V4 sem regra, regra só na matriz, regra só no catálogo, símbolo inexistente, regra sem teste, regra do motor sem `enforced_by`, `// Regra:` com ID inexistente) reprova em **todos os sete** |

Testes de propriedade (além dos casos da V4):
- **janela deslizante = força bruta** em 3.000 conjuntos aleatórios de fatos (inclui 29/02);
- **a ordem de processamento não altera o resultado** (2.000 casos com fatos e sanções embaralhados; motivação da regra §15.4);
- **regressão nunca convive com descontos**; sem regressão, saldo final = informado − Σ descontos, e o insuficiente nunca é reposto (2.000 casos).

Cobertura de instruções: `engine` 92,8%, `civil` 92,2%, `catalog` 88,3%, `domain/pads` 89,9%, `domain/sr` 87,9%. A suíte da CI passa com `-race`.

## 2. O que foi construído

| Tarefa | Entrega |
|---|---|
| F3-01 `civil` | `internal/rules/civil`: data civil sem fuso, hora opcional, comparação (antes/depois/simultâneo/indeterminado), aniversário com o caso de 29/02 em ano não bissexto (`ErrCasoLimite29Fev`), **duas funções distintas**: `MenosDeUmAno` (janela, aniversário excluído, §15.2) e `DentroPrazoReincidencia` (aniversário incluído, D-07) |
| F3-02 Reincidência | `engine.Reincidencia`: mesma natureza; término do cumprimento estritamente anterior ao fato (com horas na mesma data; iguais ou ausentes → não reconhece); prazo 1/2/4 anos; várias punições válidas = **uma** agravante; cada descarte com motivo; término desconhecido → pendência. `ProximaPerguntaReincidencia`: só pergunta depois de fixada a natureza, e não repete o que já se sabe |
| F3-03 Equivalências e janelas | `engine.Equivaler` em unidades (L=1, M=2, G=4; I-01), sem duplicar fato e sem acusações pendentes. `engine.GatilhoBomOtimo`: janela deslizante, intervalo inferior a um ano, exige fato atual na janela (D-06; histórico sozinho → alerta e pergunta). `engine.GatilhoExcepcional`: 3ª grave equivalente no período, sem janela, desconsiderando fatos consumidos (I-02) |
| F3-04 Comportamento e pontuação | `engine.ClassificarSaldo` e `engine.Pontuacao`: regressão no mesmo PADS → valor fixo 9/20/30 **sem** descontos ordinários (§15.4); senão, descontos 1/2/4 por sanção (conexas: só a principal); insuficiente segue o ordinário; saldo negativo calculado e sinalizado; dados ausentes → cálculo não concluído; pontos × classificação divergentes → pergunta |
| F3-05 Dosimetria | `engine.Dosimetria`: repreensão sem dias; atenuantes × agravantes reconhecidas → 2/3/1 dias, com a contagem da D-05; circunstância do comportamento **anterior**; circunstância sem fundamento não conta e vira pendência; relevantes serviços só com justificativa. `ValidarDias` (detenção ≤ 10; I-03) e `VerificarArt36` (sete fatores, "ausência de elementos" vale como exame). Conexão: `SugerirPrincipal` (só sugere; I-04) e `AplicarConexao` (sem confirmação → pendência e nenhuma sanção do grupo) |
| F3-06 Invariantes de escopo | `internal/domain/pads`: `ImputacaoSet` versionado e auditável (congelado não muda; reabrir exige motivo e invalida o julgamento iniciado na versão anterior); `Julgamento` com **um acusado**, resultado só para imputação do memorando, resultado global (procedência / procedência parcial / improcedência) só com todas as imputações confirmadas; `VerificarCalculo` confere em runtime: sanção ⇒ procedência confirmada, uma sanção por imputação, regressão ⇒ nenhum desconto, desconto ⇒ sanção. `internal/domain/sr`: `ObjetoSR` (portaria intocada), autorizações de fato novo e militar novo (pendente → nada em III/IV; recusado → ausente), sindicados = formais + autorizados, civil nunca sindicado; `ValidarConclusoes`: fato admitido, sujeito sindicado, civil fora da IV, uma conclusão por fato × sindicado, enquadramento com diploma, artigo e fonte. `ConclusaoSR` **não tem** campos de sanção, pontuação ou reincidência (teste por reflexão) |
| F3-07 Memória de cálculo | `engine.MontarMemoria`: os sete itens do §8.8, em pt-BR; "Propõe-se … A alteração depende do ato administrativo competente"; `[A PREENCHER]` quando faltam dados |
| F3-08 Catálogo + `rulescov` | `internal/rules/catalog/*.yaml` (14 arquivos, um por seção da matriz; 185 regras), embutido no executável, carga **estrita** (campo desconhecido = erro) e validação (ID, fluxo × prefixo, rótulos, tipos, testes, símbolos, versão, situação). `cmd/rulescov` passou a conferir catálogo × matriz × V4 × código × testes |

## 3. Decisões técnicas (registradas no REGISTRO como PT-F3-xx)

1. **PT-F3-01** Caso de 29/02 "material": o motor calcula com 28/02 **e** com 01/03; só pergunta ao usuário se os dois resultados diferirem. Não impõe critério (PADS-CMP-011).
2. **PT-F3-02** Formato do catálogo: `rotulo` e `tipo` são listas (a matriz combina, ex.: AP/IN, DET+PRG); "PT" na coluna de tipo virou `ARQUITETURA` e "Processo" virou `PROCESSO`; `modulo` guarda o módulo planejado na matriz e `enforced_by`, os símbolos Go reais, conferidos pelo `rulescov`.
3. **PT-F3-03** Duas anotações nos testes: `// Cobre: <caso do plano de testes>` e `// Regra: <ID do catálogo>`. Casos de ponta a ponta (PADS-E01…E05) não são marcados como cobertos por teste de unidade; a regra correspondente é.
4. **PT-F3-04** Reabrir o conjunto de imputações exige motivo, sobe a versão e faz o julgamento iniciado na versão anterior recusar novos resultados.
5. **PT-F3-05** SR: "indícios" exige enquadramento com diploma, artigo e fonte confirmada; "ausência" não tem enquadramento; "inconclusivo" pode ter. **[A VALIDAR na Fase 6]** com minutas de SR.
6. **PT-F3-06** Dependência nova: `go.yaml.in/yaml/v3` v3.0.5 (MIT/Apache-2.0), que já era dependência indireta; `cmd/licencas` aprovado.

## 4. O que fica para as próximas fases

- **SR-EST-010** (papel da pessoa por ato) e **SR-ESC-004** (situação funcional na data de cada fato) dependem da extração de pessoas e fatos: Fase 4/6.
- Ligar o motor ao fluxo (perguntas de reincidência, comportamento e conexão; memória de cálculo na interface: UI-CALC-01/02): Fases 4–6.
- Validadores da minuta (`CalcConsistency`, `ScopePADS`, `ScopeSR`): Fase 6, usando `VerificarCalculo` e `ValidarConclusoes`.
- Campos do catálogo para a IA (`etapas`, `condicao`, `instrucao_ia`, `falha`): Fase 5, com os testes `POL-COMP-*`.

Nenhuma pendência depende do usuário nesta fase.
