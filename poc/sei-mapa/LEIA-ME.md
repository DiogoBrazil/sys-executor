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
