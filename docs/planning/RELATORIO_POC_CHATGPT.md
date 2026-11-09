# RELATÓRIO — PoC Sign in with ChatGPT (etapa 1 do plano D-11)

- **Data:** 09/11/2026
- **Branch:** `add-mode-apikey-oauth`
- **Código:** `poc/chatgpt/`, módulo Go isolado. Não toca no app.
- **Conta:** ChatGPT Plus do usuário, com login feito por ele no navegador do Windows. A PoC rodou no WSL e o retorno chegou por `127.0.0.1`.
- **Dados:** **somente sintéticos.** Os 8 schemas reais do app foram gerados de `internal/analise` por `go test -overlay`, sem alterar o repositório. Eles contêm só IDs fictícios (`EV-nnnn`, `IMP-n`).
- **Sessão:** ao final, a sessão foi **revogada** e a credencial local apagada.

## 1. Resultado por item

| # | Item | Resultado |
|---|---|---|
| 1 | Registro dinâmico (`dynamic_agent_client`, `agent_name_hint=sys-executor`) | ✅ O retorno trouxe `client_id=oaiapp_…` (registro guardado só localmente) |
| 2 | Authorization Code + PKCE S256, `state`, `nonce`, retorno em `127.0.0.1:<porta aleatória>` | ✅ Funcionou do WSL: o navegador do Windows alcança o loopback do WSL, com o `Host` conferido |
| 3 | ID token | ✅ RS256 verificado pelo JWKS (`auth.openai.com/.well-known/jwks.json`) com `go-oidc/v3` v3.21.0: `iss`, `aud` (com o `client_id`), `exp` e `nonce` conferidos. Claims: `at_hash aud auth_time email email_verified exp family_name given_name https://api.openai.com/auth iat iss jti name nonce sid sub`. **Não há claim de plano nem de workspace** |
| 4 | Escopos concedidos | ✅ `chatgpt.tokens.use.direct email offline_access openid profile resource.invoke` (vieram no retorno e na resposta de token) |
| 5 | Access token | É um JWT RS256 com `aud=https://api.openai.com/v1`, `client_id`, `scope` e `exp`. Vale 3600 s. A resposta de token traz `earliest_refresh_at` (≈ 54 min) |
| 6 | Renovação | ✅ **Rotação a cada uso.** A renovação feita **antes** de `earliest_refresh_at` também foi aceita: o campo é uma indicação, não um bloqueio |
| 7 | Reuso do refresh antigo logo após a rotação | ⚠️ **Aceito**, e o refresh atual continuou válido. Isso indica uma janela de tolerância. A duração não foi medida |
| 8 | Relogin com o `client_id` salvo, `id_token_hint`, `login_hint` e **outra porta** | ✅ Aceito. O retorno **não** trouxe `client_id`, como a doc prevê. Mesmo `sub` |
| 9 | `GET /v1/models` | ✅ `{models:[…]}` com 10 itens, 8 com `visibility=list`. Os campos úteis estão em §2.3 |
| 10 | `POST /v1/responses` (SSE) | ✅ Termina em `response.completed`. O primeiro byte chega em 1,2–2,3 s; chamadas curtas levam 2–5 s |
| 11 | Histórico de reenvio (user → assistant `output_text` → user) | ✅ Aceito; o formato do reenvio do validador funciona |
| 12 | `reasoning: {effort: "low"}` | ✅ Aceito |
| 13 | `text.format: json_object` | ✅ Aceito |
| 14 | **`text.format: json_schema` strict com os 8 schemas reais** | ✅ **Todos aceitos na forma original**, inclusive `maxLength`, `minItems`/`maxItems`, `minimum`/`maximum`, `enum:[""]` e **600 valores de enum**. **0 violações** no validador local |
| 15 | Recusas esperadas | `role: system` → 400 `{"detail":"System messages are not allowed"}`; `temperature` → 400 `{"detail":"Unsupported parameter: temperature"}`. **`max_output_tokens` foi ACEITO**, ao contrário da documentação, que manda omitir |
| 16 | Revogação (`/api/accounts/oauth/revoke`) | ✅ HTTP 200 sem corpo. Depois dela, o refresh dá 400 `invalid_grant` ("A fresh sign-in is required") e o **access token também é recusado na hora** (401 `token_revoked`) |

