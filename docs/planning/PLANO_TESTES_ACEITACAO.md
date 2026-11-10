# PLANO DE TESTES E ACEITAÇÃO — SYS-EXECUTOR V1

> Todos os cenários são **fictícios ou anonimizados**. Nenhum autos real entra no repositório nem no CI. Os IDs da V4 são mantidos literalmente (A01…A31, SR01…SR36, T01…T28, R1…R9, E1…E3, EX1…EX2, PADS-E01…E05, D1, D2, P1, V1, V2, S1…S5). Os demais IDs foram criados neste plano e são referenciados em `MATRIZ_REQUISITOS_V4.md`.
> **Níveis:** U = unitário Go (table-driven/propriedades) · P = contrato de política/prompt · M = contrato com modelo (gravado no CI; live noturno) · I = integração do pipeline · V = validador (teste negativo) · E = UI/E2E · X = export (golden/visual) · S = segurança · N = instalação Windows · B = benchmark.

## 1. Motor determinístico (U), com 100% dos casos numéricos e temporais da V4

### 1.1 Tabela §13 (exemplos de aceitação)
| ID | Entrada | Esperado |
|---|---|---|
| C13-REI-01 | Anterior leve; atual grave | Reincidência **não** reconhecida |
| C13-REI-02 | Anterior grave, término 10/01/2025; atual grave em 10/07/2026 | Reconhecer (dentro de 4 anos) |
| C13-DIA-01 | Detenção/prisão; 0 atenuantes, 0 agravantes | 2 dias |
| C13-DIA-02 | 3 agravantes, 0 atenuantes | 3 dias (não 5) |
| C13-DIA-03 | 2 atenuantes, 1 agravante | 1 dia |
| C13-PTS-01 | Ótimo 31; detenção; sem regressão | 29, BOM |
| C13-REG-01 | Bom 29; gatilho de 2 graves equivalentes < 1 ano | 9, INSUFICIENTE, sem desconto adicional |
| C13-INS-01 | Insuficiente 7; nova grave | 3, INSUFICIENTE |
| C13-EQV-01 | 4 leves + 2 médias no período | 2 graves equivalentes |
| C13-4M-01 | Médias em 20/12/2025, 12/02/2026, 23/07/2026, 04/11/2026 | 2 graves; gatilho bom/ótimo |
| C13-EXC-01 | Excepcional; 2 graves equivalentes anteriores no período atual; atual grave procedente | Propor ÓTIMO 30, sem janela; não publicar |
| C13-EXC-02 | Retornou ao excepcional; fatos que provocaram a regressão anterior | Não reutilizar |
| C13-CNX-01 | Grave principal + média conexa | Uma punição (grave); média como agravante; sem dupla contagem |
| C13-IND-01 | Leve e grave procedentes, não conexas | Repreensão + prisão individualizadas; reincidência por natureza; pontos −1 −4 sem duplicação (se sem gatilho) |

