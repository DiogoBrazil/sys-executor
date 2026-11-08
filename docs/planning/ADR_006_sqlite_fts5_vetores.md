# ADR 006 — Biblioteca jurídica em SQLite FTS5 + vetores em BLOB

- **Status:** Aceita (08/11/2026). F0-07: FTS5 e busca vetorial exata (~2 ms) confirmados; acervo inicial pré-indexado no instalador e ajustes de recuperação (D-09.9, D-09.10).
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
Um banco SQLite (`modernc.org/sqlite`, sem CGO, com FTS5) guarda documentos, versões, dispositivos estruturados, chunks, embeddings (BLOB float32 + identificação do modelo) e notas de vigência. A similaridade vetorial é calculada por **busca exata em Go**: o acervo é de milhares de chunks e a força bruta custa milissegundos. A recuperação é híbrida (referência exata → BM25 → semântica → fusão RRF). Os autos usam um SQLite FTS5 temporário separado, apagado com a sessão.

## Alternativas consideradas
- **chromem-go:** viável, mas adiciona um segundo armazenamento a sincronizar com o FTS. Fica como contingência se o acervo crescer muito.
- **sqlite-vec:** extensão nativa (CGO ou WASM) com ganho irrelevante nesta escala.
- **Banco vetorial servidor (Qdrant etc.):** desproporcional e com rede local.

## Consequências
Um único arquivo, transações atômicas na inclusão/substituição/exclusão (V4 §40.6–40.7) e reindexação de embeddings sem tocar nos originais.
