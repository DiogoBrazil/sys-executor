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