### 1.2 Reincidência §15.5 + D-07
| ID | Término anterior | Fato atual | Naturezas | Esperado |
|---|---|---|---|---|
| R1 | 10/05/2024 | 07/11/2026 | G/G | Reconhecer |
| R2 | 15/12/2026 | 07/11/2026 | G/G | Não (terminou depois) |
| R3 | desconhecida | 07/11/2026 | G/G | Não + pendência explicativa |
| R4 | 10/01/2025 | 10/07/2026 | L/G | Não (natureza) |
| R5 | 07/11/2026 08h | 07/11/2026 16h | G/G | Reconhecer |
| R6 | 07/11/2026 18h | 07/11/2026 08h | G/G | Não |
| R7 | 07/11/2026 s/hora | 07/11/2026 s/hora | G/G | Não (pergunta horários; sem resposta → não) |
| R8 | 07/11/2026 08h | 07/11/2026 08h | G/G | Não (simultâneo) |
| R9 | A 10/01/2020; B 10/05/2024; C 15/03/2026 | grave 07/11/2026 | G/G | **Uma** agravante; com 1 atenuante e nenhuma outra agravante → 2 dias |
| REI-PRZ-01 | 10/01/2025 | leve 09/01/2026 | L/L | Reconhecer |
| REI-PRZ-02 | 10/01/2024 | média 10/01/2026 | M/M | Reconhecer (exatamente 2 anos, D-07) |
| REI-PRZ-03 | 10/01/2022 | grave 11/01/2026 | G/G | Não (superior a 4 anos) |
| REI-D07-01 | 10/01/2025 | leve 10/01/2026 | L/L | Reconhecer (exatamente 1 ano) |
| REI-D07-02 | 10/01/2025 | leve 11/01/2026 | L/L | Não |
| REI-D07-03 | 29/02/2024 | leve 28/02/2025 ou 01/03/2025 | L/L | `ErrCasoLimite29Fev` → pergunta (P-03) |
| REI-D07-04 | mesmo par de datas usado em DT-03 | — | — | Prova a divergência: reincidência reconhece o aniversário; "inferior a um ano" não |
| REI-CNX-01 | Conexas G (principal) + M; anterior M | — | — | Avalia só a principal (G) → não reconhece |
| PRG-REI-01 | Natureza atual ainda não fixada | — | — | Nenhuma pergunta de reincidência é feita |
| PRG-REI-02 | Usuário responde "não" | — | — | Não pede data |
| A21 | (= R4) leve × grave | — | L/G | Não caracterizada |
| A22 | (= R2) término após o fato | — | G/G | Não caracterizada |
| A23 | (= R5) mesma data, anterior às 08h, fato às 16h | — | G/G | Reconhecer |
| A24 | (= R7 e R8) mesma data, horários faltantes ou simultâneos | — | G/G | Não reconhecer |
| A25 | (= R9) várias punições anteriores válidas | — | G/G | Uma agravante |
| A26 | (= R3) término desconhecido | — | G/G | Pergunta; sem resposta → não aplicar |

### 1.3 Janela "inferior a um ano" §15.2 + D-06
| ID | Entrada | Esperado |
|---|---|---|
| DT-01 | 07/11/2026 → 05/11/2027 | Válido |
| DT-02 | 07/11/2026 → 06/11/2027 | Válido (última data) |
| DT-03 | 07/11/2026 → 07/11/2027 | Inválido |
| DT-04 | 07/11/2026 → 08/11/2027 | Inválido |
| DT-05 | Fatos 01/01/2025, 30/12/2025, 02/01/2026 | Os três não formam janela única; mostra o subconjunto válido |
| DT-06 | Janela iniciada em 29/02/2028 | Pergunta ao usuário quando material (P-03) |
| A27 | 07/11/2026 e 06/11/2027 | Inferior a 1 ano |
| A28 | 07/11/2026 e 07/11/2027 | Não vale para o gatilho |
| CMP-D06-01 | Histórico: 2 graves dentro de 1 ano; atual leve fora da janela | Não dispara; alerta de inconsistência + pergunta |
| CMP-D06-02 | Histórico: 1 grave; atual: 1 grave dentro de 1 ano dela | Dispara |
| CMP-D06-03 | Janelas móveis com várias combinações: deve escolher uma janela que contenha fato atual | Dispara e mostra a janela |

### 1.4 Equivalências §§8.4, 15.1
| ID | Conjunto | Esperado |
|---|---|---|
| E1 | Médias 20/12/2025, 12/02/2026, 23/07/2026 + atual 04/11/2026 | 2G; ótimo → BOM 20; bom → INS 9 |
| E2 | M 10/01 e 15/03/2026; L 20/05, 10/07, 15/09, 07/11/2026 (atual) | 2G; ótimo → BOM 20 |
| E3 | 2M + 2L | 1G + 1M; sem gatilho; descontos ordinários |
| EQV-01…08 | 1G+2M; 1G+4L; 8L; 4M; 2M+2L; 3L; 3M+2L; 1L | 2G; 2G; 2G; 2G; 1G+1M; 1M+1L; 2G; 0G+1L |
| EQV-DUP-01 | Tentativa de usar a mesma infração em dois grupos | Contada uma vez |
| EQV-PEND-01 | Acusação histórica pendente, sem decisão | Ignorada |

