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