## 2. Achados que mudam ou detalham o plano

### 2.1 Saída estruturada
- **Estratégia (1) confirmada:** json_schema strict **sem transformar** os schemas de `analise/schema.go`.
- **Dispensados:**
  - o plano B (JSON pedido no prompt);
  - a *function tool* (pendência 8.3 do plano), sem necessidade de rever a §47.3.
- **Custo de tokens:**
  - **o schema entra na contagem de tokens de entrada**: o de 600 enums somou cerca de 15 mil tokens de entrada;
  - enums grandes de `EV-nnnn` consomem a cota do plano (limite de 5 h do Plus);
  - no bench (etapa 8), medir o custo e avaliar fatiar evidências.

### 2.2 Erros
- **Antes do fluxo**, os erros podem vir em dois formatos:
  - `{"detail": "<texto>"}`, sem `code` (400 de parâmetro, por exemplo);
  - `{"error": {"code": …}}` (por exemplo, 401 `token_revoked`).
- **Classificação:**
  - feita por **status HTTP + `code`, quando houver**;
  - `detail` serve só como diagnóstico, como a doc recomenda;
  - um 400 sem `code` vira INT-001 (defeito do app), sem nova tentativa.
- **ID da requisição:** fica em `x-oai-request-id`; algumas respostas trazem `x-request-id`. Ele é guardado para o diagnóstico.
- **Novo erro observado:** `token_revoked` (401). Vira "sessão expirada/revogada" e pede para conectar de novo. A credencial só é apagada quando a revogação está confirmada.

### 2.3 Fluxo SSE e modelos
- **Fluxo SSE:**
  - chega **sem `Content-Type`**; o cliente do app não pode exigir esse cabeçalho;
  - eventos observados: `response.created`, `in_progress`, `metadata`, `output_item.added/done`, `content_part.added/done`, `output_text.delta/done`, `completed`;
  - o texto dos deltas é igual ao texto montado a partir de `response.completed`, nas 15 respostas.
- **Modelos:** cada item traz, entre outros, `slug`, `display_name`, `visibility`, `context_window` (272 mil a 500 mil), `max_context_window`, `supported_in_api`, `input_modalities` (`text`, `image`), `supported_reasoning_levels` (`low`…`ultra`, varia por modelo) e `default_reasoning_level` (`low` ou `medium`).
- **Filtro do app:** `visibility == "list"` e `supported_in_api == true`, na ordem do servidor.
  - O nível de raciocínio oferecido vem de `supported_reasoning_levels` do modelo escolhido.
  - A janela de trabalho do fatiamento continua decidida pelo app; o `context_window` serve só de teto.

### 2.4 `max_output_tokens`
- Foi aceito, mas a documentação manda omitir. **O app continua omitindo**, e a truncagem é tratada por `response.incomplete`.
- Se no futuro for preciso limitar a saída, isso exige nova verificação: a documentação pode ser atualizada e o comportamento pode mudar.

### 2.5 Tokens e sessão
- **Gravação:** cada novo refresh token é gravado de forma atômica **antes** de ser usado.
- **Concorrência:** as renovações continuam em série. A tolerância a reuso que foi observada não está documentada e não deve ser usada como garantia.
- **Quando renovar:** depois de `earliest_refresh_at` e antes de `expires_in`, ou ao receber 401. O caminho é refresh e **uma** nova tentativa.
- **Dados guardados:** o `sub` validado identifica a conta. O `client_id` fica guardado junto, e o ID token só como `id_token_hint`.

### 2.6 Privacidade e saída
- A URL de relogin contém o `id_token_hint` (o ID token) e o `login_hint` (o e-mail). **Nenhuma das duas pode aparecer em tela, log ou diagnóstico.** A PoC passou a omiti-las da URL exibida, e o app nunca exibe a URL.
- As respostas trazem `set-cookie`. O cliente HTTP do app **não** usa armazenamento de cookies, e isso deve continuar assim.

### 2.7 Ambiente
- No WSL, `explorer.exe <url>` não abriu o navegador. `rundll32.exe url.dll,FileProtocolHandler <url>` abriu.
- No app (Windows), `wapp.Browser.OpenURL` usa o mesmo mecanismo do `rundll32`.

