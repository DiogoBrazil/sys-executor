# Registro de revisões da V4 e do catálogo de regras

Toda mudança em `docs/rules/sys-executor_especificacao_consolidada_V4.md` exige, **antes** de atualizar `policies/V4.lock`:
1. revisão humana do impacto no catálogo de regras e nos testes;
2. uma entrada neste arquivo (data, o que mudou na V4, regras afetadas, quem aprovou);
3. `go run ./cmd/rulescov -atualizar-lock`.

O CI falha enquanto o SHA-256 da V4 divergir do lock (GOV-LOCK-01).

## Entradas

### 08/11/2026 — Linha de base
- V4 fixada com SHA-256 `6f213f366a39f3fdd795d70dc094b79ee7ce11f0e486bc977c4b7ddc416f024f` (Fase 1).
- Nenhuma alteração de regra. Decisões complementares D-01…D-09 em `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`.
- Aprovação: usuário (autorização da Fase 1).

### 09/11/2026 — Emenda 1 (seção 47): inferência pela Gemini API
- **O que mudou na V4:** nova seção 47 e notas **[Emenda §47]** no cabeçalho, em §29.3 e em §§34–38 (texto original mantido). A inferência por modelos locais (GGUF/llama-server) foi substituída pela Gemini API com chave do usuário; envio só do necessário, confirmação em cada tarefa, pseudonimização de dados pessoais, `store=false`, sem ferramentas; chave em DPAPI; sem internet → parada recuperável.
- **Regras jurídicas (§§1–33 e 39–46):** nenhuma alteração.
- **Regras do catálogo afetadas:** revogadas TRV-IA-001, TRV-IA-003…009, TRV-SEG-006, TRV-SEG-007 e TRV-OFF-001; reescritas TRV-OFF-002 e TRV-INS-003; novas TRV-IA-011…013, TRV-SEG-009…012 e TRV-OFF-003 (entram com o código que as garante).
- **Testes:** T14, T15, T16, T18 e T27 passam a ser cobertos pelas novas regras, no sentido da seção 47; SEG-NET-01 redefinido (só o host da Gemini API, só após a confirmação); SEG-SID-01 e MOD-01…04 obsoletos.
- **Registro:** D-10 em `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`.
- **Aprovação:** usuário, em 09/11/2026 ("Aprovo o texto").


### 09/11/2026 — Emenda 2 (seção 48): ChatGPT, cadastro de peças, enquadramento manual e barreira de privacidade
- **O que mudou na V4:**
  - nova seção 48;
  - nota no cabeçalho;
  - marcas **[Emenda §48]** em §§2.1, 2.2, 16.1, 17.4, 22.1, 29.3, 30.1, 30.2, 45, 47.1 e 47.3, com o texto original mantido.
- **Conteúdo da emenda:**
  - ChatGPT (*Sign in with ChatGPT*, plano do usuário) como 2º provedor, à escolha explícita e sem troca automática;
  - autorização de envio vinculada ao provedor e ao modelo;
  - entrada por peças cadastradas (upload ou só os documentos selecionados no SEI), com o conteúdo conferido contra a declaração;
  - enquadramento do PADS informado pelo usuário a partir de catálogo curado (RDPM arts. 15–17 com natureza fixa; Estatuto arts. 29 e 32 com natureza atribuída pelo usuário); cada dispositivo é uma imputação, e a conexão segue a seção 5;
  - barreira obrigatória de privacidade antes do envio, com revisão item a item;
  - bloqueio do ChatGPT quando há dados de criança menor de 13 anos;
  - aviso de privacidade antes do primeiro login e uso de marca restrito, ambos por exigência dos Termos do SIWC.
- **Regras jurídicas:** nenhuma alteração de conexão, dosimetria, reincidência, comportamento, pontuação, regressão, resultados do PADS, vinculação ao Memorando (§22) ou fatos novos na SR (§24.3).
- **Regras do catálogo afetadas** (lista prevista em `docs/planning/PROPOSTA_EMENDA2_V4.md`):
  - serão reescritas, revogadas ou criadas **junto com o código que as garante**, em cada etapa do plano `docs/planning/PLANO_CHATGPT_PECAS_PRIVACIDADE.md`, como na Emenda 1;
  - nesta entrada o catálogo não muda.
- **Testes:** T02, T03 e T04–T06 continuam mapeados às regras atuais até a etapa do cadastro de peças, quando passam a ser verificados no sentido da seção 48.2.
- **Registro:** D-11 em `docs/planning/REGISTRO_DUVIDAS_DECISOES.md`.
- **Aprovação:** usuário, em 09/11/2026 ("Tudo certo"). Os acréscimos sobre crianças, aviso de privacidade e marca decorrem dos Termos do SIWC fornecidos pelo usuário; a opção "pergunta + bloqueio" foi escolhida por ele.

### 09/11/2026 — Catálogo: barreira de privacidade (etapa 2 do plano D-11; V4 §48.6)
- **V4:** sem alteração (o lock não muda).
- **TRV-SEG-011 v2:** a ocultação passa a não diferenciar caixa nem acentos e a aceitar quebra de linha entre as palavras, inclusive escapada em JSON. Nome de uma só palavra só é trocado com inicial maiúscula, para "machado", o objeto, não virar marcador. Marcadores deformados são recusados para todos os tipos.
- **Nova TRV-SEG-013:** barreira obrigatória antes de cada envio.
  - Camada 1, no pedido: resíduo de dado conhecido bloqueia com IA-014.
  - Revisão item a item de possíveis nomes sem âncora, feita na autorização e no meio da etapa, sem confirmação genérica.
  - Camada 2, nos bytes da requisição, pelo transporte restrito.
