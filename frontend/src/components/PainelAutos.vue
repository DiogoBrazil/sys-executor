<script setup lang="ts">
import { computed } from "vue";
import Icone from "./ui/Icone.vue";
import type { ResumoAutos } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

const props = defineProps<{ autos: ResumoAutos }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

const ROTULO_SITUACAO: Record<string, string> = {
  LIDA: "lidas",
  OCR_OK: "por OCR",
  OCR_BAIXO: "OCR de baixa qualidade",
  ILEGIVEL: "ilegíveis",
  VAZIA: "em branco",
};

const cobertura = computed(() => {
  const c: Record<string, number> = {};
  for (const p of props.autos.paginas ?? []) c[p.situacao] = (c[p.situacao] ?? 0) + 1;
  return Object.keys(ROTULO_SITUACAO)
    .filter((k) => c[k])
    .map((k) => ({ chave: k, n: c[k], rotulo: ROTULO_SITUACAO[k] }));
});

const problemas = computed(() =>
  (props.autos.paginas ?? []).filter((p) => p.situacao === "ILEGIVEL" || p.situacao === "OCR_BAIXO"),
);

// Menções agrupadas por peça: termos distintos, contagem e a 1ª ocorrência.
const gruposMidia = computed(() => {
  const grupos: { peca: string; rotulo: string; paginas: string; termos: string[]; total: number; pagina: number; inicio: number; fim: number }[] = [];
  const porPeca = new Map<string, (typeof grupos)[number]>();
  for (const m of props.autos.midias ?? []) {
    let g = porPeca.get(m.peca);
    if (!g) {
      const p = (props.autos.pecas ?? []).find((x) => x.id === m.peca);
      g = {
        peca: m.peca,
        rotulo: p?.rotulo ?? "Peça",
        paginas: p ? paginas(p.paginaInicial, p.paginaFinal) : `p. ${m.pagina}`,
        termos: [],
        total: 0,
        pagina: m.pagina,
        inicio: m.inicio,
        fim: m.fim,
      };
      porPeca.set(m.peca, g);
      grupos.push(g);
    }
    g.total++;
    const t = m.termo.toLowerCase();
    if (!g.termos.includes(t)) g.termos.push(t);
  }
  return grupos;
});

function paginas(a: number, b: number) {
  return a === b ? `p. ${a}` : `pp. ${a}–${b}`;
}
</script>

<template>
  <section class="autos" data-teste="painel-autos">
    <div class="cabeca">
      <p><strong>{{ autos.totalPaginas }} páginas</strong> no total. Clique numa peça para abri-la no PDF.</p>
    </div>

    <ul class="cobertura" aria-label="Cobertura da leitura" data-teste="cobertura">
      <li v-for="c in cobertura" :key="c.chave" :class="c.chave">{{ c.n }} {{ c.rotulo }}</li>
    </ul>
    <p v-if="problemas.length" class="alerta" data-teste="paginas-problema">
      Páginas com leitura incerta:
      <button v-for="p in problemas" :key="p.numero" class="link" @click="emit('abrir', p.numero, 0, 0)">{{ p.numero }}</button>
    </p>

    <h3>Peças identificadas</h3>
    <ol class="pecas" data-teste="pecas">
      <li v-for="p in autos.pecas" :key="p.id" :class="{ central: p.id === autos.pecaCentral }">
        <button class="item" @click="emit('abrir', p.paginaInicial, 0, 0)">
          <Icone nome="pdf" class="ic" />
          <span class="rotulo">
            {{ p.rotulo }}
            <em v-if="p.provavel" class="selo">provável, pela ordem dos autos</em>
            <em v-if="p.id === autos.pecaCentral" class="selo central">peça central</em>
          </span>
          <span class="meta">
            {{ paginas(p.paginaInicial, p.paginaFinal) }}<template v-if="p.sei"> · SEI {{ p.sei }}</template
            ><template v-else> · documento externo</template>
          </span>
          <span v-if="p.situacao === 'ILEGIVEL' || p.situacao === 'OCR_BAIXO'" class="selo alerta">leitura incerta</span>
        </button>
      </li>
    </ol>

    <template v-if="autos.midias?.length">
      <h3>Mídias mencionadas</h3>
      <p class="nota">
        Gravações, vídeos e links citados não fazem parte do PDF. O aplicativo vai pedir uma descrição ou
        transcrição antes de usar essas provas.
      </p>
      <ul class="midias" data-teste="midias">
        <li v-for="g in gruposMidia" :key="g.peca">
          <button class="item" @click="emit('abrir', g.pagina, g.inicio, g.fim)">
            <Icone nome="relogio" class="ic" />
            <span class="rotulo">{{ g.rotulo }}</span>
            <span class="meta">{{ g.paginas }} — {{ g.termos.slice(0, 4).join(", ") }}<template v-if="g.termos.length > 4">…</template> · {{ g.total }} {{ g.total === 1 ? "menção" : "menções" }}</span>
          </button>
        </li>
      </ul>
    </template>
  </section>
</template>

<style scoped>
.autos { display: grid; gap: 0.8rem; }
.cabeca p { color: var(--tinta-suave); font-size: var(--t-sm); }
.cabeca strong { color: var(--tinta); }
h3 { font-family: var(--sans); font-size: var(--t-md); font-weight: 600; margin-top: 0.4rem; }
.cobertura { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: 0.5rem; }
.cobertura li { font-size: var(--t-sm); padding: 0.2rem 0.65rem; border-radius: 999px; background: var(--ok-claro); color: var(--ok); border: 1px solid #c3dfc9; }
.cobertura li.ILEGIVEL, .cobertura li.OCR_BAIXO { border-color: var(--alerta-linha); color: var(--alerta); background: var(--alerta-claro); }
.cobertura li.VAZIA { background: var(--superficie-2); color: var(--tinta-suave); border-color: var(--linha); }
.alerta { color: var(--alerta); font-size: var(--t-sm); }
.link { border: 0; background: none; color: var(--foco); text-decoration: underline; padding: 0 0.2rem; }
.pecas, .midias {
  list-style: none; margin: 0; padding: 0; display: grid; gap: 0.5rem;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 19rem), 1fr));
}
.item {
  width: 100%; height: 100%;
  text-align: left;
  display: grid;
  grid-template-columns: auto 1fr;
  align-content: start;
  gap: 0.15rem 0.6rem;
  padding: 0.65rem 0.8rem;
  border: 1px solid var(--linha);
  border-radius: var(--r-sm);
  background: var(--superficie-2);
  transition: border-color var(--dur-1) var(--ease), background var(--dur-1) var(--ease);
}
.item > :not(.ic) { grid-column: 2; }
.ic { grid-row: 1 / span 3; color: var(--acento); margin-top: 0.1rem; }
.item:hover { border-color: var(--acento); background: var(--acento-claro); }
.central .item { border-color: var(--acento); border-width: 1px 1px 1px 4px; }
.rotulo { font-weight: 500; }
.meta { color: var(--tinta-suave); font-size: var(--t-sm); }
.selo { font-style: normal; font-size: var(--t-xs); color: var(--tinta-suave); margin-left: 0.4rem; }
.selo.central { color: var(--acento); font-weight: 600; }
.selo.alerta { color: var(--alerta); }
.nota { color: var(--tinta-suave); font-size: var(--t-sm); max-width: 80ch; }
</style>
