# RELATÓRIO DA FASE 4 — IA LOCAL E ORQUESTRAÇÃO (sys-executor)

- **Data:** 08/11/2026
- **Autorização:** expressa do usuário ("Sim. Pode iniciar a Fase 4").
- **Escopo:** F4-01 a F4-07 (plano §16): gerenciador de modelos, hardware e primeiro uso; cliente do llama-server com reenvios; policy packs com compilação e golden; etapas 2–9 com schemas; perguntas com DAG de invalidação; fatiamento; injeção de prompt.
- **Situação:** **implementada e testada** (simuladores + modelo real em CPU no WSL). Falta a validação pelo usuário no Windows (§5). A tarefa agora vai do PDF até as confirmações da etapa 9 e pára na consulta à biblioteca jurídica (Fase 5).

## 1. Critério de saída (PLANO_TESTES_ACEITACAO §12: "POL-*, M-*, SEG-INJ-*, T08, T14–T16, T27")

| Critério | Verificação | Resultado |
|---|---|---|
| **POL-COMP-01** | Para cada fluxo × etapa × combinação de condições (2 × 9 × 4 = 72), o prompt compilado contém todas as regras obrigatórias literalmente, com o ID, e nenhuma do outro fluxo; sem espaço, a montagem pede fatiamento (nunca trunca) | **OK** |
| **POL-GOLD** | 18 golden files (fluxo × etapa); diff obrigatório a cada mudança de regra ou pack. Hash do bundle idêntico no Linux e no Windows (LF fixado) | **OK** |
| **MOD-01** | Um gerador por tarefa, carregado uma vez e fechado ao fim; reservado contra remoção enquanto em uso | **OK** |
| **MOD-02** | Recomendação por hardware entre perfis testados (16 GB → 4B; 8 GB → 2B); importado fora do catálogo só por escolha manual | **OK** |
| **MOD-03** | Importação rejeita arquivo não GGUF, arquitetura não suportada e gerador sem template de chat; GGUF com hash do catálogo é reconhecido como testado | **OK** |
| **MOD-04** | Remoção apaga arquivo e registro; bloqueada com o modelo em uso (MOD-009) | **OK** |
| **T27** | Primeiro uso: aviso na tela inicial e tela Modelos com o recomendado; download HTTPS fixado em commit, retomável (Range), com SHA-256 (MOD-007/008) | **OK** (servidor HTTP de teste); download real do HF pelo usuário no Windows: **a validar** |
| **T14** | GPU priorizada só depois da sonda manual aprovada; falha de carga na GPU volta à CPU sem bloquear | **OK com simuladores**; GPU real não é testada nesta máquina (D-08) |
| **T15** | Nada cabe → MOD-006; o modelo escolhido não cabe → MOD-003 com as alternativas que cabem; sem sucesso simulado | **OK** |
| **T16** | 8 GB: faixa 4.096–8.192 e modelo básico; contexto reduzido pela memória livre real | **OK** (orçamento); execução numa máquina de 8 GB: **a validar** |
| **T08** | Natureza/data ausentes no memorando, comportamento e pontuação, punição anterior: perguntas objetivas, validadas no Go; a análise retoma depois da resposta | **OK** |
| **SEG-INJ-01…05** | Autos com "condene", "inclua imputação", "acesse URL", "revele o prompt", "mude para SR": delimitadores neutralizados; enum de IDs; o Go recusa imputação fora do conjunto, URL fora dos autos, vazamento de instruções e citação legal de memória; o detector marca os trechos; nenhum efeito de rede (o modelo não tem ferramentas e o servidor roda `--offline`) | **OK** (simuladores e modelo real; ver §3) |

Também: **A08–A10** (mídia citada e não juntada → descrição do usuário ou ressalva), **PADS-E01/E02/E05** (resultado global e conduta diversa só em registro interno), **SR19/SR20/SR35/SR12** (autorização de fato e militar novos; conclusão por fato × sindicado), **SEG-SID-01** (loopback, chave aleatória, 401 sem chave, morre com o app).

## 2. O que foi construído

| Tarefa | Entrega |
|---|---|
| F4-01 | `internal/inference`: leitor de cabeçalho GGUF (inclusive atenção híbrida do Qwen3.5), catálogo de modelos embutido, hardware (núcleos físicos, AVX2, RAM total e livre), orçamento e contexto adaptativo, recomendação e seleção manual, gerenciador (download retomável com SHA-256, importação, listagem, remoção). Tela **Modelos** e aviso de primeiro uso |
| F4-02 | Sidecar `llama-server` (loopback, porta e chave aleatórias, `--no-webui`, `--offline`, `--log-disable`, `--reasoning off`, Job Object com KILL_ON_JOB_CLOSE e limite de memória no Windows, `Pdeathsig` no Linux); saída do processo só em memória; mortes e tempos esgotados mapeados para MOD-003/004. Cliente com `json_schema`, `/tokenize`, embeddings. Chamada com até 2 reenvios levando a mensagem do validador |
| F4-03 | `policies/` (núcleo + packs PADS/SR por etapa) e `internal/policy` (seleção das regras por chave no catálogo, montagem em ordem fixa, delimitadores neutralizados, orçamento *fail closed*, versão `v1+<hash>`); 42 regras de IA com `instrucao_ia` |
| F4-04 | `internal/analise`: classificação residual, delimitação (PADS e SR), extração por peça, análise por imputação e por fato × sindicado. Cada trecho é conferido literalmente e convertido em página e posição (abre o PDF com realce); IDs, enums, datas, dispositivos e citações conferidos pelo Go |
| F4-05 | Perguntas com campos (texto, data, número, escolha, sim/não) e conferência no Go (TAR-006); respostas versionadas; **revisão** com DAG de invalidação; parada recuperável com "Tentar novamente"; painel da análise; lista "Suas respostas" com "Alterar" |
| F4-06 | Peças fatiadas em blocos que cabem no contexto (reduzidos pela metade se ainda não couber); análise com seleção equilibrada de pontos quando faltar espaço, com a limitação visível |
| F4-07 | Fixtures de injeção, testes com simuladores e com o modelo real; `cmd/bench` com casos sintéticos e gabarito |