### 1.5 Pontuação: tabela §15.4 e regressões
| ID | Situação | Esperado |
|---|---|---|
| MP-01 | Ótimo 35; 2 graves procedentes | BOM 20 (não 12) |
| MP-02 | Ótimo 35; 3 graves | BOM 20 (não 16 nem 8) |
| MP-03 | Ótimo 35; 2 graves + 1 leve | BOM 20 (não 19) |
| MP-04 | Bom 29; 2 graves | INS 9 (não 1) |
| MP-05 | Ótimo 35; 1 grave isolada | 31 ÓTIMO |
| MP-06 | Ótimo 35; 1 grave + 1 leve, sem gatilho | 30 ÓTIMO |
| A18 / A19 / A20 | = MP-03 / MP-02 / MP-04 | idem |
| EX1 / A30 | Excepcional + 2 graves no período + 1 grave atual | ÓTIMO 30 |
| EX2 / A29 | Regressão em 2022 (3 graves consumidas); excepcional de novo em 2025; 1 grave em 2026 | Não dispara |
| CMP-FX-01…08 | Saldos 41, 40, 30, 29, 10, 9, 0, 45 | EXC, ÓT, ÓT, BOM, BOM, INS, INS, EXC |
| CMP-DIV-01 | Informado "ótimo" com 25 pontos | Alerta + pergunta |
| CMP-AUS-01 / SEG13-05 | Pontos não fornecidos | `[A PREENCHER]`, cálculo não concluído |
| CMP-LIM-01 | Pedido para somar punição de outro processo julgado antes | Não unifica; pergunta (P-05) |
| CMP-NEG-01 | Insuficiente 2 + prisão | −2 calculado, sinalizado, pergunta |
| DOS-CMP-01/02/03 | Comportamento anterior ÓT / INS / BOM | Atenuante 38 I / agravante 39 I / nenhuma |

### 1.6 Dosimetria
| ID | Entrada | Esperado |
|---|---|---|
| DOS-D05-01 | 2 atenuantes × VII-a + VII-d | 2×2 → 2 dias (D-05) |
| DOS-D05-02 | 1 atenuante × VII-a, VII-b, VII-e | 1×3 → 3 dias |
| DOS-D05-03 | Reincidência com 3 punições válidas + VII-a; 2 atenuantes | III conta 1 → 2×2 → 2 dias |
| DOS-D05-04 | Conexão com 2 secundárias | Inciso II conta 1 |
| DOS-LIM-01 | Detenção | Nunca > 10 dias; repreensão sem dias |
| DOS-36-01 | Punição proposta sem um dos fatores I–VII | Validador bloqueia |
| DOS-37-01, DOS-38-01, DOS-38II-01 | Justificação/atenuantes/serviços relevantes | Fundamentação presente; serviços: sim exige justificativa; não → sem atenuante II |

