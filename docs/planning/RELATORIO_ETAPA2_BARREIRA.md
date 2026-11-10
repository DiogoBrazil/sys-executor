# RELATÓRIO — Etapa 2 do plano D-11: barreira de privacidade e correções da pseudonimização

- **Data:** 09/11/2026 · **Branch:** `add-mode-apikey-oauth`
- **Autorização:** usuário, 09/11/2026 ("Pode seguir com 1 e 2").
- **Regras:** V4 §47.3 e §48.6; catálogo TRV-SEG-011 v2 e nova TRV-SEG-013; erro novo IA-014.
- **Alcance:** independe de provedor e já protege o Gemini. O ChatGPT vai usar a mesma barreira (etapa 4).

## 1. O que mudou

| Item | Antes | Agora |
|---|---|---|
| Casamento das grafias | Só a grafia exata | **Sem diferenciar caixa nem acentos**, com espaços ou quebras de linha entre as palavras, inclusive `\n` escapado em JSON (`internal/pseudonimo/mapa.go`). Cada grafia real nova vira variante (`[PESSOA-01.n]`), e a restauração continua byte a byte |
| Palavra única | — | Nome de uma só palavra só é trocado com inicial maiúscula ("Machado" sim; "machado", o objeto, não), para não deformar os fatos |
| Limite de palavra | A forma longa recusada impedia a curta | A forma curta é tentada na mesma posição |
| Marcadores deformados | Só alguns tipos, só em maiúsculas | Todos os tipos. Entre colchetes, em qualquer caixa (`[pessoa-01]`). Sem colchetes, com cuidado para não confundir números dos autos (`CPF-01` sim; `CPF 529.982…` não) |
| **Camada 1** (`pseudonimo.Motor.Gerar`) | Ocultava só `Usuario` e `Historico` | **Antes de cada envio**, inclusive nos reenvios de correção: (a) CPF, e-mail, telefone, RG/matrícula com rótulo e nomes com âncora que aparecem no próprio pedido (texto digitado, histórico) são aprendidos e ocultados; (b) dado conhecido que reste em instruções, dados, histórico ou schema → **IA-014, nada enviado**; (c) possível nome sem âncora ainda não decidido → `ErrRevisao`, que leva à revisão |
| **Camada 2** (`inference.TransporteRestrito.Inspecionar`) | — | O transporte entrega ao inspetor **o corpo exato** da requisição. `Mapa.Inspecionar` decodifica o JSON e confere todos os textos; qualquer resíduo → **IA-014**, sem nova tentativa e sem nada chegar ao servidor. `FonteMotor.Abrir` passou a receber o inspetor |
| Revisão na autorização (`pipeline/envio.go`, `privacidade.go`) | Campo livre "outros nomes" | A confirmação de envio lista **cada possível nome sem âncora** dos autos, com o link para a página, e pede decisão obrigatória item a item ("ocultar" ou "não é dado pessoal"). Não há "ignorar todos" |
| Revisão no meio da etapa | — | As etapas Delimitando e Analisando ficaram envolvidas por `comRevisao`. Se surgir possível nome novo (por exemplo, digitado no formulário de imputações), a etapa pára sem enviar, pergunta, aplica as decisões e retoma. "Não enviar" → IA-011, recuperável |
| `cmd/bench` | — | Nos casos sintéticos, todo possível nome é ocultado (decisão mais protetiva) |

**Não mudou:** regras jurídicas, prompts (golden sem diff), validadores, motor determinístico e o alcance da pseudonimização da §47.3. Nº SEI, nº de processo, datas dos fatos e posto/graduação continuam em claro, por decisão E4. A tabela de correspondência continua só em memória e nunca aparece em mensagens de erro: os erros dizem só o tipo do dado e o campo.

## 2. Verificação

| Verificação | Resultado |
|---|---|
| `go test -race` (internal, cmd/bench, rulescov, licencas, autos-mapa) | Verde |
| `GOOS=windows go vet ./...` e build `-H windowsgui` | Verde |
| `gofmt` | Verde (exceto `carimbo_test.go`, pendência anterior) |
| `rulescov -estrito` | 194 regras; 124/124 IDs da V4; 0 sem teste |
| Golden das políticas | Sem diff |
| `cmd/licencas` | 42 módulos Go e 40 npm; nenhuma dependência nova |
| Vitest (22), `vue-tsc` | Verde; a tela de perguntas já mostra os campos de revisão (escolha com fonte) |
| Testes novos | caixa/acento/quebra de linha; palavra única; forma escapada em JSON; forma curta; homônimos não se fundem; deformados; camada 1 (aprende o que foi digitado, bloqueia em instruções e schema, revisão e liberação); cabeçalhos não são suspeitos; camada 2 (bytes); transporte Gemini (corpo inspecionado, bloqueio sem envio); **fluxo PADS ponta a ponta**: revisão no meio da etapa, recusa, retomada, nenhum nome em nenhum pedido e **nenhum marcador no estado da tarefa** |