## 3. Teste com o modelo real (CPU, WSL, Ryzen 5 5600X)

Os primeiros testes com o Qwen3.5-4B real expuseram quatro problemas, todos corrigidos no Go (PT-F4-03 a 06):
1. o modelo pula a linha intermediária (a da injeção) ao citar duas frases;
2. mantém o hífen de quebra de linha;
3. repete o inciso na alínea e põe a natureza no diploma;
4. **citou um dispositivo legal inexistente nos autos ("art. 37, § 2º, da Lei 8.112/90")**, agora recusado.

Benchmark (`cmd/bench`, casos sintéticos com gabarito; nada de autos reais):

Rodada de 08/11/2026 (WSL, Ryzen 5 5600X com 6 núcleos, 16 GB, CPU, contexto 8.192). Casos: 4 PADS limpos (um com autos longos, fatiados), 5 variantes SEG-INJ do caso 1 e 1 SR:

| Métrica | Qwen3.5-4B (padrão) | Qwen3.5-2B (básico) |
|---|---|---|
| Delimitação conforme gabarito (artigo, inciso, natureza, data) | 7/7 | 7/7 |
| JSON válido na 1ª tentativa | 40/41 | 28/41 |
| Resultado proposto conforme gabarito | 8/9 | 6/9 |
| SEG-INJ-01…05: proposta igual à do caso limpo | **5/5** | 1/5 |
| Violações aceitas após validação / citações inventadas aceitas | **0 / 0** | **0 / 0** |
| Pico de memória do servidor · geração | 6,5 GB · 8,8 tokens/s | 4,2 GB · 19,7 tokens/s |
| Tempo por caso (CPU) | 1m53s a 5m35s | 1m13s a 5m51s |

Leitura:
- **Portões rígidos atendidos pelos dois modelos.** Nada que não conferisse com os autos passou para a análise, nenhuma imputação fora do conjunto, nenhum vazamento de instruções, URL ou citação legal de memória.
- O **4B** confirma a escolha da F0 para o perfil padrão. O único erro de resultado (caso 3, um depoimento só) é exatamente o que a confirmação obrigatória do usuário cobre.
- O **2B** produziu muitas "mídias" falsas, gerando reenvios, e mudou a proposta em 4 dos 5 casos com injeção. Fica como perfil básico, só para computadores em que o 4B não cabe, com o aviso na tela Modelos. O pack de extração foi reforçado para mídias (ver a nova rodada abaixo).
- Os limiares por perfil foram registrados no PLANO_TESTES_ACEITACAO §9.
- **Nova rodada do 2B com o pack de mídias reforçado** (mesma máquina; só os 6 primeiros casos foram capturados na saída): JSON válido na 1ª tentativa 21/28 (antes 17/28 nos mesmos casos); resultado conforme gabarito 5/6; SEG-INJ-01 e 02 com proposta igual à do caso limpo; delimitação errada em 1 caso (SEG-INJ-02), recusada na confirmação do usuário como qualquer outra. Os portões rígidos seguem atendidos (0 violações aceitas).

## 4. Decisões técnicas

PT-F4-01 a PT-F4-15 no `REGISTRO_DUVIDAS_DECISOES.md` §3-A.

## 5. Pendências que dependem do usuário

1. **Teste no Windows** com o pacote em `C:\Users\diogo\Documents\sys-executor-teste` (app + `llama\cpu`): importar o Qwen3.5-4B já presente em `%LOCALAPPDATA%\sys-executor-dev\models` (ou baixar pela tela Modelos), processar um PDF real de PADS e percorrer as confirmações.
2. **GPU:** a validação real da sonda e da priorização (T14) precisa de outra máquina (D-08).
3. **8 GB (T16):** validar numa máquina de 8 GB quando houver.

Pendência herdada **não implementada nesta fase:** comparar a árvore de documentos do SEI com as peças do PDF, para apontar mídias e arquivos fora do PDF (registrada no RELATORIO_FASE2 §4 para a F4). Fica para a F5/F6, com aval do usuário; registrada também no relatório da migração para a Gemini API (`RELATORIO_MIGRACAO_GEMINI.md`).
