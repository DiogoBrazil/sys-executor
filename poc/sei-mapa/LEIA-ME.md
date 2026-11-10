# F0-08 — Mapeamento do SEI/RO (executado pelo próprio usuário)

Ferramenta **somente de observação**. Ela não clica, não digita e não grava nada em disco. O perfil do Edge é temporário e é apagado ao sair. Os downloads são **bloqueados** de propósito, e todos os dígitos aparecem mascarados como `#`.

## O que é registrado (e só isto)
- Os valores de `acao=` e os **nomes** (não os valores) dos parâmetros de cada navegação.
- Os hosts contatados.
- A estrutura dos formulários: tipo e nome dos campos, nunca o que foi digitado.
- Os textos de botões e de links que coincidem com rótulos conhecidos da interface ("Gerar Arquivo PDF do Processo", "Todos os documentos disponíveis", "Gerar"…).
- O início de um download, com o nome sugerido mascarado.

## Como executar
1. Gerar o executável (no WSL): `cd poc/sei-mapa && GOOS=windows GOARCH=amd64 go build -o sei-mapa.exe .`
2. Copiar o `.exe` para uma pasta do Windows (ex.: `cp sei-mapa.exe /mnt/c/Users/<usuário>/Documents/`) e executar no **PowerShell**: `.\sei-mapa.exe`. No bash do WSL, `.\` não funciona.
3. No Edge que abrir:
   1. Faça login (usuário, senha, órgão "PM"). Se houver OTP/2FA, digite normalmente.
   2. Abra **um processo de teste ou já encerrado** pelo número (Pesquisa rápida).
   3. Clique em **"Gerar Arquivo PDF do Processo"** → **"Todos os documentos disponíveis"** → **"Gerar"**.
   4. Se possível, repita com um processo de acesso restrito para observar a reautenticação.
   5. **Não** use nenhuma função de escrita (incluir documento, assinar, enviar, concluir…).
4. Volte ao console e digite `sair` + Enter.
   - Se o console mostrar `[dialogo] >>> a página está PARADA`, leia a mensagem e digite `aceitar` ou `cancelar`. Nunca aceite nada que altere o processo.
   - As linhas `[resp] HTTP …` e `[falha] …` mostram o status de cada página carregada, para diagnóstico.
5. **Revise** o bloco "RESUMO" e as linhas `[tela]`. Se aparecer algo que identifique pessoa ou processo, apague antes de colar. Depois cole aqui na conversa.

## Já observado no teste de fumaça (página pública de login, sem credenciais, 08/11/2026)
- Formulário POST com `txtUsuario`, `pwdSenha` (há um campo texto homônimo, provavelmente o "mostrar senha"), `selOrgao` (select) e o oculto `hdnAcao`.
- Botão "ACESSAR"; links "Entrar com gov.br" e "Autenticação em dois fatores".
- Um único host: `sei.sistemas.ro.gov.br`.

## Etapa 7 do plano D-11 — tela "Apenas selecionados" (10/11/2026)

Continua **somente observação**: nada é clicado, digitado ou gravado. O que a extensão acrescenta:
- **Comando `mapear`:** descreve a tela "Gerar Arquivo PDF do Processo" em todos os quadros:
  - rádios `rdoTipo`;
  - *selects*, como `selDocumentosApenas` (se é múltipla, quantas opções tem, a forma de até 10 opções e dos valores);
  - caixas de seleção e as linhas da tabela em que estão;
  - tabelas (cabeçalhos e forma de até 4 linhas);
  - campos ocultos, botões e scripts externos;
  - o código das funções da página ligadas à seleção (`gerar()` e os handlers dos controles), com o conteúdo das strings mascarado.
- **POST da geração:** em `procedimento_gerar_pdf`, mostra os **nomes** dos campos e a **forma** dos valores, por exemplo `hdnDocumentosApenas = "########,########"`. O POST de outras telas, como o do login, **não é lido**.
- **Forma, nunca texto:**
  - cada palavra vira `Aa`, `AA` ou `a`, e cada número vira `#` repetido;
  - só ficam em claro as palavras da lista fixa de tipos documentais e rótulos (`tiposDocumentais` em `mapear.go`), por exemplo `Termo de Inquirição - AA AA AA (#######)`;
  - nomes de pessoas, unidades e números nunca aparecem.
- **Teste:** `go test .` (com o Chrome de desenvolvimento, sem GPU) confere que nada da página fictícia vaza.

### Roteiro
1. **Gerar o executável** (no WSL):
   ```bash
   cd poc/sei-mapa && GOOS=windows GOARCH=amd64 go build -o sei-mapa.exe .
   cp sei-mapa.exe /mnt/c/Users/<usuário>/Documents/
   ```
2. **Executar** no PowerShell, na pasta do executável: `.\sei-mapa.exe`.
3. **No Edge que abrir:**
   1. Faça o login.
   2. Abra **um processo de teste ou já encerrado**, de preferência com poucos documentos.
   3. Clique no ícone **"Gerar Arquivo PDF do Processo"**.
   4. Volte ao console, digite **`mapear`** + Enter. Esta leitura mostra a tela antes da seleção.
   5. No Edge, marque **"Apenas selecionados"**. Se aparecer uma lista, uma tabela ou uma janela de escolha, **selecione 1 ou 2 documentos**.
   6. No console, digite **`mapear`** de novo. Esta leitura mostra a tela depois da seleção.
   7. No Edge, clique em **"Gerar"**. O download é bloqueado de propósito, e o console mostra as linhas `[post-gerar]`.
   8. **Não** use nenhuma função de escrita.
4. **Encerrar:** digite `sair` + Enter.
5. **Revisar e colar na conversa:**
   - as linhas `[gerar-pdf]` (as duas leituras);
   - as linhas `[post-gerar]`;
   - as linhas `[nav]` da geração;
   - o RESUMO.
   
   **Revise antes de colar.** Se alguma palavra em claro identificar pessoa ou processo, apague-a.
