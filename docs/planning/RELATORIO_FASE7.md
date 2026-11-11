# Relatório — Fase 7 (documentos e finalização), PADS

- **Autorização do usuário (11/11/2026):** concluir as partes 3 e 4 da F6 e a F7, com conferência completa dele ao final.
- **Base:** ADR-007 (DOCX por escritor OOXML próprio; PDF pelo Typst) e V4 §§31.3 e 32.

## O que foi feito
- **Pacote `internal/documento`:** o DOCX e o PDF saem do mesmo `minuta.Documento`.
  - **DOCX** (`docx.go`): `archive/zip` + XML, sem biblioteca de terceiros. Estilos por tipo de bloco, cabeçalho de página com o logotipo e as duas linhas, rodapé com o campo de página.
  - **PDF** (`typst.go`): gera o código Typst e chama o `typst.exe` de `componentes\typst\`. Todo o texto da minuta vai dentro de literais de string, para que nada dele seja interpretado como marcação. Os intermediários ficam na pasta da sessão e são apagados em seguida.
- **Formatação** (V4 §32; os números do §32.2 são proposta):
  - A4; margens superior e inferior de 2 cm, esquerda de 3 cm, direita de 1,5 cm;
  - Times New Roman 12 (Liberation Serif como segunda opção no PDF), entrelinha simples, justificado, recuo de primeira linha de 2,5 cm;
  - cabeçalho: logotipo centralizado, "SECRETARIA DE SEGURANÇA, DEFESA E CIDADANIA" e "POLÍCIA MILITAR DO ESTADO DE RONDÔNIA" em negrito; título "JULGAMENTO DE PADS";
  - `[A PREENCHER]` realçado em amarelo;
  - espaço de assinatura sem assinatura.
- **Sem dados a mais:** o DOCX não leva autor nem data de geração; o PDF não leva autor. O nome sugerido é `Minuta_Julgamento_PADS_v<versão>_<data>_<hora>`, sem nome de pessoa.
- **Logotipo:** lido de `componentes\cabecalho\` (`logo-pmro.png` ou `logo-cabecalho-docs.jpeg`). Sem imagem válida, o cabeçalho sai só com o texto. O arquivo do usuário **não entrou no repositório**.
- **Interface:** na revisão, "Salvar DOCX" e "Salvar PDF" abrem o diálogo "Salvar como". O aplicativo não guarda cópia nem o caminho escolhido.
- **Erros:** EXP-001 (falha ao gerar), EXP-002 (Typst ausente; novo no catálogo), ARQ-002 (falha ao gravar).
- **Finalização:** "Aprovar e finalizar" está descrita no `RELATORIO_FASE6.md`, parte 4.

## Verificação
- **DOCX:** todas as partes são XML bem formado; estilos, margens, cabeçalho, rodapé e realce conferidos; com e sem logotipo.
- **PDF:** compilado de verdade com o Typst 0.15.1 (Linux), lido de volta com o pdfium e comparado com o texto da minuta, inclusive caracteres que quebram marcação (`#`, `$`, `*`, `_`, `@`, `\`, aspas). Páginas A4, mais de uma página, intermediários apagados.
- Amostra fictícia conferida visualmente (`SYSEXEC_AMOSTRA=<pasta> go test -run TestAmostra ./internal/documento`).
- **Não verificado nesta máquina:** a abertura do DOCX no Word e o PDF gerado no Windows com a Times New Roman. O Word abriu o DOCX da PoC da F0, que tem a mesma estrutura.

## Diferenças em relação ao plano
- **Prévia:** o plano previa a prévia do PDF dentro do aplicativo (PDF.js). Ficou a prévia em folha HTML, que já existia; o PDF é conferido no arquivo salvo.
- **Logotipo:** a V4 §32.1 cita `logo-pmro.png` (brasão da PMRO). O arquivo fornecido é o logotipo do Governo do Estado (`logo-cabecalho-docs.jpeg`). Os dois nomes são aceitos; qual usar é decisão do usuário.
- **Typst:** não vai dentro do `.exe`; é um componente ao lado, como o Tesseract. Nesta sessão ele não pôde ser baixado para a pasta de teste; o usuário precisa colocá-lo (guia §4.2).
- **Liberation Serif** (reserva da Times New Roman, D-09.8) não foi empacotada: no Windows a Times New Roman existe. Fica para a F8.
- **Testes visuais** (diferença perceptual entre DOCX e PDF, F7-05) não foram feitos.

## A fazer
- SR (título "RELATÓRIO DE SINDICÂNCIA REGULAR", alíneas, estrutura própria).
- Decisões do usuário listadas no guia §6.0, item 4.
