<script setup lang="ts">
// Arquivos de uma peça do cadastro: vários PDFs, na ordem em que entram nos
// autos. O conteúdo fica no Go (sessão cifrada); aqui só nome e páginas.
import type { PecaCadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

defineProps<{ peca: PecaCadastro; ocupado: boolean; teste: string }>();
const emit = defineEmits<{ anexar: []; remover: [arquivoID: string]; mover: [arquivoID: string, delta: number] }>();
</script>

<template>
  <div class="arquivos" :data-teste="teste">
    <ol v-if="peca.arquivos?.length" class="lista">
      <li v-for="(a, i) in peca.arquivos" :key="a.id" :data-teste="teste + '-arquivo'">
        <span class="nome" :title="'Resumo do hash: ' + a.hash">{{ a.nome }}</span>
        <span class="paginas">{{ a.paginas }} pág.</span>
        <span class="acoes">
          <button type="button" class="botao discreto" :disabled="ocupado || i === 0" :aria-label="'Subir ' + a.nome" @click="emit('mover', a.id, -1)">↑</button>
          <button type="button" class="botao discreto" :disabled="ocupado || i === peca.arquivos.length - 1" :aria-label="'Descer ' + a.nome" @click="emit('mover', a.id, 1)">↓</button>
          <button type="button" class="botao discreto" :disabled="ocupado" :data-teste="teste + '-remover'" @click="emit('remover', a.id)">Remover</button>
        </span>
      </li>
    </ol>
    <p v-else class="vazio">Nenhum arquivo anexado.</p>
    <div>
      <button type="button" class="botao" :disabled="ocupado" :data-teste="teste + '-anexar'" @click="emit('anexar')">
        {{ peca.arquivos?.length ? "Anexar mais PDFs" : "Anexar PDF(s)" }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.arquivos { display: grid; gap: 0.5rem; }
.lista { margin: 0; padding-left: 1.4rem; display: grid; gap: 0.25rem; }
.lista li { display: flex; gap: 0.6rem; align-items: center; flex-wrap: wrap; }
.nome { font-weight: 500; overflow-wrap: anywhere; }
.paginas { color: var(--tinta-suave); font-size: var(--t-sm); }
.acoes { margin-left: auto; display: flex; gap: 0.1rem; }
.vazio { color: var(--tinta-suave); font-size: var(--t-sm); }
</style>
