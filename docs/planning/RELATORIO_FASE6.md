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

## A fazer
- **Parte 2 — redação:** modelo do documento, textos fixos, redação por trechos com a IA.
- **Parte 3 — validação:** validadores determinísticos, verificação de suporte, laço de correção.
- **Parte 4 — revisão:** prévia da minuta com o aviso de revisão obrigatória e pedidos de ajuste.
- SR.