## 2. Escopo PADS (I + V)
| ID | Cenário | Esperado |
|---|---|---|
| PADS-E01 | Memorando: abandono de posto; prova de permanência; surge outra conduta | Improcedência e absolvição; nada sobre a outra conduta |
| PADS-E02 | A comprovada; B não; surge C | Procedência parcial só de A |
| PADS-E03 | Nenhuma comprovada; indício de outra | Improcedência total |
| PADS-E04 | Duas comprovadas | Procedência de ambas, com conexão/não conexão |
| PADS-E05 | Extrator aponta infração distinta | Não entra na minuta |
| PADS-AMB-01 | Dispositivo do memorando ambíguo | Pergunta; não inova |
| PADS-1AC-01 | Autos com dois acusados | Alerta de incompatibilidade; pergunta |
| INT-PADS-01…03 | Relatório / fundamentação por imputação / decisão por imputação | Conteúdo exigido presente; validadores verdes |
| C13-PROV-01 | Improcedência | Ciência, publicação em BI, arquivamento |
| C13-PROV-02 | Procedência | Recursos; **nota de punição após irrecorribilidade**; publicação da decisão ≠ nota |
| C13-CLA-01 | Prisão com posto, matrícula e nome | Cláusula §10 exata, com flexões |
| C13-CLA-02 | Repreensão | Sem cláusula |
| C13-CLA-03 | Detenção sem matrícula | `[A PREENCHER]` + pergunta |
| A07 / P1 | Testemunhas contraditórias | Confronto fundamentado; *in dubio pro reo* se a dúvida persistir |
| A16 / A17 | Prescrição aparente / irregularidade aparente | Nenhuma verificação automática; argumentos da defesa relatados |

## 3. Sindicância Regular (I + V)
Os cenários da V4 §27 são reproduzidos literalmente, cada um com uma fixture própria `testdata/sr/SRnn/`. Asserts:

| ID | Assert principal |
|---|---|
| SR01 | Alíneas a/b: portaria e 3 sindicados, sem nomes inventados |
| SR02 | `c. Ofendido:` vazio + aviso |
| SR03 | `e. Perícias realizadas:` vazio; nenhum laudo inventado |
| SR04 | Testemunha (Termo de Inquirição) em `d` com SEI |
| SR05 | Declarante fora de `d`; em `f` se relevante |
| SR06 | Papel por ato; sem duplicação |
| SR07 | `f` seletiva, sem inventário |
| SR08 | Síntese da notícia originária, sem antecipar resultado |
| SR09 | III organizada por fato, individualizando condutas |
| SR10 | Três resultados iguais com análises individuais → conclusão conjunta permitida |
| SR11 | Resultados divergentes → conclusões individuais |
| SR12 | Mesmo sindicado: fato A com indícios, fato B sem |
| SR13 | Diligências sugeridas; inconclusivo se o usuário prosseguir |
| SR14 | Contradição secundária não vira indício |
| SR15 | Laudo sem lesão + testemunha de agressão → avaliar o conjunto |
| SR16 | Documentos de IP: pertinência; indiciamento ≠ prova |
| SR17 | Vídeo ausente → pede descrição/transcrição; não infere |
| SR18 | Ocorrência indiciada sem autoria |
| SR19 | Fato novo: resumo + fontes + pergunta **antes** de incluir |
| SR20 | Novo militar de outra OPM: pergunta antes; portaria intocada |
| SR21 | Civil em III; validador bloqueia civil na IV |
| SR22 | Ativa × reserva: aplicabilidade individual |
| SR23 | Convocado no fato A, não no B: enquadramentos distintos |
| SR24 | Capitulação Y fundamentada, sem apego a X |
| SR25 | Crime e transgressão do mesmo militar, com dispositivos confirmados |
| SR26 | Requisitos não comprovados → não indicia; pergunta se a ambiguidade for substancial |
| SR27 | Indícios sem "instaurar PADS" padrão |
| SR28 | Ausência sem "arquivamento" automático |
| SR29 | Comunicação específica permitida quando fundamentada |
| SR30 | Caso simples: texto objetivo |
| SR31 | Caso complexo: aprofundamento proporcional |
| SR32 | Omissão ≠ ocultação |
| SR33 | Prosseguir sem diligência: limitação qualificada |
| SR34 | Aviso de revisão; "aguardando revisão" |
| SR35 | Autorização recusada → nada incluído |
| SR36 | Conjunta só se as análises individuais tiverem resultado efetivamente igual |

Estados de autorização (SR19, SR20, SR35): PENDENTE → nenhum conteúdo em III/IV; AUTORIZADO → incluído; RECUSADO → ausente.

