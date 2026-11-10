<script setup lang="ts">
import { computed } from "vue";
import { acessarSEI, cadastrarPecas, estado, voltarAoInicio } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";
import PainelSEI from "../components/PainelSEI.vue";
import Montagem from "./Montagem.vue";

const aguardandoArquivo = computed(() => estado.visao?.estado === "RECEBENDO_UPLOAD");
const emCadastro = computed(() => aguardandoArquivo.value && !!estado.visao?.cadastro);
const noSEI = computed(() => !!estado.visao?.estado.startsWith("SEI_"));
</script>

<template>
  <main class="entrada">
    <div class="cabeca">
      <h1>Como você vai trazer os autos?</h1>
      <p>Os dois caminhos levam ao mesmo processamento.</p>
    </div>

    <PainelSEI v-if="noSEI" />

    <div v-else-if="!aguardandoArquivo" class="opcoes">
      <button class="opcao" :disabled="estado.ocupado" data-teste="entrada-SEI" @click="acessarSEI">
        <span class="nome">Acessar o SEI/RO</span>
        <span class="desc">
          Você entra com seu usuário, senha e código de verificação. O aplicativo localiza o processo e
          gera o PDF completo. A senha não é guardada.
        </span>
        <span class="req">Requer internet</span>
      </button>
      <button class="opcao" :disabled="estado.ocupado" data-teste="entrada-UPLOAD" @click="cadastrarPecas">
        <span class="nome">Cadastrar as peças</span>
        <span class="desc">
          Você indica cada peça (Memorando ou Portaria, defesa, depoimentos, alegações e outros documentos) e anexa
          os PDFs de cada uma. Não pede senha.
        </span>
        <span class="req">A análise por IA requer internet</span>
      </button>
    </div>

    <Montagem v-else-if="emCadastro" :key="estado.visao!.id" />

    <template v-if="!emCadastro">
      <AvisoErro v-if="estado.erro" :erro="estado.erro" />
      <AvisoErro v-else-if="!noSEI && estado.visao?.erro" :erro="estado.visao.erro" data-teste="erro-arquivo" />
    </template>

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

</style>
