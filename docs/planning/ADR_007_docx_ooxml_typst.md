# ADR 007 — DOCX por writer OOXML próprio + PDF por Typst (D-03)

- **Status:** Aceita (08/11/2026). F0-06: o Word abriu o DOCX sem reparo; Typst 0.15.1 embute TNR no Windows. Liberation Serif empacotada como fallback (D-09.8).
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
Um DocModel único gera:
- (a) DOCX, por writer OOXML próprio sobre um template .docx base (estilos, cabeçalho com brasão, rodapé com numeração), usando `archive/zip` + `encoding/xml`;
- (b) PDF, pelo Typst CLI embutido (Apache-2.0), com uma fonte `.typ` gerada.

Times New Roman vem do Windows (não redistribuída). Fallback métrico: Liberation Serif (OFL), empacotada (D-09.8).

## Alternativas consideradas
- **LibreOffice headless:** ~350 MB, mais lento, maior superfície de ataque (rejeitado pelo usuário em D-03).
- **Word via COM:** nem sempre presente; dois caminhos de teste.
- **Bibliotecas DOCX de terceiros:** unioffice (comercial), outras imaturas ou com licença restritiva.
- **HTML → PDF pelo Edge:** a hifenização e a paginação dependem do navegador.

## Consequências
A paginação do DOCX e do PDF pode diferir; o conteúdo e a formatação são idênticos e verificados por golden files e diff de texto. A prévia mostra o PDF.