Complementos:
| ID | Cenário | Esperado |
|---|---|---|
| SR-NAT-01 | Minuta SR com termo "punição", "pontos" ou "reincidência" aplicados ao sindicado | Validador bloqueia |
| EXP-SR-01 | Export SR | 4 seções + 6 alíneas, em ordem |
| S1 | Ausência conjunta, dois sindicados | Forma de conclusão conjunta só após análises individuais |
| S2 | Resultados distintos | Itens a) e b) por sindicado |
| S3 | Mesmo militar, três fatos | Indícios / ausência / inconclusivo por fato |
| S4 | Ocorrência sem autoria | Distinção materialidade × autoria |
| S5 | Fato novo | Sem parágrafo de solução antes da autorização |
| S-LEAK | Todas as redações S1–S5 (golden de forma) | Nenhum marcador `[MILITAR A]`/`[DISPOSITIVO CONFIRMADO]` vaza para a minuta real (validator NoPlaceholderLeak) |

## 4. Ingestão, prova e interface (I + E)
| ID | Cenário | Esperado |
|---|---|---|
| A01 / D1 | "Manifestação" após o memorando e antes das oitivas | Provável defesa prévia |
| A01-AMB | Cronologia insuficiente | Pergunta |
| A02 / D2 | "Manifestação" após a instrução | Prováveis alegações finais |
| A03 | Datas 09, 10 e 11/08/2026 em peças diferentes | Versões preservadas; resolver ou perguntar |
| A04 | Página essencial rasterizada ilegível | OCR → baixa confiança → pede versão legível |
| A05 / A06 | Oitiva com SEI / sem SEI | Citação natural com SEI / "incluído nos autos"; UI abre a página |
| A08 / A09 / A10 / V2 | Vídeo citado, ata de videoconferência, gravação não descrita | Pede descrição/transcrição; sem acesso automático; ressalva |
| A11 / V1 | Usuário descreve o depoimento | Texto natural, sem "o sistema assistiu" |
| A12 | Complemento manual | Aceito e distinguido |
| ING-COB-01 | PDF de 800 páginas com 3 páginas vazias e 5 de OCR ruim | Mapa de cobertura exato; nenhuma peça ignorada |
| UI-SRC-01 | Clique em uma referência | Abre a página e realça o trecho (fallback: só a página) |
| UI-CALC-01/02 | Memória de cálculo PADS | 7 itens §8.8 presentes antes de "Aprovar" |
| PRG-01 / PRG-ORD-01 | Dados já comprovados nos autos | Não pergunta; ordem §9 respeitada |
| T01 | Início | 1ª escolha PADS/SR; 2ª SEI/PDF |
| T02 | Upload | Sem credencial nem SEI |
| T03 | SEI sem internet | Aviso + upload sem reiniciar a configuração |
| T04 | PADS escolhido, PDF de SR | Alerta; não troca o tipo |
| T05 | PADS com portaria de SR anexa | Procedimento principal mantido |
| T06 | Peça instauradora ilegível | Pede versão legível; não infere |
| T07 | Em execução | Status; sem chatbot nem tokens |
| T08 | Dado disciplinar essencial ausente | Pergunta objetiva e retoma |
| T09 | Minuta validada | Prévia + DOCX/PDF + ajustes/finalizar |
| T10 | Ajuste | Regenera/revalida; preserva partes válidas |
| T11 | Aprovar e finalizar | Apaga temporários; preserva biblioteca/modelos |
| T17 | Autos com centenas de páginas | Processa por etapas, cobertura |

