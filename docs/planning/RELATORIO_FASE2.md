# RELATÓRIO DA FASE 2 — INGESTÃO DOS AUTOS (sys-executor)

- **Data:** 08/11/2026
- **Autorização:** expressa do usuário ("Pode seguir com a fase 2"), incluindo o push da branch `development`.
- **Escopo:** F2-01 a F2-07 (plano §16): envio do PDF, extração (pdfium), OCR, segmentação e função documental, evidências/busca/cobertura, visualizador com realce e cliente SEI com guarda.
- **Situação:** implementação concluída e testada com componentes reais (Edge, Tesseract, pdfium). **Faltam três validações do usuário** com dados reais (§4).

## 1. Critérios de saída (PLANO_TESTES_ACEITACAO §12)

| Critério | Verificação | Resultado |
|---|---|---|
| **A01/D1, A02/D2** — "Manifestação" antes/depois da instrução | `segment`: cronologia decidida pelo Go (D-09.5): antes do 1º termo → defesa prévia **provável**; depois do último → alegações finais **prováveis**; entre termos → indeterminada (pergunta na F4). Fixture SEI sintética de ponta a ponta no pipeline | **OK** |
| **A03** — datas divergentes em peças diferentes | `autos.Datas()`: todas as menções preservadas com página e posição (numéricas, por nome do mês, por extenso); as de OCR ficam "não confirmadas" (D-09.3). Resolver/perguntar é da Fase 4 | **OK (base determinística)** |
| **A04** — peça essencial ilegível | Mapa de cobertura marca ILEGÍVEL/OCR_BAIXO por página e por peça; peça central ilegível gera pergunta (T06). Pergunta para outras peças essenciais: Fase 4 | **Parcial (F4 completa)** |
| **A05/A06** — oitiva com/sem SEI | Peça guarda o número SEI ou é marcada como documento externo; a interface abre a página. A redação da citação é da Fase 6 | **OK (parte de ingestão)** |
| **A08** — mídia citada e não incorporada | `segment.DetectarMidias` (vídeo, gravação, câmera, áudio, mídia, videoconferência, link externo…) com página e posição; painel "Mídias mencionadas". A pergunta por descrição/transcrição é da Fase 4 | **OK (detecção)** |
| **T02** — upload sem credencial | Nenhum estado SEI no histórico; nenhum campo de senha no fluxo | **OK** |
| **T03** — SEI sem internet | Edge real contra porta fechada → REDE-001; a tarefa volta à escolha da entrada com o aviso e o tipo preservado; o envio do PDF segue | **OK** |
| **T04** — PADS escolhido, PDF de SR (e o inverso) | Pergunta "Os autos parecem ser de…", sem trocar o tipo; opções: enviar outro PDF, continuar, cancelar | **OK** |
| **T05** — PADS com portaria de SR anexa | Memorando encontrado → compatível, mesmo com a portaria depois | **OK** |
| **T06** — peça instauradora ilegível | Pergunta pedindo versão legível | **OK** |
| **T17 / ING-COB-01** — autos longos, cobertura exata | PDF sintético de **800 páginas** (3 em branco, 5 só com desenho) processado em **0,4 s**: 792 lidas, 3 vazias, 5 ilegíveis, nenhuma página fora de peça; progresso real em páginas | **OK** |
| **UI-SRC-01** — clique abre a página com realce | PDF servido por token da tarefa e sem cache; `Destaque` devolve as caixas do trecho (pdfium/OCR) e o PDF.js desenha o realce; sem posição, abre só a página | **OK** (realce testado no backend; visualizador validado por build e tipos) |
| **SEG13-01 / 02** — OTP e reautenticação | Edge real + simulador: código pedido ao usuário; processo sigiloso pede **só a senha, de novo**; segredos nunca aparecem nas visões enviadas à interface (teste com senha sentinela) | **OK** |
| **SEG13-07** — ação de escrita no SEI | Guarda CDP deny-by-default; o simulador tenta link, `<img>` e `fetch` para ações de escrita e alteração de 2FA: **nenhuma chegou ao servidor** | **OK** |

