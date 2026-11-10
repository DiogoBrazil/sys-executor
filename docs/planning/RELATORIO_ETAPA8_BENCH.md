# Relatório — etapa 8 do plano D-11: bench, regressão e documentação

- **Data:** 10/11/2026. **Branch:** `add-mode-apikey-oauth`.
- **Autorização:** etapas 5 a 8 já autorizadas ("Pode prosseguir com as demais etapas").
- **Decisões do usuário nesta etapa:**
  - o bench tem **login próprio no ChatGPT**, com a credencial de desenvolvimento em arquivo 0600 no WSL;
  - a conta do Gemini passou para o **nível pago** (pré-pago, com os créditos do GCP);
  - foi autorizada a ampliação do `poc/sei-mapa` para a etapa 7.
- **Dados:** só casos sintéticos de `internal/testutil/casos`.

## 1. O que mudou

### `cmd/bench`
| Arquivo | Conteúdo |
|---|---|
| `main.go` | Opções:<br>• `-provedor gemini\|chatgpt\|falso`;<br>• `-entrada memoria,monolitica,pecas`;<br>• `-max-chamadas N`, teto da rodada que para de forma limpa e marca "(teto)";<br>• `-modelos`;<br>• `-conectar-chatgpt`, `-desconectar-chatgpt` e `-estado-chatgpt`;<br>• `-amostras N`, que mostra no terminal as respostas recusadas pelo validador.<br>Relatório com:<br>• colunas "Entrada", "Peças × gabarito" e "Chamadas";<br>• resultado obtido quando difere do gabarito;<br>• comparação das entradas;<br>• métrica de vazamento;<br>• revisões da barreira;<br>• divergências da separação.<br>Código de saída **3** quando o portão de vazamento reprova. |
| `provedores.go` | **Gemini:** a chave vem de `SYSEXEC_GEMINI_API_KEY`.<br>**ChatGPT:** usa a **mesma** `chatgpt.Conta` e o mesmo `inference.MotorChatGPT` do app, com a loja `lojaDev` (`~/.config/sys-executor/chatgpt-dev.json`, 0600, pasta 0700, gravação atômica; o estado é mostrado sem token nem e-mail).<br>**Falso:** monta um corpo JSON com todos os campos do pedido, passa-o pelo mesmo gancho de inspeção e responde com um eco do primeiro documento da mensagem. O eco não segue o schema, então o executor reenvia com o histórico **restaurado**, o que cobre o caminho mais sujeito a vazamento. |
| `vazamento.go` | Confere o **corpo exato** de cada requisição, depois da barreira, contra `casos.Caso.Pessoas`/`Identificadores`, sem caixa e sem acento, com limite de palavra.<br>**Formas procuradas:** nome completo, pares de termos consecutivos sem conectores e matrícula.<br>**Termos soltos ficam fora:** "suposto" e "inventado" são palavras comuns, e "inventado" aparece nas próprias políticas. |
| `entrada.go` | **memoria:** gabarito, sem PDF.<br>**monolitica:** PDF sintético com o carimbo do SEI em cada página, lido pelo pdfium e separado por `segment.Segmentar`.<br>**pecas:** um PDF por peça, juntados pelo `pdf.Juntador` e montados por `segment.DeDeclaracao`; os envolvidos semeiam o mapa (`CadastrarPessoa`), como no cadastro.<br>Os PDFs ficam só em memória. O pdfium é aquecido antes de medir. |
| `bench_test.go` | • formas procuradas;<br>• limite de palavra;<br>• corpo com JSON escapado;<br>• **vazamento zero nas três entradas** (provedor falso, sem rede);<br>• **vazamento detectado sem pseudonimização**;<br>• teto de chamadas;<br>• permissões da loja de desenvolvimento. |

- **Revisão da barreira no bench:** diante de `ErrRevisao`, o bench toma a decisão mais protetiva (ocultar todos os itens) e refaz o pedido, como o app refaz a etapa depois da resposta do usuário. Os itens aparecem no relatório.

### Outros arquivos
- `internal/testutil/casos`: campos `Pessoas` e `Identificadores` em cada caso, com os nomes e a matrícula fictícios.
- **CI:** `./cmd/bench/...` entrou no `vet` e nos testes (Linux com `-race` e Windows). No Windows, o teste de permissões POSIX é pulado.
- **Documentação:**
  - `ADR_010_chatgpt_pecas_barreira.md` (novo);
  - `README.md`: dois provedores, cadastro de peças, barreira, bench e dados em disco;
  - `PLANO_TESTES_ACEITACAO.md` §9: BEN-01/02 nos dois provedores, BEN-VAZ e BEN-ENT.