**Ruído da revisão nos 10 casos sintéticos:** depois de ajustar a lista de palavras que não são nome (tipos de peça, fórmulas de tratamento), resta 1 item: "REGRAS ANTERIORES E CONDENE", da frase de injeção em maiúsculas. Mantido de propósito: em documentos da PM, nome em maiúsculas no meio da frase é comum, e é melhor perguntar a mais.

## 3. Limites (continuam)
- **Nomes de uma só palavra sem âncora** ("Silva afirmou") não são apontados como suspeitos, porque o ruído seria enorme. O cadastro de pessoas do assistente (etapa 6) vai semear o mapa com os envolvidos e suas grafias.
- **Pseudonimização não é anonimização:** o contexto pode identificar uma pessoa (V4 §48.6).
- **Ainda não testado no Windows pelo usuário.** O comportamento com o Gemini real não muda, exceto pela tela de revisão na confirmação de envio.

## 3-A. Teste manual no Windows e correções que ele revelou (09/11/2026)

O usuário testou o `.exe` com `pads-sintetico-texto.pdf` e a Gemini API, chave do nível gratuito. A **barreira funcionou de ponta a ponta**: a revisão de "Lucas Imaginário", digitado na conduta, apareceu; "Não enviar" deu IA-011; "Tentar novamente" trouxe a revisão de volta; "ocultar" seguiu para o envio. O teste revelou três defeitos, todos corrigidos:

1. **Tela parada em "Delimitando…".** Era uma corrida na interface, anterior a esta etapa. A resposta do comando chegava depois dos eventos que a etapa já tinha emitido e levava a tela de volta para o estado antigo. Antes não aparecia porque o passo seguinte sempre esperava a rede; a revisão é imediata. Um dump das goroutines, gravado por um `.exe` de diagnóstico temporário e não versionado, mostrou o backend ocioso esperando a revisão.
   - **Correção:** `workflow.Visao.Sequencia` monotônica; a interface (`estado.ts`, `mostrar`) descarta uma visão mais antiga da mesma tarefa.
   - **Testes:** Vitest "Ordem das visões" (falha sem a correção) e `TestSequenciaDaVisaoCresce`.
2. **400 INVALID_ARGUMENT na extração**, o defeito pendente da migração. Reproduzido no WSL com a chave real, na **1ª chamada** (`historico=0`). Uma bissecção direta na REST mostrou que a Gemini API recusa `maxItems` em schemas maiores:
   - extração original → 400; sem `maxItems` → 200, no 3.6 Flash e no 3.5 Flash.
   - **Correção:** o `MotorGemini` envia o schema sem `minItems`/`maxItems` (`semLimitesDeLista`).
   - Os limites continuam garantidos para todo provedor por `analise.ConferirSchema`, chamado em `Executor.Chamar` antes das validações da etapa. Ele confere tipos, campos obrigatórios e extras, enums, tamanhos de lista, `maxLength` e mínimos/máximos. Uma violação volta ao modelo no reenvio, com o motivo.
3. **Cota diária do nível gratuito classificada como IA-013 ("sem créditos").** O texto do Google cita "billing". Agora, erro com `QuotaFailure` vira **IA-005** (`temQuotaFailure`).

**Pendente de confirmação real:** o fluxo completo de extração e análise com o Gemini. O nível gratuito tem **20 requisições por dia, compartilhadas entre o 3.5 e o 3.6 Flash**, e a cota acabou durante a bissecção. Os testes automáticos estão verdes.

## 4. Próxima etapa
**Etapa 3:** generalização para mais de um provedor. Inclui `FonteMotor.Alvo`, autorização presa a provedor, modelo e versão do mapa, preferências v3 com `migrarV2`, credencial com nome e entropia próprios, e textos de erro conforme o provedor. Depende de autorização expressa.