Ambiente: suíte Go completa nativa no Windows 11 do usuário (13 pacotes), com **Edge** (sem janela) e **Tesseract 5.4** reais — OCR da página escaneada da fixture com confiança 96,3%. CI no GitHub (Linux e Windows).

## 2. O que foi construído

| Tarefa | Entrega |
|---|---|
| F2-01 Upload | Diálogo nativo; PDF lido para a memória e gravado cifrado na sessão; PDF vazio/inválido mantém a tarefa aguardando; protegido (PDF-004) ou corrompido (PDF-003) volta ao envio com alerta, sem perder o tipo |
| F2-02 pdfium | `internal/ingest/pdf`: go-pdfium em WebAssembly (sem CGO), texto com caixas em pontos (origem no topo) e renderização |
| F2-03 OCR | `internal/ingest/ocr`: Tesseract por stdin/stdout (sem temporários), TSV com caixas e confiança; `internal/ingest/autos`: OCR só onde não há texto útil, página em branco detectada sem OCR, OCR em paralelo |
| F2-04 Segmentação | `internal/ingest/segment`: limites por número SEI, título e mudança nativo/escaneado; 18 funções documentais; cronologia pelo Go; compatibilidade T04–T06; mídias |
| F2-05 Evidências | Texto por página com fragmentos posicionados; datas mencionadas; índice FTS5 **em memória** (nada dos autos em disco em claro); resumo para a interface sem texto dos autos |
| F2-06 Visualizador | PDF.js 6.4.299 (carregado sob demanda), realce, navegação e busca; painel de peças, cobertura e mídias |
| F2-07 SEI | `internal/sei`: chromedp sobre o Edge instalado, sem janela; perfil e download na pasta da sessão; guarda deny-by-default; login, 2FA, pesquisa, reautenticação, geração e download; erros mapeados (SEI-001…008, REDE-001); simulador fictício `internal/testutil/seisim` |
| Tarefa | Perguntas pontuais; pedidos de dados ao usuário (credenciais, código, senha); erro recuperável; parada explícita (TAR-004) nas etapas de fases futuras, preservando a consulta aos autos; sessão apagada só depois que a etapa em execução para |

## 3. Decisões técnicas (registradas no REGISTRO como PT-F2-xx)

1. **PT-F2-01** Índice de busca dos autos só em memória (SQLite `:memory:`), não um arquivo temporário: o arquivo guardaria o texto em claro e contornaria a cifra da sessão.
2. **PT-F2-02** A guarda do SEI aplica a lista de ações a **qualquer tipo de recurso** nos controladores (`controlador.php`, `controlador_ajax.php`): uma `<img src="controlador.php?acao=…">` também executaria a ação.
3. **PT-F2-03** Toda falha do SEI devolve a tarefa à escolha da entrada, com o aviso e o tipo preservados, para o usuário enviar o PDF.
4. **PT-F2-04** Etapa de fase futura **pára** a tarefa com aviso (não a destrói): o usuário consulta os autos e cancela quando quiser. Substitui o comportamento da F1 (falha imediata).
5. **PT-F2-05** Edge **sem janela** (`--headless=new`), sem disfarçar o agente do navegador; `SYSEXEC_SEI_VISIVEL=1` mostra a janela para diagnóstico.

## 4. Pendências que dependem do usuário