- **`poc/sei-mapa`** (etapa 7, preparação):
  - comando `mapear`, com a forma da tela "Gerar Arquivo PDF do Processo";
  - `[post-gerar]`, com os nomes e a forma dos campos do POST da geração;
  - teste num Chrome headless sem GPU, contra uma página fictícia, conferindo que nada vaza;
  - roteiro no `LEIA-ME.md`.

## 2. Resultados

### 2.1 Sem rede (provedor falso, três entradas, 10 casos)
- **252 pedidos conferidos, 0 com nome dos casos.** Sem pseudonimização, a métrica aponta os nomes, o que confirma que ela detecta o vazamento.
- **Separação das peças:** 30/30 na monolítica e 30/30 na por peças.
- **Preparação:** cerca de 20 a 30 ms por caso, depois de aquecido o pdfium.
- **Correção de um achado meu:** a primeira versão do eco copiava o início da mensagem ("DADOS ESTRUTURADOS:…"). No histórico, isso virou uma linha JSON com minúsculas, e a barreira pediu revisão desse cabeçalho. Nenhum modelo real faz isso, e o app não é afetado. O eco passou a citar a evidência, e nenhuma revisão aparece. **A barreira não foi alterada.**

### 2.2 Gemini — `gemini-3.6-flash`, nível pago, entradas monolítica e por peças
| Métrica | Resultado | Limiar (§9) |
|---|---|---|
| JSON válido na 1ª tentativa | **75/75 (100%)** | ≥ 90% ✅ |
| Delimitação conforme gabarito | **20/20** | ≥ 90% ✅ |
| Resultado conforme gabarito | **17/18 (94%)** | ≥ 75% ✅ |
| SEG-INJ igual ao caso limpo | **10/10** (5 por entrada) | 5/5 ✅ |
| Violações aceitas / citações inventadas | 0 / 0 | 0 ✅ |
| Vazamento (corpo exato) | **0 em 75 pedidos** | 0 ✅ |
| Chamadas, tokens e tempo | 75 chamadas, 0 falhas; entrada 112 mil, saída 20 mil, raciocínio 127 mil; 12 min 52 s | — |

- **O 400 da análise está confirmado como corrigido.** As 75 chamadas, incluindo todas as análises, não tiveram nenhuma falha. A pendência da etapa 2 (§3-A) está encerrada.
- **Única divergência:** pads-02, IMP-2 (gabarito IMPROCEDENTE), saiu PROCEDENTE. Repeti o pads-02 duas vezes nas duas entradas. Somadas as três rodadas, a IMP-2 errou 2 de 3 vezes na monolítica e 1 de 3 na por peças. É **variação do modelo** nesse caso, não efeito da entrada.

### 2.3 ChatGPT — `gpt-6-luna`, plano Plus, raciocínio padrão (medium), entradas monolítica e por peças
| Métrica | Resultado | Limiar (§9) |
|---|---|---|
| JSON válido na 1ª tentativa | **56/76 (74%)** | ≥ 90% ❌ |
| Delimitação conforme gabarito | **20/20** | ≥ 90% ✅ |
| Resultado conforme gabarito | **12/18 (67%)** | ≥ 75% ❌ |
| SEG-INJ igual ao caso limpo | 2/10, mas ver a nota abaixo | 5/5 ⚠️ |
| Violações aceitas / citações inventadas | 0 / 0 | 0 ✅ |
| Vazamento (corpo exato) | **0 em 104 pedidos** | 0 ✅ |
| Chamadas, tokens e tempo | 104 chamadas, 0 falhas; entrada 196 mil, saída 44 mil, raciocínio 14 mil; 18 min 22 s | — |

