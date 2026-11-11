# Relatório — Fase 5 (biblioteca jurídica), recorte essencial

- **Autorização do usuário (11/11/2026):** seguir para a F5 antes das Fases 2 e 3 do visual, para chegar ao Julgamento de PADS completo.
- **Decisões do usuário:**
  - **busca somente lexical** por enquanto (resolve a pendência da V4 §47.7; nenhum embedding, nenhum envio a provedor);
  - **F5 essencial** agora; o restante da F5 fica para depois da F7;
  - começar pelo **PADS**.
- **Sem mudança** em regra jurídica, golden, V4 ou dependências (45 módulos Go, 40 npm).

## O que foi feito

| Item do plano | Situação | Onde |
|---|---|---|
| F5-01 Leitor de normas | ✅ | `internal/legal/norma.go` |
| F5-02 Acervo inicial de `docs/law_library` | ✅ embutido no `.exe` (14 documentos; 2 escaneados lidos por OCR) | `cmd/bibgen`, `internal/legal/dados/` |
| F5-03 Busca | ✅ referência exata + palavras (FTS5/BM25) com tesauro. **Sem busca semântica** (decisão acima) | `internal/legal/biblioteca.go`, `referencia.go` |
| F5-04 Nota de vigência (D-01) | ✅ art. 41, § 3º do RDPM fora da recuperação, sem alerta | `internal/legal/vigencia.go`, `dados/notas_vigencia.yaml` |
| Etapa "Consultando fontes jurídicas" | ✅ PADS; na SR a etapa só passa adiante | `internal/pipeline/fontes.go` |
| F5-05 Gestão (incluir, substituir, excluir, bloqueio) | ❌ adiada | — |
| Tela de duplicidade RDPM × Coletânea | ❌ adiada (a busca já não repete o mesmo artigo) | — |
| F5-06 Pesquisa web opcional | ❌ adiada (nasce desligada, V4 §41) | — |

### Leitor de normas
- Separa caput, parágrafo, inciso e alínea, com a página e o contexto (título › capítulo › seção).
- **Redação histórica × atual:** quando o mesmo dispositivo aparece de novo, a redação anterior e o que estava abaixo dela ficam marcados como históricos. A nota "(Redação dada pelo…)" é guardada.
- "(Revogado pelo…)" no próprio texto marca o dispositivo como revogado.
- Citação no começo de uma linha ("Art. 65, inciso V, da Constituição…") não abre artigo: a numeração tem de seguir a sequência.
- **Partes:** quando a numeração recomeça, começa outra norma (o decreto que aprova e o regulamento aprovado; cada norma da Coletânea).
- Documento sem artigos (DAO, POP) vira um trecho por página.

### Acervo inicial
- `internal/legal/dados/manifesto.yaml` identifica cada original: título, siglas ("RDPM", "Estatuto"…) e categoria (V4 §40.5). Um PDF fora do manifesto faz o `bibgen` falhar.
- `go run ./cmd/bibgen` extrai o texto com o pdfium e grava `internal/legal/dados/acervo.json.gz` (cerca de 400 kB), embutido no `.exe`. A gravação é reprodutível.
- Cada documento guarda o **SHA-256 do original**; o começo dele é a "versão" mostrada em cada passagem (V4 §40.6).
- **OCR:** as Resoluções 208/2017 e 148/2002 são escaneadas. Foram lidas com o Tesseract da pasta de teste. O texto por OCR fica marcado e a interface pede conferência no original. Sem Tesseract, o `bibgen` reaproveita o texto já gravado, se o original não mudou.
- Um teste confere que o acervo embutido corresponde aos arquivos de `docs/law_library` (hash por hash).

| Documento | Págs. | Partes | Artigos | Dispositivos |
|---|---|---|---|---|
| RDPM | 28 | 2 | 99 | 455 |
| Estatuto (DL 09-A/1982) | 89 | 1 | 158 | 1.095 |
| Lei 4.302/2018 (LOB) | 24 | 1 | 51 | 398 |
| Regulamento Geral (Dec. 26.648/2021) | 64 | 2 | 106 | 1.165 |
| Coletânea (CL-2-PM) | 252 | 10 | 193 | 803 |
| Demais (9 documentos) | 316 | 10 | 88 | 590 |

### Busca
- **Referência exata:** "art. 39, parágrafo único, RDPM", "art. 17, inciso V, do RDPM", "art. 39, VII, a". Devolve a redação em vigor; para inciso ou alínea, traz também o caput.
- **Por palavras:** SQLite FTS5 em memória, um bloco por artigo, ordenado por BM25. Plural e acento não atrapalham; o tesauro (`dados/tesauro.yaml`) leva "pena" a "punição", por exemplo.
- O mesmo artigo na norma avulsa e na cópia da Coletânea aparece uma vez, e fica a avulsa.
- Toda passagem traz documento, arquivo, versão, página e referência por extenso. A passagem **não prova vigência** (V4 §40.3).

### Etapa "Consultando fontes jurídicas" (PADS)
- Recupera o dispositivo de **cada imputação** (pelo enquadramento confirmado) e a **base do julgamento** no RDPM:
  - sempre: arts. 4º, 32 e 33;
  - havendo imputação procedente: arts. 36, 37, 38, 39, 41, 45, 50, 58, 70 e 71.
- O que não for encontrado vira **lacuna** mostrada ao usuário; nada é inventado.
- O resultado aparece na aba Análise, seção **"Fontes jurídicas"**.
- Não usa a IA nem a rede.
- Depois dela, o PADS para em **"Calculando comportamento e sanção"** (TAR-004), que é a F6.

## Verificação
- `go test ./internal/... ./cmd/bench/...` verde; `-race` em `legal` e `pipeline`.
- Testes novos: LIB-PARSE-01 a 05, T28, LIB-HYB-01, LIB-D01-01 e a conferência das fontes no fluxo completo do PADS.
- `gofmt`, `GOOS=windows go vet ./...`, `rulescov -estrito` (202 regras; 77 com `enforced_by`), bench `-provedor falso` (0 violações), licenças (45 Go, 40 npm), golden sem diff, vitest 35/35.
- `go run ./cmd/bibgen -conferir`: acervo em dia.

## Pendências e observações
1. **Lei nº 2.484/2011** (citada na cláusula de cumprimento em residência) e a **Constituição Federal** não estão na biblioteca. Se a minuta precisar citá-las, o usuário inclui os textos em `docs/law_library` ou a citação fica como texto fixo aprovado (decidir na F6).
2. **Leitura imperfeita em normas secundárias:** a Coletânea e a Resolução 162/2004 trazem manuais com numeração própria; parte desses trechos fica marcada como histórica por engano. RDPM, Estatuto, LOB e Regulamento Geral foram conferidos.
3. **LIB-D01-02** (validador bloqueia citação do art. 41, § 3º na minuta) depende da F6.
4. **SR:** falta definir o conjunto de dispositivos da etapa.
5. **Adiados:** gestão do acervo, tela de duplicidade, pesquisa web, LIB-DUP-RDPM-01, T21 a T23, LIB-CAT-01.
