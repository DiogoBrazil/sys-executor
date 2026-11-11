<script setup lang="ts">
// Arquivos de uma peça do cadastro: vários PDFs, na ordem em que entram nos
// autos. O conteúdo fica no Go (sessão cifrada); aqui só nome e páginas.
// Pelo SEI, os documentos separados do PDF do processo podem ser anexados
// daqui (V4 §48.4, revisão de 10/11/2026): os que têm o tipo sugerido igual
// ao da peça aparecem primeiro, mas a escolha é sempre do usuário.
// compacto: para as linhas da tabela de depoimentos e documentos.
// carregando: o anexo em curso (escolha dos arquivos, cópia e leitura).
import { computed, ref } from "vue";
import type { DocumentoSEI, PecaCadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import Icone from "./ui/Icone.vue";

const props = defineProps<{ peca: PecaCadastro; ocupado: boolean; teste: string; documentos?: DocumentoSEI[]; compacto?: boolean; carregando?: boolean }>();
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
const tem = computed(() => (props.peca.arquivos?.length ?? 0) > 0);
</script>

<template>
  <div class="arquivos" :class="{ compacto }" :data-teste="teste">
    <!-- compacto: um selo por arquivo -->
    <template v-if="compacto">
      <span v-for="a in peca.arquivos" :key="a.id" class="arq-selo" :title="'Resumo do hash: ' + a.hash" :data-teste="teste + '-arquivo'">
        <Icone nome="check" :tamanho="14" /><span class="nome">{{ a.nome }}</span><span class="pag">· {{ a.paginas }} pág.</span>
        <button type="button" class="tira" :disabled="ocupado" :aria-label="'Remover ' + a.nome" :data-teste="teste + '-remover'" @click="emit('remover', a.id)">
          <Icone nome="fechar" :tamanho="13" />
        </button>
      </span>
      <span v-if="carregando" class="arq-carregando" role="status"><span class="anel" aria-hidden="true"></span>Carregando PDF…</span>
      <button v-else type="button" class="zona mini" :disabled="ocupado" :data-teste="teste + '-anexar'" @click="emit('anexar')">
        <Icone nome="mais" :tamanho="15" />{{ tem ? "Mais PDF" : "Anexar PDF" }}
      </button>
    </template>

    <!-- completo: lista de arquivos e zona de anexo -->
    <template v-else>
      <ol v-if="tem" class="lista">
        <li v-for="(a, i) in peca.arquivos" :key="a.id" :data-teste="teste + '-arquivo'">
          <Icone nome="pdf" :tamanho="20" class="ic" />
          <span class="nome-bloco"><strong class="nome" :title="'Resumo do hash: ' + a.hash">{{ a.nome }}</strong><span class="pag">{{ a.paginas }} {{ a.paginas === 1 ? "página" : "páginas" }}</span></span>
          <span class="acoes">
            <button type="button" class="botao icone" :disabled="ocupado || i === 0" :aria-label="'Subir ' + a.nome" title="Subir" @click="emit('mover', a.id, -1)"><Icone nome="subir" :tamanho="16" /></button>
            <button type="button" class="botao icone" :disabled="ocupado || i === peca.arquivos.length - 1" :aria-label="'Descer ' + a.nome" title="Descer" @click="emit('mover', a.id, 1)"><Icone nome="descer" :tamanho="16" /></button>
            <button type="button" class="botao icone" :disabled="ocupado" :aria-label="'Remover ' + a.nome" title="Remover" :data-teste="teste + '-remover'" @click="emit('remover', a.id)"><Icone nome="lixeira" :tamanho="16" /></button>
          </span>
        </li>
      </ol>
      <div v-if="carregando" class="linha-carregando" role="status">
        <span class="anel" aria-hidden="true"></span>
        <span><strong>Anexando PDF…</strong><br /><span class="pag">Escolha os arquivos na janela que se abriu; em seguida eles são copiados e as páginas contadas.</span></span>
        <span class="corre" aria-hidden="true"></span>
      </div>
      <button v-else type="button" class="zona" :class="{ vazia: !tem }" :disabled="ocupado" :data-teste="teste + '-anexar'" @click="emit('anexar')">
        <Icone :nome="tem ? 'mais' : 'pdf'" :tamanho="20" />
        <span>{{ tem ? "Anexar mais PDFs" : "Anexar PDF(s)" }}</span>
        <small v-if="!tem">Nenhum arquivo anexado.</small>
      </button>
    </template>

    <div v-if="livres.length" class="sei">
      <label>
        <span :class="{ 'visualmente-oculto': compacto }">Documento do SEI</span>
        <select v-model="escolha" :disabled="ocupado" :data-teste="teste + '-sei'">
          <option value="">{{ compacto ? "Documento do SEI…" : "Escolha um documento…" }}</option>
          <option v-for="d in livres" :key="d.id" :value="d.id">{{ rotuloDoc(d) }}</option>
        </select>
      </label>
      <button type="button" class="botao" :disabled="ocupado || !escolha" :data-teste="teste + '-anexar-sei'" @click="anexarDoSEI">Anexar do SEI</button>
    </div>
  </div>
</template>

<style scoped>
.arquivos { display: grid; gap: 0.5rem; }
.lista { margin: 0; padding: 0; list-style: none; display: grid; gap: 0.4rem; }
.lista li {
  display: flex; gap: 0.7rem; align-items: center;
  padding: 0.5rem 0.4rem 0.5rem 0.8rem; border: 1px solid var(--linha); border-radius: var(--r-md); background: var(--superficie-2);
}
.ic { color: var(--acento); flex: none; }
.nome-bloco { flex: 1; min-width: 0; display: grid; }
.nome { font-weight: 500; overflow-wrap: anywhere; font-size: var(--t-sm); }
.pag { font-size: var(--t-xs); color: var(--tinta-suave); }
.acoes { display: flex; gap: 0.1rem; }
.acoes .botao.icone { min-height: 2rem; width: 2rem; }
.zona {
  display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 0.3rem 0.5rem;
  padding: 0.7rem 0.8rem; border: 1.5px dashed var(--linha-forte); border-radius: var(--r-md);
  background: transparent; color: var(--acento); font-weight: 600; font-size: var(--t-sm); min-height: 2.9rem;
  transition: border-color var(--dur-1) var(--ease), background var(--dur-1) var(--ease);
}
.zona.vazia { padding: 1.2rem 0.8rem; }
.zona small { flex-basis: 100%; text-align: center; color: var(--tinta-suave); font-weight: 400; }
.zona:hover:not(:disabled) { border-color: var(--acento); background: var(--acento-claro); }
.zona:disabled { opacity: 0.55; }
.zona.mini { padding: 0.45rem 0.7rem; min-height: 2.6rem; justify-content: flex-start; }
.linha-carregando {
  position: relative; overflow: hidden;
  display: flex; align-items: center; gap: 0.7rem;
  padding: 0.65rem 0.8rem; border: 1px solid var(--alerta-linha); background: var(--alerta-claro); border-radius: var(--r-md); font-size: var(--t-sm);
}
.anel { width: 1.2rem; height: 1.2rem; border-radius: 50%; border: 3px solid var(--alerta-linha); border-top-color: var(--acento); animation: gira 0.8s linear infinite; flex: none; }
.corre { position: absolute; left: 0; bottom: 0; height: 3px; width: 40%; background: var(--acento); animation: corre 1.4s ease-in-out infinite; }
@keyframes gira { to { transform: rotate(360deg); } }
@keyframes corre { from { transform: translateX(-100%); } to { transform: translateX(250%); } }
.compacto { display: flex; flex-wrap: wrap; gap: 0.35rem; align-items: center; }
.arq-selo {
  display: inline-flex; align-items: center; gap: 0.3rem; max-width: 100%;
  border: 1px solid #c3dfc9; background: var(--ok-claro); color: var(--ok); border-radius: var(--r-sm); padding: 0.3rem 0.3rem 0.3rem 0.55rem; font-size: var(--t-xs);
}
.arq-selo .nome { font-size: var(--t-xs); font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 10.5rem; }
.arq-selo .pag { color: var(--ok); }
.tira { border: 0; background: transparent; color: var(--ok); width: 1.5rem; height: 1.5rem; border-radius: 4px; display: grid; place-items: center; }
.tira:hover:not(:disabled) { background: #d4ead9; }
.arq-carregando {
  display: inline-flex; align-items: center; gap: 0.45rem; font-size: var(--t-xs); color: var(--alerta);
  border: 1px solid var(--alerta-linha); background: var(--alerta-claro); border-radius: var(--r-sm); padding: 0.45rem 0.6rem;
}
.arq-carregando .anel { width: 0.95rem; height: 0.95rem; border-width: 2.5px; }
.sei { display: flex; gap: 0.5rem; align-items: end; flex-wrap: wrap; flex-basis: 100%; }
.sei label { display: grid; gap: 0.2rem; flex: 1 1 14rem; font-size: var(--t-sm); }
.sei select { max-width: 100%; }
</style>