- **Novo código de erro:** IA-014.
- **Testes:** SEG-PSEUDO-01 ampliado; novo SEG-BAR-01.
- **Golden dos prompts:** sem alteração.
- **Aprovação:** usuário, por meio do plano D-11 aprovado (etapa 2 autorizada em 09/11/2026).

### 09/11/2026 — Catálogo: mais de um provedor (etapa 3 do plano D-11; V4 §48.1)
- **V4:** sem alteração; o lock não muda.
- **TRV-SEG-010 v2:** a confirmação de envio vale para **um provedor e um modelo**. Se um dos dois mudar depois da autorização, o motor anterior é fechado e a pergunta é feita de novo antes de qualquer envio.
- **Nova TRV-IA-014:** dois provedores, escolhidos explicitamente. O motor abre o alvo autorizado e não há troca automática.
- **Erros:** textos de IA-003 a IA-011 neutros quanto ao provedor; novos IA-015 a IA-020, do ChatGPT. Nenhum código foi reutilizado.
- **Preferências v3:** `provedor_ia` e `modelo_chatgpt`, com migração da v2.
- **Credencial:** loja nomeada, com arquivo e entropia próprios.
- **Testes:** novos SEG-ENV-02 e IA-PROV-01.
- **Golden:** sem alteração.
- **Aprovação:** usuário, por meio do plano D-11 aprovado (etapa 3 autorizada em 09/11/2026).

### 09/11/2026 — Catálogo: provedor ChatGPT (etapa 4 do plano D-11; V4 §§48.3, 48.6, 48.7)
- **V4:** sem alteração; o lock não muda.
- **TRV-SEG-012 v2:** destinos de origem única dos dois provedores. ChatGPT só em `api.openai.com` e `auth.openai.com`, nunca em endpoints privados.
- **Novas regras:**
  - TRV-SEG-014: credencial do ChatGPT;
  - TRV-SEG-015: login oficial, permissão do plano, aviso de privacidade e marca;
  - TRV-SEG-016: dados de criança bloqueiam o ChatGPT;
  - TRV-IA-015: Responses API.
- **Erro novo:** IA-021.
- **Testes:** SEG-SIWC-01/02, SEG-CRI-01, IA-SIWC-01.
- **Dependências novas:** `github.com/coreos/go-oidc/v3` e `github.com/go-jose/go-jose/v4` (Apache-2.0); `golang.org/x/oauth2` (BSD-3-Clause, indireta).
- **Golden:** sem alteração.
- **Aprovação:** usuário, por meio do plano D-11 aprovado (etapa 4 autorizada em 09/11/2026).

### 09/11/2026 — Catálogo de enquadramento (etapa 5 do plano D-11; V4 §48.5)
- **V4:** sem alteração; o lock não muda.
- **Novo `internal/rules/enquadramento/dispositivos.yaml`:** 130 dispositivos na redação vigente, com arquivo e página de origem e a norma que deu a redação.
  - RDPM: art. 15 com 17 incisos (natureza leve), art. 16 com 44 (média) e art. 17 com 39 (grave).
  - Estatuto (DL 09-A/1982): art. 29 com 18 incisos mais as 5 alíneas do XVIII, e art. 32 com 7 incisos, sem natureza.
  - Extraído dos PDFs de `docs/law_library`. **Revisão humana do usuário: pendente.**
- **PADS-ESC-001 v2:** o enquadramento vem do usuário. **Nova PADS-ENQ-001.**
- **Formulário das imputações:**
  - diploma, depois dispositivo com filtro (campo `BUSCA`), depois natureza só no Estatuto;
  - divergência com o citado no Memorando exige confirmação com a §22.1;
  - a visibilidade dos campos agora é encadeada, no Go e na interface.
- **Golden:** sem alteração.
- **Aprovação:** usuário, por meio do plano D-11 aprovado (etapas 5–8 autorizadas em 09/11/2026); revisão da curadoria pendente.

### 09/11/2026 — Catálogo: cadastro de peças (etapa 6 do plano D-11; V4 §48.4)
- **V4:** sem alteração; o lock não muda.
- **Nova TRV-ING-007:** cadastro peça por peça (PADS e SR), com vários arquivos por peça, ordem e origem preservadas, cada oitiva uma peça, arquivo repetido apontado, junção em memória na ordem processual, só na sessão; os dados do usuário são cruzados com a leitura da peça central e não ampliam as acusações nem o objeto.
- **TRV-ING-001 v2:** a função declarada no cadastro vale; a classificação pelo conteúdo e pela cronologia é conferência (a `instrucao_ia` não muda).
- **TRV-ING-002 v2:** conteúdo que contradiz a declaração gera pergunta, nunca troca.
- **TRV-PRV-005 v2:** a origem operacional fica registrada por arquivo do cadastro.
- **TRV-UI-001 a TRV-UI-004 v2:** "upload" passa a ser o cadastro de peças; a compatibilidade parte da peça central declarada, com o conteúdo conferido.
- **Erros novos:** ARQ-004 (arquivo já anexado) e ARQ-005 (limite do cadastro).
- **Testes:** CAD-PEC-01, CAD-PEC-02 e CAD-PEC-03; T01–T04 e T06 passam a ser verificados no sentido da seção 48.2.
- **Golden:** sem alteração.
- **Aprovação:** usuário, por meio do plano D-11 aprovado (etapas 5–8 autorizadas em 09/11/2026).