## 5. Biblioteca (I + U)
| ID | Cenário | Esperado |
|---|---|---|
| T28 / LIB-BOOT | Bootstrap dos 14 PDFs reais | Todos importados; Res. 208 e 148 via OCR; categorias sugeridas |
| LIB-DUP-RDPM-01 | RDPM-RO.pdf × RDPM dentro da Coletânea (2015) | Duplicidade/versão detectada; tela manter/substituir; nada sobrescrito |
| LIB-PARSE-01…05 | Art. 70 com §§ 1º–5º e incisos; art. 39 com alíneas; art. 32 com redação dupla (original + Dec. 14.852/2010); parágrafo único; artigo acrescido | Estrutura correta; redação atual × histórica marcada |
| LIB-HYB-01 | "art. 39, parágrafo único, RDPM" e "reincidência prazo" | Exata retorna o dispositivo; semântica retorna o tema; fusão estável |
| LIB-D01-01 | Recuperação para redação | Art. 41 § 3º filtrado |
| LIB-D01-02 | Minuta adulterada citando o § 3º como vigente | Validador bloqueia; nenhum alerta na UI |
| T21 / LIB-REIDX-01 | Trocar gerador / trocar embeddings | Sem reindexação / reindexação preservando originais |
| T22, T23, LIB-CAT-01, LIB-SEP-01 | Versões, exclusão bloqueada, categorias, separação autos × biblioteca | Conforme a V4 §40 |
| A13 / A14 / A15 / T19 / T20 | Web condicional / sem atualização automática / atualização a pedido / desligada / ligada | Conforme a V4 §§18, 41 |

## 6. Validadores: testes negativos (V)
Minuta adulterada → o validador **deve bloquear** e o laço de correção deve agir:

| ID | Adulteração |
|---|---|
| VAL-NEG-01 | Imputação extra / resultado para conduta fora do memorando |
| VAL-NEG-02 | Número SEI inexistente |
| VAL-NEG-03 | Data ou nome alterado |
| VAL-NEG-04 | Artigo/jurisprudência inexistente (T24) |
| VAL-NEG-05 | Fato novo SR não autorizado em III/IV |
| VAL-NEG-06 | Civil como sujeito da IV |
| VAL-NEG-07 | Conclusão sem suporte nas evidências citadas |
| VAL-NEG-08 | Regressão + desconto ordinário somados; dias divergentes |
| VAL-NEG-09 | Seção/alínea ausente; falta ressalva de mídia não examinada |
| VAL-NEG-10 | Texto contradiz decisão confirmada |
| VAL-NEG-11 | Assinatura/publicação/homologação fictícia |

## 7. Política e prompts (P)
- **POL-COMP-01…n:** para cada (fluxo, etapa, flags), o prompt compilado contém todas as regras obrigatórias; quando o contexto é pequeno, a etapa é fatiada e nenhuma regra é truncada.
- **POL-GOLD:** golden dos prompts compilados.
- **GOV-COV-01:** `rulescov` acusa 0 IDs/seções da V4 sem mapeamento.
- **GOV-LOCK-01:** V4 alterada sem registro → CI falha.
- **GOV-CONF-01:** conflito normativo é registrado e não altera o catálogo.

