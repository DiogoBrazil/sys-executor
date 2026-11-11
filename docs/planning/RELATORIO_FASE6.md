# Relatório — Fase 6 (cálculo, composição e validação), PADS

- **Autorização do usuário (11/11/2026):** F6 para o PADS, em seguida à F5 essencial.
- **Decisões do usuário:**
  - o art. 5º, LV, da Constituição entra como **texto fixo** na minuta;
  - a Lei nº 2.484/2011 fica pendente de confirmação dele (a cláusula de cumprimento em residência segue a redação aprovada da V4 §10);
  - os modelos de `docs/draft_templates/PADS` servem de referência de forma e estilo.
- A fase é entregue em partes, cada uma com `.exe` e roteiro de teste.

## Parte 1 — cálculo oficial (11/11/2026)
- **Etapa "Calculando comportamento e sanção"** (`internal/pipeline/calculo.go`), sem IA e sem rede. O motor determinístico (`internal/rules/engine`) já existia e não mudou; a etapa só o alimenta com as respostas confirmadas:
  - **conexão** (V4 §5): nas conexas, só a principal recebe sanção; as demais entram como agravante (art. 39, II);
  - **sanção pela natureza** (art. 58, II): leve → repreensão; média → detenção; grave → prisão;
  - **circunstâncias**: as atenuantes e agravantes que o usuário reconheceu ao confirmar o resultado, o comportamento anterior (art. 38, I, ou art. 39, I), os relevantes serviços (art. 38, II) e a reincidência (art. 39, III), calculada pelo motor com a data de término informada;
  - **dias** (V4 §6.3): 2, 3 ou 1, conforme a comparação entre atenuantes e agravantes; repreensão não tem dias;
  - **comportamento proposto** (V4 §§8 e 15.4): regressão especial ou desconto ordinário, nunca os dois.
- **Conferência pelo usuário:** a pergunta "Confira a dosimetria e o comportamento proposto" mostra a dosimetria de cada sanção e os sete itens da memória de cálculo (V4 §8.8). Os dias são uma proposta e podem ser ajustados (detenção de 1 a 10).
- **Pendências do motor** (data de término ausente, 29/02, saldo negativo…) aparecem como alertas. Comportamento e pontuação que não combinam **impedem** a confirmação, até a correção em "Suas respostas".
- **Revisão:** mudar um resultado, a conexão, o comportamento, o histórico, a reincidência ou os serviços derruba a confirmação e refaz o cálculo. Alterar só o cálculo não refaz a análise.
- As invariantes do julgamento (`pads.Julgamento.VerificarCalculo`) são conferidas a cada cálculo.
- Sem imputação procedente, a etapa passa direto.
- O painel ganhou a seção **"Cálculo oficial"**.
- **Correção junto com esta parte:** hífen de fim de linha na leitura dos autos (`internal/ingest/pdf`): o pdfium o marcava com U+0002, e "Verifica-se" era lido como "Verificase".
- **Verificação:** fluxo completo do PADS com motor roteirizado (2 dias e 23 pontos no caso base; 3 dias com reincidência depois de rever a resposta), `gofmt`, `vet`, `rulescov -estrito`, vitest.

## Parte 2 — redação da minuta (11/11/2026)
- **Decisões do usuário:** os dados que o aplicativo não tem (autoridade, nº do PADS, local, quem assina) saem como `[A PREENCHER]`, sem pergunta; a cláusula de cumprimento em residência mantém a redação da V4 §10, com a Lei nº 2.484/2011.
- **Etapa "Elaborando a minuta"** (`internal/pipeline/redigir.go`), só para o PADS. Na SR a etapa pára como etapa futura.
- **A IA redige três tipos de trecho** (`internal/analise/redacao.go`), um por chamada, na etapa "redacao" das políticas (os prompts de referência não mudaram; a tarefa de cada trecho vai no pedido):
  - o relato do RELATÓRIO (fatos, defesa prévia, provas, alegações finais);
  - a FUNDAMENTAÇÃO de cada imputação, presa ao resultado que o usuário confirmou;
  - os sete fatores do art. 36 de cada imputação punida; sem elemento nos autos, "Não há elementos nos autos sobre este fator."
- **Conferência de cada trecho pelo Go**, com reenvio à IA em caso de recusa:
  - todo parágrafo cita evidências que existem;
  - citação de norma só do que veio da biblioteca, das evidências ou do texto fixo aprovado (art. 5º, LV, da Constituição);
  - nada de página ou folha, de afirmação de ter visto mídia, de identificador interno, de `[A PREENCHER]` escrito pela IA nem do art. 41, § 3º.
  - Trecho que a IA não consegue redigir com segurança, ou imputação sem prova extraída, fica `[A PREENCHER]` com pendência; nada é inventado.
