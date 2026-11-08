<script setup lang="ts">
import { computed } from "vue";
import { acessarSEI, enviarPDF, estado, voltarAoInicio, voltarParaEntrada } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const aguardandoArquivo = computed(() => estado.visao?.estado === "RECEBENDO_UPLOAD");
</script>

<template>
  <main class="entrada">
    <div class="cabeca">
      <h1>Como você vai trazer os autos?</h1>
      <p>Os dois caminhos levam ao mesmo processamento.</p>
    </div>

    <div v-if="!aguardandoArquivo" class="opcoes">
      <button class="opcao" :disabled="estado.ocupado" data-teste="entrada-SEI" @click="acessarSEI">
        <span class="nome">Acessar o SEI/RO</span>
        <span class="desc">
          Você entra com seu usuário, senha e código de verificação. O aplicativo localiza o processo e
          gera o PDF completo. A senha não é guardada.
        </span>
        <span class="req">Requer internet</span>
      </button>
      <button class="opcao" :disabled="estado.ocupado" data-teste="entrada-UPLOAD" @click="enviarPDF">
        <span class="nome">Enviar o PDF do processo</span>
        <span class="desc">
          Use o PDF completo que você já gerou no SEI/RO ("Gerar Arquivo PDF do Processo", com todos os
          documentos). Não pede senha.
        </span>
        <span class="req">Funciona sem internet</span>
      </button>
    </div>

    <section v-else class="arquivo" data-teste="aguardando-pdf">
      <h2>Selecione o PDF completo do processo</h2>
      <p>O arquivo é copiado de forma cifrada para a área temporária desta tarefa e apagado ao final.</p>
      <div class="acoes">
        <button class="botao primario" :disabled="estado.ocupado" data-teste="selecionar-pdf" @click="enviarPDF">
          Selecionar PDF
        </button>
        <button class="botao" :disabled="estado.ocupado" @click="voltarParaEntrada">Escolher outro caminho</button>
      </div>
    </section>

    <AvisoErro v-if="estado.erro" :erro="estado.erro" />
    <AvisoErro v-else-if="aguardandoArquivo && estado.visao?.erro" :erro="estado.visao.erro" data-teste="erro-arquivo" />

    <div>
      <button class="botao discreto" :disabled="estado.ocupado" data-teste="voltar-inicio" @click="voltarAoInicio">
        Trocar o tipo de minuta
      </button>
    </div>
  </main>
</template>

<style scoped>
.entrada {
  max-width: 980px;
  margin: 0 auto;
  padding: 2.6rem 2rem 2rem;
  display: grid;
  gap: 1.8rem;
}
.cabeca h1 { font-size: var(--t-xl); }
.cabeca p { margin-top: 0.4rem; color: var(--tinta-suave); }

.opcoes { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1.2rem; }
.opcao {
  text-align: left;
  display: grid;
  gap: 0.6rem;
  align-content: start;
  padding: 1.4rem 1.5rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
}
.opcao:hover:not(:disabled) { border-color: var(--tinta); }
.nome { font-family: var(--serif); font-size: var(--t-lg); font-weight: 600; }
.desc { color: var(--tinta-suave); max-width: 52ch; }
.req { font-size: var(--t-sm); color: var(--tinta-suave); border-top: 1px solid var(--linha); padding-top: 0.6rem; }

.arquivo {
  display: grid;
  gap: 0.8rem;
  padding: 1.6rem;
  background: var(--superficie);
  border: 1px dashed var(--linha-forte);
  border-radius: var(--raio);
}
.arquivo h2 { font-size: var(--t-lg); }
.arquivo p { color: var(--tinta-suave); max-width: 64ch; }
.acoes { display: flex; gap: 0.8rem; flex-wrap: wrap; margin-top: 0.4rem; }
</style>
