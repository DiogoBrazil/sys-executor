// Package win reúne as chamadas específicas do Windows usadas pelo
// sys-executor: ACL do diretório efêmero, identificação de processos vivos
// (varredura de resíduos) e exclusão do Windows Error Reporting (dumps).
//
// Fora do Windows o pacote fica vazio; os chamadores usam arquivos com
// build tags para os equivalentes de desenvolvimento.
package win
