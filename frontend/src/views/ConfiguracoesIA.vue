<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { estado, fecharConfigIA, selecionarProvedor } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";
import ConfigChatGPT from "../components/ConfigChatGPT.vue";
import ConfigGemini from "../components/ConfigGemini.vue";

const m = computed(() => estado.ia);
// Serviço usado nas análises: escolha explícita (V4 §48.1); não há troca
// automática entre os serviços.
const provedor = computed(() => m.value?.provedor || (m.value?.modeloSelecionado ? "GEMINI" : ""));
const aba = ref<"CHATGPT" | "GEMINI">("CHATGPT");
watch(
  () => m.value !== null,
  (carregou) => {
    if (carregou && provedor.value === "GEMINI") aba.value = "GEMINI";
  },
  { immediate: true },
);
const opcoes = [
  { id: "CHATGPT", rotulo: "ChatGPT, com o seu plano" },
  { id: "GEMINI", rotulo: "Gemini API, com a sua chave" },
] as const;
</script>

<template>
  <main class="config" data-teste="tela-config-ia">
    <div class="topo">
      <h1>Configurações do assistente</h1>
      <button class="botao" data-teste="fechar-config-ia" @click="fecharConfigIA">Voltar</button>
    </div>
    <p class="intro">
      A leitura e a análise dos autos usam um serviço de IA na internet: o ChatGPT, com o seu plano, ou a Gemini API, com
      a sua chave. Em cada tarefa você confirma o envio antes; nomes e outros dados pessoais reconhecidos são trocados por
      marcadores, e o arquivo PDF não é enviado.
    </p>

    <p v-if="!m" class="nota">Carregando…</p>
    <template v-else>
      <AvisoErro v-if="m.erro" :erro="m.erro" />
      <p v-if="m.ocupado" class="nota" role="status" aria-live="polite" data-teste="ia-ocupado">Consultando o serviço…</p>

      <section class="servico" aria-label="Serviço usado nas análises">
        <h2>Serviço usado nas análises</h2>
        <div class="opcoes" role="radiogroup">
          <label v-for="o in opcoes" :key="o.id" :data-teste="'provedor-' + o.id">
            <input type="radio" name="provedor" :value="o.id" :checked="provedor === o.id" :disabled="m.ocupado" @change="selecionarProvedor(o.id)" />
            {{ o.rotulo }}
          </label>
        </div>
        <p v-if="provedor === 'CHATGPT'" class="selo-uso" data-teste="usando-plano">Usando seu plano ChatGPT</p>
        <p v-else-if="!provedor" class="nota">Nenhum serviço escolhido ainda.</p>
        <p class="nota">A troca de serviço só vale para envios autorizados depois dela: a tarefa em andamento pede nova confirmação.</p>
      </section>

      <div class="abas" role="tablist">
        <button role="tab" :aria-selected="aba === 'CHATGPT'" :class="{ ativa: aba === 'CHATGPT' }" data-teste="aba-chatgpt" @click="aba = 'CHATGPT'">
          ChatGPT
        </button>
        <button role="tab" :aria-selected="aba === 'GEMINI'" :class="{ ativa: aba === 'GEMINI' }" data-teste="aba-gemini" @click="aba = 'GEMINI'">
          Gemini API
        </button>
      </div>
      <ConfigChatGPT v-if="aba === 'CHATGPT'" />
      <ConfigGemini v-else />
    </template>
  </main>
</template>

<style scoped>
.config { max-width: 920px; margin: 0 auto; padding: 2.6rem 2rem; display: grid; gap: 1.4rem; }
.topo { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
.topo h1 { font-size: var(--t-xl); }
.intro { max-width: 68ch; color: var(--tinta-suave); }
.nota { font-size: var(--t-sm); color: var(--tinta-suave); }
.servico { background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); padding: 1.1rem 1.3rem; display: grid; gap: 0.6rem; }
.servico h2 { font-size: var(--t-lg); }
.opcoes { display: flex; flex-wrap: wrap; gap: 1.2rem; }
.opcoes label { display: flex; gap: 0.45rem; align-items: center; }
.selo-uso { justify-self: start; font-size: var(--t-sm); border: 1px solid var(--ok); color: var(--ok); border-radius: 999px; padding: 0.1rem 0.7rem; }
.abas { display: flex; gap: 0.3rem; border-bottom: 1px solid var(--linha-forte); }
.abas button { font: inherit; background: none; border: 1px solid transparent; border-bottom: none; padding: 0.5rem 1rem; border-radius: var(--raio) var(--raio) 0 0; cursor: pointer; color: var(--tinta-suave); }
.abas button.ativa { border-color: var(--linha-forte); background: var(--superficie); color: var(--tinta); font-weight: 600; margin-bottom: -1px; }
</style>
