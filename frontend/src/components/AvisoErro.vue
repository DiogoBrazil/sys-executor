<script setup lang="ts">
import { ref } from "vue";
import type { ErroTela } from "../estado";

const props = defineProps<{ erro: ErroTela & { detalhe?: string }; tipo?: "erro" | "aviso" }>();
const copiado = ref(false);

async function copiar() {
  try {
    await navigator.clipboard.writeText(`Diagnóstico (${props.erro.codigo}): ${props.erro.detalhe}`);
    copiado.value = true;
    setTimeout(() => (copiado.value = false), 2000);
  } catch {
    copiado.value = false;
  }
}
</script>

<template>
  <div class="aviso" :class="{ erro: tipo !== 'aviso' }" role="alert" data-teste="aviso-erro">
    <h3>{{ erro.titulo }}</h3>
    <p>{{ erro.mensagem }}</p>
    <p class="acao">{{ erro.acao }}</p>
    <template v-if="erro.detalhe">
      <p class="detalhe" data-teste="detalhe-erro">Diagnóstico ({{ erro.codigo }}): {{ erro.detalhe }}</p>
      <button class="botao discreto" type="button" data-teste="copiar-diagnostico" @click="copiar">
        {{ copiado ? "Copiado" : "Copiar diagnóstico" }}
      </button>
    </template>
  </div>
</template>

<style scoped>
.detalhe { margin-top: 0.4rem; font-size: var(--t-xs); color: var(--tinta-suave); word-break: break-word; max-height: 9rem; overflow: auto; }
</style>
