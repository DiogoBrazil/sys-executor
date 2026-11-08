<script setup lang="ts">
import type { ErroTela } from "../estado";

defineProps<{ erro: ErroTela & { detalhe?: string }; tipo?: "erro" | "aviso" }>();
</script>

<template>
  <div class="aviso" :class="{ erro: tipo !== 'aviso' }" role="alert" data-teste="aviso-erro">
    <h3>{{ erro.titulo }}</h3>
    <p>{{ erro.mensagem }}</p>
    <p class="acao">{{ erro.acao }}</p>
    <p v-if="erro.detalhe" class="detalhe" data-teste="detalhe-erro">Diagnóstico ({{ erro.codigo }}): {{ erro.detalhe }}</p>
  </div>
</template>

<style scoped>
.detalhe { margin-top: 0.4rem; font-size: var(--t-xs); color: var(--tinta-suave); word-break: break-word; }
</style>