1. ~~**`acao_ajax`**~~ **Resolvido (08/11/2026):** o usuário inspecionou o ícone PDF (link comum) e a função `gerar()` (envio de formulário): a geração não usa `controlador_ajax.php`. A guarda mantém todo `acao_ajax` bloqueado; o simulador passou a reproduzir o HTML real.
2. ~~**P-09 — estrutura de um PDF real**~~ **Resolvida.** A 1ª e a 2ª execução do `autos-mapa` sobre um PDF real de 67 páginas (08/11/2026; só estrutura, sem texto) mostraram:
   - **não há** o rodapé "Referência: Processo nº … SEI nº …" suposto; o SEI **carimba toda página** com `<nome do documento na árvore> (<nº SEI>) SEI <nº do processo> / pg. <página>` (parênteses às vezes ausentes);
   - documentos trazidos de outro processo mantêm **carimbos antigos** na mesma linha;
   - algumas fontes faziam o modo por retângulos do pdfium devolver letras separadas e duplicadas;
   - nomes reais da árvore: capa, memorando, despacho, ofício, certidão, defesa prévia, notificações, "Anexo - Ciência…", atas de presença por videoconferência, alegações finais, termo de remessa.
   Correções aplicadas: segmentação pelo carimbo deste processo (número de processo dominante e `pg.` = página), função pelo nome da árvore, texto pelo modo por caractere, carimbo ignorado na decisão de OCR e de página em branco, funções "capa" e "ciência", atas de videoconferência como instrução na cronologia. **3ª execução (08/11/2026): validada.** 67 de 67 páginas com carimbo (3 processos distintos nos carimbos; os antigos foram descartados); **30 peças**, na mesma ordem e com os mesmos tipos da árvore do processo vista no mapeamento do SEI; memorando encontrado (PADS compatível); blocos de assinatura/autenticação do SEI agora legíveis; 1 página escaneada (só carimbo como texto) corretamente enviada ao OCR e mantida na peça certa. Conferência com a árvore do processo (print do usuário): **31 documentos na árvore, 30 peças no PDF, mesma ordem e mesmos limites**; o documento ausente é um vídeo (arquivo compactado), que o SEI não inclui no PDF consolidado. As 4 peças sem tipo eram Termo de Juntada, Representação, Procuração e Informação, agora reconhecidas.
   - **Melhoria registrada para a Fase 4 (V4 §17; A08–A10):** na obtenção pelo SEI, ler a lista de documentos da árvore (nome e tipo de arquivo, sem conteúdo) e comparar com as peças do PDF, para apontar mídias e arquivos que ficaram fora (como o vídeo). No envio manual, só as menções no texto permitem detectar a mídia.
   - Limitação da ferramenta: palavras genéricas permitidas podem revelar o gênero gramatical ("a militar"); nenhuma outra informação identificável foi observada.
3. **Teste com o SEI real** no app (pacote de teste em `Documentos\sys-executor-teste`). Rodadas de 08/11/2026:
   - 1ª: SEI-005 sem pista → diagnóstico passou a descrever a tela (estrutura, sem valores).
   - 2ª e 3ª: a página respondeu "Informe a Senha.". Causa (página pública de login): o SEI/RO usa o componente **MaskedPassword**, que troca o campo de senha por um campo **oculto** `name="pwdSenha"` (enviado ao servidor) e um campo **visível** `id="pwdSenha"` **sem name**, que mostra "●" e repassa os caracteres; a validação lê o campo visível por id. O cliente procurava só por `name` e caía no campo oculto. Correção: campos localizados por `name` ou `id`, preferindo o visível, com digitação real; simulador reproduz o componente (código próprio) também na reautenticação; contraprova confirma que a busca só por `name` reproduz o erro.
   - 4ª: login, 2FA, pesquisa e reautenticação de sigiloso **funcionaram** no SEI real; falhou na geração do PDF ("elemento não encontrado"). Provável causa: o rótulo da opção "Todos os documentos disponíveis"; a opção passa a ser localizada também pelo id `optTodos` (o que `gerar()` consulta) e aceita como já marcada; cada passo informa qual elemento faltou, com a estrutura da tela.

## 5. Limitações conhecidas

- A senha digitada chega ao Go como `string` (Wails) antes de virar `[]byte`; essa cópia não pode ser zerada (V4 §33 reconhece o limite da limpeza de memória gerenciada).
- O componente do visualizador (PDF.js) não tem teste E2E automatizado: o realce é validado no backend; a interface, por build e tipos.
- Tesseract empacotado no teste é o da F0 (com DLLs GPL/LGPL); a recompilação sem elas (D-09.7) é da Fase 8.