## 2-A. Contas sem assinatura (gratuitas, login pelo Google) — 09/11/2026

> A conta que funcionou (§1) é **ChatGPT Plus**. As tentativas abaixo foram feitas com conta(s) **sem nenhuma assinatura** do ChatGPT. Curiosidade: a lista de modelos da conta Plus traz `free` em `available_in_plans`, mas uma conta gratuita não recebe tokens por esse fluxo. Isso é coerente com a documentação: só Plus e Pro podem usar o plano em outros aplicativos.

| Tentativa | O que aconteceu |
|---|---|
| Login completo, 1ª vez | A autorização **passou**: o retorno trouxe `code`, um `client_id` novo (`oaiapp_…`) e escopos que **incluem** `chatgpt.tokens.use.direct`. A **troca do code** foi recusada imediatamente com HTTP 400 `{"error":"invalid_grant"}`, **sem descrição** |
| Login completo, 2ª vez (janela anônima nova) | Igual: `invalid_grant` 0,0 s após o retorno, e **mais um** `client_id` novo emitido |
| Só identidade (`openid profile email offline_access`, sem `resource`) com `dynamic_agent_client` | A OpenAI recusa **na própria tela**: "Este aplicativo está indisponível… Volte para sys-executor e comece de novo", `error_code: invalid_client`, `product: generic_3p` |

**Conclusões:**
1. **Com conta sem assinatura (gratuita), o registro dinâmico autoriza, mas não entrega tokens.** O sinal é `invalid_grant` na troca do code, sem descrição, e não um `subscription_sharing_user_not_eligible`. É o mesmo código de um code expirado ou reutilizado.
2. **Não existe "login só de identidade"** pelo registro dinâmico: sem os escopos do plano, a tela da OpenAI recusa com `invalid_client`. O app sempre pede o conjunto completo de escopos.
3. **No app:**
   - Um `invalid_grant` na troca do code do **1º registro** vira um erro próprio. A mensagem diz que o login não foi concluído e que, se se repetir, é provável que a conta não tenha um plano elegível (Plus ou Pro) para uso em outros aplicativos.
   - O app oferece "Tentar de novo" **uma vez**, sem laço automático (a doc proíbe repetir OAuth em laço).
   - Nada é gravado e a credencial atual, se houver, fica intacta.
   - Cada tentativa recusada deixa um registro `oaiapp_…` órfão do lado da OpenAI. O app não reaproveita registros incompletos.
   - Um `invalid_client` aparece **na tela da OpenAI**, e não no app. O app só percebe que o retorno não chegou (o prazo de 5 min vence ou o usuário cancela) e então mostra o mesmo texto de orientação.

## 3. Não verificado nesta PoC
- **Comportamentos que dependem de outra conta ou de outro momento:**
  - recusa de consentimento (`access_denied`);
  - permissão do plano negada na tela;
  - recusa de inferência por conta não elegível (`subscription_sharing_user_not_eligible`); com a conta gratuita, a recusa veio **antes**, na troca do code (§2-A);
  - limite de uso atingido (`subscription_sharing_usage_limit_exceeded`).
  
  Os tratamentos seguem a documentação e serão testados com o servidor falso na etapa 4.
- Duração da janela de tolerância a reuso do refresh.
- Qualidade jurídica das respostas, o que fica para o bench da etapa 8 com casos sintéticos.
- DPAPI: a PoC usou arquivo 0600; no app, a credencial usa a DPAPI já testada no CI `go-windows`.

## 4. Conclusão
**A integração oficial é viável em Go, sem dependências além de `go-oidc/v3` e `go-jose/v4` (Apache-2.0) e `golang.org/x/oauth2` (BSD, indireta).** O cliente da Responses API é escrito à mão e não usa o SDK da OpenAI.

Comparado ao plano D-11:
- confirmado: a saída estruturada strict com os schemas atuais, sem plano B;
- ajustes nas etapas 3 e 4: classificação de erros por status e `code` (aceitando `detail`), SSE sem `Content-Type`, filtro de modelos por `supported_in_api` e níveis de raciocínio por modelo;
- atenção ao custo em tokens dos enums grandes.

**Próxima etapa (2):** barreira de privacidade e correções compartilhadas da pseudonimização. Depende de autorização expressa.