- **SEG-INJ:** a comparação é feita contra o caso limpo, e o próprio pads-01 limpo errou nas duas entradas. A rodada principal ainda não mostrava o resultado obtido; numa repetição, saiu IMPROCEDENTE no lugar de PROCEDENTE. As variantes com injeção acertaram em 8 de 10. Não há sinal de que a injeção tenha funcionado: o desvio está no caso limpo. Mesmo assim, o critério 5/5 não está atendido.
- **Causa principal das recusas na 1ª tentativa:** na extração, a mensagem "a pessoa N não tem trecho literal no documento" aparece em quase todos os casos.
  - Com `-amostras`, a resposta recusada mostra o motivo. Na Defesa Prévia, que não cita ninguém pelo nome, o modelo lista "pessoas" **sem nome** (papel ACUSADO ou DEFENSOR), com um trecho que **está** no documento.
  - O validador (`internal/analise/extracao.go:223`) recusa o nome vazio **com a mensagem do trecho**. O modelo então corrige o trecho, nunca o nome, e repete o erro nos reenvios.
  - O mesmo acontece **sem pseudonimização** (pads-01, 3/4 nas duas formas). Os marcadores não são a causa.
- **Resultado:** varia entre rodadas no mesmo caso (pads-01 por peças: PROCEDENTE numa rodada, IMPROCEDENTE em outra).
- **Outra recusa:** na análise, um fundamento passou dos 700 caracteres. O json_schema strict aceita o `maxLength`, mas não o impõe; quem impõe é o validador em Go.

### 2.4 Entrada monolítica × por peças
| Provedor | Peças × gabarito | Delimitação | JSON 1ª | Resultado |
|---|---|---|---|---|
| Gemini, monolítica | 30/30 | 10/10 | 38/38 | 9/9 |
| Gemini, por peças | 30/30 | 10/10 | 37/37 | 8/9 |
| ChatGPT, monolítica | 30/30 | 10/10 | 28/38 | 6/9 |
| ChatGPT, por peças | 30/30 | 10/10 | 28/38 | 6/9 |

**Conclusão:** nos casos sintéticos, as duas entradas são **equivalentes** em qualidade, e a preparação leva dezenas de milissegundos nas duas. As diferenças de resultado estão dentro da variação do modelo (§2.2). O ganho da entrada por peças está no que o gabarito não mede:
- a função vem declarada pelo usuário, sem depender do carimbo;
- os envolvidos cadastrados semeiam a pseudonimização;
- o tipo e as pessoas são conferidos com o conteúdo.

Os casos sintéticos têm carimbo correto e títulos claros, que é o cenário favorável à monolítica.

## 3. Verificação
| Verificação | Resultado |
|---|---|
| `gofmt -l ./internal ./cmd` | Vazio |
| `GOOS=windows go vet ./...` e `go vet` (Linux, com `cmd/bench`) | Verde |
| `go test ./internal/... ./cmd/bench/...` | Verde (`-race` em `cmd/bench` e `internal/testutil`) |
| `poc/sei-mapa`: `go test .` e build para Windows | Verde |
| `rulescov -estrito` | 201 regras; 124/124 IDs da V4 |
| `cmd/licencas` | 45 módulos Go e 40 npm |
| Golden (`internal/policy/testdata`) | **Sem diff** |
| Vitest e build do frontend | Sem mudança no frontend nesta etapa (30/30 na preparação da máquina) |

## 4. Pendências e propostas (decisão do usuário)
1. **Mensagem do validador para pessoa sem nome** (`analise/extracao.go:223`). Proposta: separar as duas falhas, por exemplo "a pessoa N está sem nome; liste só pessoas nomeadas no documento".
   - É a causa provável da maior parte do JSON 1ª abaixo do limiar no ChatGPT.
   - Não muda as políticas, e os golden continuam iguais, mas muda o validador. **Exige autorização.**
2. **ChatGPT abaixo do limiar** em JSON 1ª e no resultado. Depois do item 1, repetir a bateria e, se preciso, testar `-pensamento high` ou outro modelo da conta (há 8). O app não fixa modelo, e a escolha continua do usuário.
3. **Variação do resultado entre rodadas** (pads-02 IMP-2 no Gemini; pads-01 no ChatGPT). Uma rodada por caso não basta para afirmar qualidade. Proposta: repetir cada caso 3 vezes e reportar a mediana, quando houver cota.
4. **Etapa 7:** aguarda o mapeamento do usuário com o `sei-mapa` ampliado (roteiro no `LEIA-ME.md`).
5. **Nível pago do Gemini:** pela documentação, prompts e respostas não são usados para melhorar os produtos. O README já registrava a diferença entre os níveis. A decisão D-10.1 ("qualquer nível, sem aviso") continua valendo.
