<script setup lang="ts">
// Arquivos de uma peça do cadastro: vários PDFs, na ordem em que entram nos
// autos. O conteúdo fica no Go (sessão cifrada); aqui só nome e páginas.
// Pelo SEI, os documentos separados do PDF do processo podem ser anexados
// daqui (V4 §48.4, revisão de 10/11/2026): os que têm o tipo sugerido igual
// ao da peça aparecem primeiro, mas a escolha é sempre do usuário.
import { computed, ref } from "vue";
import type { DocumentoSEI, PecaCadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

const props = defineProps<{ peca: PecaCadastro; ocupado: boolean; teste: string; documentos?: DocumentoSEI[] }>();
const emit = defineEmits<{ anexar: []; anexarSei: [documentoID: string]; remover: [arquivoID: string]; mover: [arquivoID: string, delta: number] }>();

const ROTULOS: Record<string, string> = {
  MEMORANDO: "Memorando", PORTARIA: "Portaria", DEFESA_PREVIA: "defesa prévia", ALEGACOES_FINAIS: "alegações finais",
  DEPOIMENTO: "depoimento", OUTRO: "outro documento",
};
const combina = (d: DocumentoSEI) => d.sugestao === props.peca.tipo && (props.peca.tipo !== "DEPOIMENTO" || !props.peca.papel || d.papel === props.peca.papel);
const livres = computed(() => {
  const ds = (props.documentos ?? []).filter((d) => !d.usado);
  return [...ds.filter(combina), ...ds.filter((d) => !combina(d))];
});
const escolha = ref("");
function rotuloDoc(d: DocumentoSEI): string {
  const sei = d.sei ? `SEI ${d.sei} – ` : "";
  return `${sei}${d.titulo} (${d.paginas} pág.; sugestão: ${ROTULOS[d.sugestao] ?? d.sugestao})`;
}
function anexarDoSEI() {
  if (!escolha.value) return;
  emit("anexarSei", escolha.value);
  escolha.value = "";
}
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
    <div v-if="livres.length" class="sei">
      <label>Documento do SEI
        <select v-model="escolha" :disabled="ocupado" :data-teste="teste + '-sei'">
          <option value="">Escolha um documento…</option>
          <option v-for="d in livres" :key="d.id" :value="d.id">{{ rotuloDoc(d) }}</option>
        </select>
      </label>
      <button type="button" class="botao" :disabled="ocupado || !escolha" :data-teste="teste + '-anexar-sei'" @click="anexarDoSEI">Anexar do SEI</button>
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
.sei { display: flex; gap: 0.5rem; align-items: end; flex-wrap: wrap; }
.sei label { display: grid; gap: 0.2rem; flex: 1 1 18rem; }
.sei select { max-width: 100%; }
</style>