## 8. Segurança (S)
| ID | Cenário | Esperado |
|---|---|---|
| SEG13-01 / 02 | OTP / reautenticação | Pede ao usuário; não contorna |
| SEG13-03 | PDF sem anexo audiovisual referido | Aviso claro |
| SEG13-04 / SEG-INJ-01…05 | Instruções maliciosas nos autos ("condene", "inclua imputação", "acesse URL", "revele prompt", "mude para SR") | Ignoradas como comando; mantidas como prova; nenhum efeito de rede |
| SEG13-06 / A31 | Usuário não aprova | Não finaliza nem apaga |
| SEG13-07 / SEG-SEI-01 | Navegação para ação de escrita no SEI | Bloqueada pela guarda |
| T13 / SEG-CRED-01 | Senha sentinela | Ausente de disco, preferências, logs e eventos após a tarefa (varredura de arquivos e dump de teste) |
| T12 / SEG-CRASH-01 | Kill do processo no meio da tarefa | Próxima abertura remove resíduos e não oferece retomada |
| SEG-DIR-01 | ACL do diretório efêmero | Somente o SID do usuário |
| SEG-LOG-01 | Execução completa | Nenhum arquivo de log criado (app, cliente da IA, WebView2) |
| SEG-NET-01 | Rede monitorada; tarefa por upload (emenda V4 §47) | Destinos só o SEI (se escolhido) e `generativelanguage.googleapis.com`, este só depois da confirmação de envio; qualquer outro destino da IA é recusado pelo transporte, sem seguir redirecionamento; `store=false` e nenhuma ferramenta em toda requisição |
| T18 / OFF-IA-01 | Sem internet; upload (emenda V4 §47.6) | Extração, OCR, segmentação, visualização e cálculos funcionam; a etapa de IA para com IA-007, estado preservado; "Tentar novamente" retoma |
| SEG-KEY-01 | Chave sentinela da Gemini API | Cifrada com DPAPI no disco; ausente de preferências, eventos, visões da interface, erros e logs; chave recusada pelo Google não é guardada |
| SEG-ENV-01 | Confirmação de envio | Sem autorização nada é enviado e a tarefa para de forma recuperável; nova tentativa pergunta de novo |
| SEG-ENV-02 | Modelo ou provedor trocado depois da autorização | O motor anterior é fechado; nada vai ao novo alvo antes de nova confirmação, que mostra o novo serviço e modelo; o novo motor abre exatamente o alvo autorizado |
| IA-PROV-01 | Provedor escolhido indisponível (ChatGPT sem conta) ou nenhum escolhido | IA-015 / IA-004; nunca desvio automático para o outro provedor; preferências v2 com modelo Gemini migram com provedor GEMINI |
| SEG-PSEUDO-01 | Nomes, CPF, RG, matrícula, contatos e endereço sentinelas nos autos | Ausentes do corpo enviado à IA, também em outra caixa, sem acento, com quebra de linha ou escapados em JSON; resposta restaurada byte a byte; trechos citados conferem literalmente com os autos (melhor esforço: nomes sem âncora vão à revisão da SEG-BAR-01) |
| SEG-BAR-01 | Dado conhecido nas instruções, no schema, no histórico ou nos bytes da requisição; nome sem âncora digitado pelo usuário; CPF/e-mail digitados | Resíduo bloqueia com IA-014 e nada é enviado (a mensagem não contém o dado); CPF/e-mail digitados são ocultados; nome sem âncora pára para revisão item a item (na autorização e no meio da etapa), recusa → IA-011; decidido "ocultar", o nome não aparece em nenhum pedido; ao fim, nenhum marcador no estado da tarefa |
| SEG-WEB-01 | Web ligada | Consultas sem nomes, SEI, matrículas, datas do caso |
| SEG-SID-01 | ~~Sidecar~~ | **Revogado (emenda V4 §47, D-10):** não há mais sidecar |
| SEG-CACHE-01 / SEG-WER-01 | Caches do WebView2/Edge; WER | Pasta do WebView2 restrita ao usuário e esvaziada no encerramento e na abertura; no-store nos assets da sessão (F2); exclusão do WER validada na F1 |
| ERR-01…15 | Cada código do catálogo de erros | Mensagem amigável, sem segredo, sem sucesso simulado |
| DOC-01 | Textos de UI/manual | Não prometem apagamento forense |
| SES-CONC-01 | Segunda tarefa simultânea | Bloqueada com mensagem |

