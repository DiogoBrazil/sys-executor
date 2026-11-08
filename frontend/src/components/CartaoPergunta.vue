<script setup lang="ts">
import { estado, responder } from "../estado";
import type { PerguntaVisao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

defineProps<{ pergunta: PerguntaVisao }>();
</script>

<template>
  <section class="pergunta" role="group" :aria-labelledby="'perg-' + pergunta.id" data-teste="pergunta">
    <h2 :id="'perg-' + pergunta.id">{{ pergunta.titulo }}</h2>
    <p>{{ pergunta.contexto }}</p>
    <div class="opcoes">
      <button
        v-for="(o, i) in pergunta.opcoes"
        :key="o.id"
        class="botao"
        :class="{ primario: i === 0 }"
        :disabled="estado.ocupado"
        :data-teste="'opcao-' + o.id"
        @click="responder(pergunta.id, o.id)"
      >
        {{ o.rotulo }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.pergunta {
  border: 2px solid var(--tinta);
  border-radius: var(--raio);
  background: var(--superficie);
  padding: 1.3rem 1.4rem;
  display: grid;
  gap: 0.7rem;
}
.pergunta h2 { font-size: var(--t-lg); }
.pergunta p { max-width: 68ch; color: var(--tinta-suave); }
.opcoes { display: flex; flex-wrap: wrap; gap: 0.7rem; margin-top: 0.3rem; }
</style>
