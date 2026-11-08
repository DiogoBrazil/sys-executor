# ADR 005 — OCR com Tesseract 5 como sidecar

- **Status:** Aceita (08/11/2026). F0-05: stdin OK, CER 0,08%. Datas/números de OCR entram como não confirmados (D-09.3); binário recompilado sem GPL/LGPL e com strip (D-09.7); tessdata `por` + `osd`.
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
Tesseract 5 (Apache-2.0) empacotado, com `tessdata_best` `por`+`eng` (Apache-2.0), chamado por página **somente** quando o texto nativo é insuficiente. A confiança por página é registrada. Peça essencial com baixa confiança → pede versão legível (V4 §16.3).

## Alternativas consideradas
- **Windows.Media.Ocr:** exige identidade de pacote (MSIX) e pacote de idioma OCR pt-BR possivelmente ausente.
- **PaddleOCR/RapidOCR:** dependências Python/ONNX pesadas; qualidade em pt-BR a provar.
- **OCR via modelo multimodal (Qwen3.5 com visão):** caro em CPU, sem medida de confiança; pode ser avaliado no futuro como reforço.

## Consequências
Instalador cresce cerca de 30–60 MB. A qualidade é medida com as Res. 208 e 148 da biblioteca (escaneadas) e com fixtures rasterizadas.