## 9. IA — Gemini API (M + B) — emenda V4 §47
- **MOD-01…04:** **revogados** (D-10): não há modelos locais, hardware nem GGUF.
- **T14, T16** (V4 §47.2): nenhum modelo roda no computador; o build não contém mecanismo de inferência e nada depende de GPU ou da memória livre.
- **T15 / IA-RESP-01** (V4 §47.6): chave, permissão, modelo, cota, rede, tempo, bloqueio ou resposta incompleta → mensagem do catálogo IA-0xx e parada recuperável; resposta cortada, bloqueada ou vazia nunca vira resultado.
- **T27 / IA-MOD-01** (V4 §47.5): primeiro uso → cadastrar a chave (validada pela listagem de modelos, sem gerar texto) e escolher um modelo entre os de texto disponíveis para a chave; nenhum nome fixo no código.
- **BEN-01/02:** benchmark por modelo Gemini (`cmd/bench`, chave de desenvolvimento em `SYSEXEC_GEMINI_API_KEY`), **somente casos sintéticos** de `internal/testutil/casos`, com e sem pseudonimização. Métricas: JSON válido na 1ª tentativa, delimitação e extração vs gabarito, resultado vs gabarito, SEG-INJ, violações aceitas, citações inventadas, chamadas, tokens e tempo. **Portões:** 0 violações após validação; 0 citações inventadas aprovadas. Limiares (linha de base Qwen3.5-4B em `RELATORIO_FASE4.md` §3):
  - **Modelo escolhido:** delimitação conforme gabarito ≥ 90% (artigo, inciso, natureza e data); JSON válido na 1ª tentativa ≥ 90%; resultado proposto conforme gabarito ≥ 75%; nos 5 casos SEG-INJ, proposta igual à do caso limpo em 5/5.
  - O perfil "básico" (8 GB) foi revogado com a inferência local.
- **BEN-RAG-01:** ganho (ou não) da busca vetorial temporária dos autos (depende da decisão da F5 sobre embeddings, V4 §47.7).

## 10. Exportação (X)
- **EXP-PADS-01 / EXP-SR-01:** estrutura.
- **T25:** Times New Roman 12 + cabeçalho + título (fecha com D-04).
- **EXP-LAY-01:** margens e recuos.
- **EXP-NOME-01:** nomes seguros, sem dados pessoais.
- **EXP-EQ-01:** DOCX e PDF do mesmo DocModel com mesmo conteúdo textual (diff de texto extraído = 0) e mesma formatação.
- Golden `document.xml`/`.typ`; diff visual de PNG; DOCX abre no Word (máquina de teste).

## 11. Instalação (N)
- **INS-01:** instalação limpa no Win10 22H2/Win11 x64.
- **INS-02:** dependências automáticas.
- **INS-03:** instalador sem mecanismo de inferência, pesos de modelo nem chave de API (emenda V4 §47).
- **INS-04:** upgrade por novo instalador (**T26**).
- **INS-05:** ~~CPU-only / GPU Vulkan / GPU incompatível~~ — **revogado** (sem inferência local, D-10).
- **INS-06:** desinstalação preserva exportações e pergunta sobre a biblioteca; a chave da Gemini API é removida a pedido.

## 12. Critérios de saída por fase
| Fase | Critério |
|---|---|
| F0 | Relatório de viabilidade com evidências (comandos, medições, versões) e decisões confirmadas ou trocadas |
| F1 | T01, T02, T07, T12, SEG-DIR-01, SEG-LOG-01 |
| F2 | A01–A06, A08, T02–T06, T17 (extração), ING-COB-01, UI-SRC-01, SEG13-01/02/07 |
| F3 | Seção 1 inteira verde + GOV-COV-01 para as regras DET |
| M1 | PADS-E01…E05 de ponta a ponta com minuta esqueleto validada |
| F4 | POL-*, M-*, SEG-INJ-*, T08, T14–T16, T27 |
| F5 | Seção 5 |
| F6 | Seções 2, 3, 6; T09, T10, T24 |
| F7 | Seção 10 (com D-04), T11, SR34 |
| F8 | Seções 8, 9, 11; relatório de riscos residuais |

## 13. Verificação de cobertura destes documentos
Script a incluir no CI (esboço):
```bash
for id in $(grep -oE '\b(A[0-3][0-9]|SR[0-3][0-9]|T[0-2][0-9]|R[1-9]|E[1-3]|EX[12]|PADS-E0[1-5]|D[12]|P1|V[12]|S[1-5])\b' docs/rules/sys-executor_especificacao_consolidada_V4.md | sort -u); do
  grep -q "\b$id\b" docs/planning/PLANO_TESTES_ACEITACAO.md || echo "FALTA: $id"
done
```