- **O aplicativo compõe o resto** (`internal/minuta/`): cabeçalho, abertura do relatório, lista das imputações com o texto do dispositivo (da biblioteca), ampla defesa, DECISÃO (resultado global e por imputação), natureza e punição, arts. 37, 38 e 39 com as circunstâncias contadas pelo motor, dias, proposta de pontos e comportamento (ou a justificativa de regressão da V4 §15.4), cláusula de residência (só com detenção ou prisão), providências (nota de punição só depois da irrecorribilidade) e fecho.
- **A dados pessoais:** as chamadas passam pela mesma barreira das outras etapas (pseudonimização na ida, restauração na volta; revisão de possíveis nomes, se surgir algum).
- **Trechos prontos não são pedidos de novo** quando a etapa recomeça. Rever uma resposta da análise descarta a redação; rever só o cálculo refaz a composição.
- **Interface:** a minuta aparece na tela (`MinutaPrevia.vue`) com o aviso de revisão obrigatória (V4 §26.3), os campos a preencher realçados, a lista do que conferir, as fontes dos autos de cada parágrafo da IA e o botão "Copiar o texto". Cenário `#minuta` na galeria.
- Depois da redação, o PADS segue para a validação (parte 3). A minuta só aparece na tela depois dela.
- **Verificação:** fluxo completo com motor roteirizado (minuta conferida frase a frase nos pontos fixos; 2 dias no caso base e 3 com reincidência), vitest 36/36, captura da galeria revisada.
- **Limites conhecidos:**
  - o gênero do acusado não é conhecido: os textos fixos usam "o(a) acusado(a)";
  - a voz da decisão segue os modelos do usuário ("RESOLVO", "imponho"); a proposta de pontos fica como proposta (PADS-RED-006);

## Parte 3 — validação automática (11/11/2026)
- **Etapa "Validando a minuta"** (`internal/pipeline/validar.go`), obrigatória antes da revisão e depois de cada ajuste (V4 §42.1).
- **Verificação de suporte** (`internal/analise/validacao.go`, etapa "validacao" das políticas), uma chamada por trecho:
  - a IA recebe cada parágrafo e **só** as evidências que ele cita, e responde SUPORTA, PARCIAL ou NAO_SUPORTA, com o motivo;
  - na fundamentação, responde também se o texto contraria o resultado confirmado pelo usuário;
  - a resposta precisa trazer um item por parágrafo, e o motivo sempre que não for SUPORTA.
- **Laço de correção** (V4 §42.2):
  - trecho com NAO_SUPORTA ou contradição é **reescrito** com os problemas apontados e conferido de novo, até 2 vezes;
  - se persistir, vira item de "O que conferir ou completar", com o início do parágrafo e o motivo;
  - PARCIAL não é reescrito: vira item para conferência.
- Se a IA não conseguir reescrever com segurança, o texto anterior é mantido e a pendência fica à vista. Se a conferência não puder ser feita, o trecho é marcado para conferência redobrada. Nada é dado como validado sem ter sido.
- **Trecho já conferido não é conferido de novo** enquanto o texto não mudar.
- **Validadores determinísticos** (`internal/minuta/validar.go`), sobre a minuta inteira, contra as respostas e o cálculo oficial:
  - seções obrigatórias na ordem;
  - todas as imputações do memorando, e só elas, cada uma com o resultado confirmado; resultado global coerente;
  - uma punição por sanção calculada, com a natureza, a espécie e os dias do cálculo; pontos e comportamento do cálculo;
  - cláusula de residência só com detenção ou prisão; nota de punição só com punição; absolvição sem punição;
  - proibições de forma (art. 41, § 3º; identificador interno; página, mídia e marcador no texto da IA; assinatura ou publicação simulada);
  - parágrafo da IA com evidência, e evidência existente.
- Falha dos validadores **não bloqueia** a revisão: a minuta vai ao usuário com o alerta VAL-001 e a lista das falhas em destaque.
- **Verificação:** 16 adulterações da minuta, cada uma apanhada pelo validador certo. No fluxo com motor roteirizado, a fundamentação recusada uma vez foi reescrita e aprovada.

## Parte 4 — revisão e ajustes (11/11/2026)
- **Estado "Aguardando sua revisão":** a minuta validada fica na tela com a versão, o aviso de revisão obrigatória e as ações.
- **Solicitar ajustes** (V4 §31.3): o usuário escreve o pedido (até 2.000 caracteres).
  - A IA **classifica** o pedido (etapa "ajuste"): redação de trecho(s), decisão confirmada, texto fixo ou indefinido.
  - **Redação:** só os trechos atingidos são reescritos, com o pedido e as mesmas provas; os demais ficam iguais. Em seguida a minuta inteira é validada de novo, e só os trechos mudados voltam à verificação de suporte.
  - **Decisão** (resultado, punição, dias, pontos, circunstâncias): nada é alterado; o aplicativo orienta a usar "Alterar" em "Suas respostas" (V4 §42.2, item 2).
  - **Texto fixo** (cabeçalho, dispositivo, providências, fecho, formatação): não é reescrito pela IA; a orientação é ajustar no DOCX.
  - O pedido passa pela barreira de privacidade como qualquer envio. Fica só na memória da tarefa, com o resultado, e some com ela.
- **Aprovar e finalizar** (V4 §§11 e 29.1): confirmação explícita ("Pode finalizar"), com aviso se nenhum arquivo foi salvo. A tarefa vai a "Finalizada" e a sessão é apagada.
- Alterar uma resposta continua possível na revisão: a minuta é retirada, refeita e validada de novo.
- **Limites conhecidos:**
  - o pedido que mistura redação e decisão é tratado como decisão (nada é reescrito);
  - a verificação de suporte, a classificação e a reescrita só foram exercitadas com IA simulada.

## A fazer
- SR.
