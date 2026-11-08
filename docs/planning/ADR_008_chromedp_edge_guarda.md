# ADR 008 — Automação SEI com chromedp sobre Microsoft Edge e guarda de navegação

- **Status:** Aceita (08/11/2026). F0-08 executada pelo usuário com login real, 2FA, processo sigiloso e geração do PDF (RELATORIO_FASE0 §3). chromedp fixado em v0.23. As URLs carregam `infra_hash`, então a navegação é só por preenchimento e clique em elementos localizados por rótulo. A reautenticação de sigiloso pede a senha de novo (V4 §§13, 33).
- **Contexto:** V4 §§2, 36–43 e o plano `PLANO_IMPLEMENTACAO_SYS_EXECUTOR.md`.

## Decisão
O chromedp controla o **Microsoft Edge** instalado (Chromium), com perfil descartável por tarefa no diretório efêmero. Credenciais e OTP são digitados na UI do app, mantidos em `[]byte` e zerados após o uso. Os detectores de estado se baseiam em conteúdo e rótulos, nunca em IDs dinâmicos. Uma **guarda de navegação** (CDP `Fetch`) aplica allowlist de host (`sei.sistemas.ro.gov.br`) e de `acao=` de leitura, geração de PDF e download; o resto é bloqueado. O cliente Go **não possui** métodos de escrita. O download vai para o diretório efêmero e é verificado (PDF válido, número do processo, contagem de peças).

## Alternativas consideradas
- **Empacotar Chromium:** cerca de 150 MB a mais e atualização de segurança por conta do app.
- **Navegador visível em que o usuário navega sozinho:** perde a automação pedida (V4 §2.1). Pode servir de *fallback* para estados desconhecidos.
- **API do SEI (SOAP/WS):** exige credenciais de sistema e autorização institucional, fora do escopo da V1.

## Consequências
Depende do mapeamento real das telas (F0-08, executada pelo usuário). Mudanças na UI do SEI geram uma mensagem amigável e a alternativa de upload.
