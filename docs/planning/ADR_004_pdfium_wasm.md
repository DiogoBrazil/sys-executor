# ADR 004 — Extração e renderização de PDF com go-pdfium (WebAssembly)

- **Status:** Aceita (08/11/2026). F0-04: 806 páginas em 0,67 s (4 workers); contingência CGO desnecessária.
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
Usar `klippa-app/go-pdfium` (MIT; PDFium Apache-2.0) no modo WebAssembly/wazero: sem CGO, sandboxado, com texto estruturado e posições (para realce) e renderização de página para OCR. A prévia na UI usa PDF.js (Apache-2.0).

## Alternativas consideradas
- **MuPDF/go-fitz:** AGPL, incompatível com distribuição fechada sem licença comercial.
- **pdfcpu:** ótimo para estrutura, fraco para extração de texto com posições.
- **Poppler CLI:** binários GPL, empacotamento mais pesado.

## Consequências
O desempenho é cerca de metade do nativo, e o risco em autos de 800+ páginas será medido na F0-04. Contingência: go-pdfium em modo CGO multiprocesso.
