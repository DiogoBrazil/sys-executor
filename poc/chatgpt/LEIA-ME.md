# PoC — Sign in with ChatGPT (etapa 1 do plano D-11)

Módulo Go isolado. **Somente dados sintéticos.** Não toca no pipeline do app.

## O que verifica
1. Registro dinâmico (`dynamic_agent_client` → `client_id` emitido `oaiapp_…`).
2. Login com Authorization Code + PKCE S256, `state`, `nonce` e retorno em `127.0.0.1` (porta aleatória).
3. ID token: assinatura (JWKS), emissor, destinatário, expiração e `nonce`, via `github.com/coreos/go-oidc/v3`.
4. Escopos concedidos, em especial `chatgpt.tokens.use.direct`.
5. Refresh com rotação e reuso do refresh antigo (comando `reuso`).
6. `GET /v1/models`.
7. `POST /v1/responses` com `store:false`/`stream:true`, SSE até `response.completed`, histórico de reenvio, recusas esperadas (`system`, `max_output_tokens`, `temperature`), `reasoning`, `json_object` e **`json_schema` strict com os schemas reais** do app (`schemas/`, gerados de `internal/analise` sem dados pessoais).
8. Relogin com o `client_id` salvo e outra porta.
9. Revogação.

## Como rodar (WSL ou Windows)
```
cd poc/chatgpt
go test ./...                       # testes offline (servidor falso)
go run . login                      # abre o navegador; entre com a conta ChatGPT
go run . status
go run . modelos
go run . bateria -modelo <slug>     # ~15 requisições curtas
go run . renovar
go run . relogin
go run . reuso                      # PODE invalidar a sessão; rode por último
go run . revogar                    # revoga e apaga a credencial local
```

## Segurança
- Nenhum token é impresso.
- A credencial da PoC fica em `$(os.UserConfigDir)/sys-executor-poc/chatgpt.json`, com permissão 0600. No WSL o caminho é `~/.config/sys-executor-poc/`. Diferente do app, **não** usa DPAPI e guarda também o access token, para não renovar a cada comando. `revogar` remove o arquivo.
- O cliente HTTP só aceita `https://auth.openai.com` e `https://api.openai.com` e não segue redirecionamentos.
- O retorno aceita um único `GET /auth/callback`, com o `Host` conferido.
