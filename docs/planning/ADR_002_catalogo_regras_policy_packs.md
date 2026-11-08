# ADR 002 — Catálogo de regras + policy packs + motor + validadores

- **Status:** Aceita como planejamento (08/11/2026). A F0-03 reforçou a necessidade do validador Go (injeção de prompt, datas, cronologia); ver D-09.
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
As regras da V4 são operacionalizadas por cinco mecanismos complementares:
1. catálogo YAML com IDs estáveis e origem V4;
2. motor determinístico em Go;
3. policy packs por etapa, com núcleo invariável e regras selecionadas por chave;
4. saídas JSON Schema com enums dinâmicos;
5. validadores independentes, com laço de correção.

Detalhes em `REGRAS_IA_ESTRATEGIA.md`.

## Alternativas consideradas
- **V4 integral no prompt:** não cabe (≈45k tokens × contextos de 4k–32k); modelos pequenos a seguem mal; enforcement probabilístico.
- **Resumo da V4 no prompt:** perde exceções (vedado pela V4 §§28.4, 46).
- **Recuperação vetorial de regras:** pode omitir uma proibição crítica; usada só para fontes jurídicas, nunca para regras.

## Consequências
Exige manter o catálogo sincronizado com a V4 (`rulescov`, `V4.lock`). Em troca, cada regra é rastreável, testável e versionada.
