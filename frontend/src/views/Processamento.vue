<script setup lang="ts">
import { computed, defineAsyncComponent, h, ref } from "vue";
import { abrirConfigIA, cancelar, estado, retomar, revisar, voltarAoInicio } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";
import CartaoPergunta from "../components/CartaoPergunta.vue";
import MinutaPrevia from "../components/MinutaPrevia.vue";
import PainelAnalise from "../components/PainelAnalise.vue";
import PainelAutos from "../components/PainelAutos.vue";
import Carregando from "../components/ui/Carregando.vue";
import Figura from "../components/ui/Figura.vue";
import Icone from "../components/ui/Icone.vue";
import Janela from "../components/ui/Janela.vue";
// O PDF.js só é carregado quando o usuário abre os autos; enquanto o
// componente chega, um indicador ocupa a tela (pedido do usuário, 11/11/2026).
const VisualizadorFontes = defineAsyncComponent({
  loader: () => import("../components/VisualizadorFontes.vue"),
  loadingComponent: {
    render: () => h("div", { class: "carga-visualizador" }, [h(Carregando, { texto: "Abrindo os autos…", sobreposto: true })]),
  },
  delay: 0,
});

const v = computed(() => estado.visao!);
const confirmando = ref(false);
const fonte = ref<{ pagina: number; inicio: number; fim: number } | null>(null);
function abrir(pagina: number, inicio: number, fim: number) {
  fonte.value = { pagina, inicio, fim };
}

// Gavetas e modais (o que alongava a página fica fora da tela principal).
const verAnalise = ref(false);
const verRespostas = ref(false);
const verAutos = ref(false);
function abrirFonteDe(pagina: number, inicio: number, fim: number) {
  verAnalise.value = false;
  verAutos.value = false;
  abrir(pagina, inicio, fim);
}
async function alterar(id: string) {
  verRespostas.value = false;
  await revisar(id);
}

// Falhas do serviço de IA (chave, conta, modelo, cota, conexão): atalho para as configurações.
const erroDeIA = computed(() => (v.value.erro?.codigo ?? "").startsWith("IA-"));

// A minuta validada aguarda o usuário (V4 §31.3).
const emRevisao = computed(() => v.value.estado === "AGUARDANDO_REVISAO");

const percentual = computed(() => {
  const p = v.value.progresso;
  return p && p.total > 0 ? Math.min(100, Math.round((p.atual / p.total) * 100)) : null;
});

// Robô quando a IA trabalha; engrenagens nas etapas feitas neste computador.
const ESTADOS_IA = new Set(["DELIMITANDO", "ANALISANDO", "REDIGINDO", "AJUSTANDO", "VALIDANDO"]);
const figura = computed(() => (ESTADOS_IA.has(v.value.estado) ? "robo" : "engrenagem"));

// Etapa ativa: gira enquanto há trabalho; pulsa quando espera o usuário.
const esperaUsuario = computed(() => !!v.value.pergunta || emRevisao.value || !!v.value.parada);
const etapaAtiva = computed(() => v.value.etapas?.find((m) => m.situacao === "ativa")?.rotulo ?? "");
// Parada com erro (retomável ou não): a etapa ativa aparece como "parou".
const parouComErro = computed(() => !!v.value.parada && !!v.value.erro && v.value.erro.codigo !== "TAR-004");
const etapaFalhou = computed(() => parouComErro.value || (v.value.etapas ?? []).some((m) => m.situacao === "falhou"));
const resumoAndamento = computed(() => {
  const es = v.value.etapas ?? [];
  const i = es.findIndex((m) => m.situacao === "ativa" || m.situacao === "falhou");
  if (i < 0) return "";
  return etapaFalhou.value ? `parado na etapa ${i + 1} de ${es.length}` : `etapa ${i + 1} de ${es.length}`;
});

const totalAnalise = computed(() => (v.value.painel?.secoes ?? []).reduce((s, x) => s + x.itens.length, 0));
const totalRespostas = computed(() => v.value.respostas?.length ?? 0);

const tituloFinal = computed(() => {
  switch (v.value.estado) {
    case "FINALIZADA":
      return "Tarefa finalizada";
    case "CANCELADA":
      return "Tarefa cancelada";
    default:
      return "A tarefa não pôde continuar";
  }
});

async function confirmarCancelamento() {
  confirmando.value = false;
  await cancelar();
}
</script>

<template>
  <main class="proc" :class="{ fim: v.terminal }">
    <!-- Andamento: trilha horizontal com as consultas na mesma faixa -->
    <nav v-if="!v.terminal" class="andamento" aria-label="Andamento da tarefa">
      <div class="topo-and">
        <strong>Andamento</strong>
        <span class="sub" :class="{ parou: etapaFalhou }">{{ resumoAndamento }}</span>
        <span class="espaco"></span>
        <button v-if="v.painel" type="button" class="botao ferramenta" data-teste="abrir-analise" @click="verAnalise = true">
          <Icone nome="lampada" :tamanho="16" />Análise<span v-if="totalAnalise" class="conta">{{ totalAnalise }}</span>
        </button>
        <button v-if="totalRespostas" type="button" class="botao ferramenta" data-teste="abrir-respostas" @click="verRespostas = true">
          <Icone nome="check" :tamanho="16" />Suas respostas<span class="conta">{{ totalRespostas }}</span>
        </button>
        <button v-if="v.autos" type="button" class="botao ferramenta" data-teste="abrir-autos-lidos" @click="verAutos = true">
          <Icone nome="pasta" :tamanho="16" />Autos lidos
        </button>
        <button v-if="v.autos" type="button" class="botao ferramenta" data-teste="consultar-autos" @click="abrir(1, 0, 0)">
          <Icone nome="pdf" :tamanho="16" />Consultar os autos
        </button>
        <button v-if="!confirmando" type="button" class="botao ferramenta perigo" data-teste="cancelar" @click="confirmando = true">
          Cancelar tarefa
        </button>
      </div>
      <div v-if="confirmando" class="confirma" role="alert">
        <p>Os arquivos temporários desta tarefa serão descartados. Não há como retomar depois.</p>
        <button class="botao perigo confirmar" data-teste="confirmar-cancelar" @click="confirmarCancelamento">Cancelar e descartar</button>
        <button class="botao" @click="confirmando = false">Continuar a tarefa</button>
      </div>
      <ol class="trilha" :class="{ trabalhando: !esperaUsuario, parada: parouComErro }" aria-label="Etapas da tarefa" data-teste="linha-do-tempo">
        <li v-for="(m, i) in v.etapas" :key="m.chave" :class="m.situacao" :aria-current="m.situacao === 'ativa' ? 'step' : undefined">
          <span class="marca" aria-hidden="true">
            <svg v-if="m.situacao === 'concluida'" viewBox="0 0 24 24" class="check"><path d="M6 12.5l4 4L18 8" /></svg>
            <template v-else-if="m.situacao === 'falhou'">!</template>
            <template v-else>{{ i + 1 }}</template>
          </span>
          <span class="rotulo">{{ m.rotulo }}</span>
          <span v-if="m.situacao === 'ativa' && parouComErro" class="selo-etapa parou">parou</span>
          <span v-else-if="m.situacao === 'ativa'" class="selo-etapa">{{ esperaUsuario ? "sua vez" : "em andamento" }}</span>
          <span v-else-if="m.situacao === 'falhou'" class="selo-etapa parou">parou</span>
          <span class="visualmente-oculto">({{ m.situacao }})</span>
        </li>
      </ol>
    </nav>

    <!-- Fim da tarefa (finalizada, cancelada ou falha) -->
    <section v-if="v.terminal" class="cartao-fim" :class="v.estado.toLowerCase()">
      <span class="selo-fim" aria-hidden="true">
        <svg v-if="v.estado === 'FINALIZADA'" viewBox="0 0 24 24" class="check-fim"><path d="M5 12.5l4.2 4.2L19 7" /></svg>
        <Icone v-else :nome="v.estado === 'FALHA' ? 'alerta' : 'fechar'" :tamanho="36" />
      </span>
      <h1>{{ tituloFinal }}</h1>
      <p v-if="v.estado === 'FINALIZADA'" class="nota" data-teste="finalizada">
        Os autos, a análise e a minuta foram descartados deste computador. Os arquivos que você salvou permanecem onde os gravou.
      </p>
      <AvisoErro v-if="v.erro && v.estado === 'FALHA'" :erro="v.erro" />
      <p v-else-if="v.erro" class="nota">{{ v.erro.mensagem }}</p>
      <ul v-if="v.estado === 'FINALIZADA'" class="blocos-fim">
        <li><span>Minuta</span><strong>Aprovada por você</strong></li>
        <li><span>Arquivos salvos</span><strong>Onde você os gravou</strong></li>
        <li><span>Dados da tarefa</span><strong>Apagados deste computador</strong></li>
      </ul>
      <button class="botao primario grande" data-teste="voltar-inicio" @click="voltarAoInicio">
        {{ v.estado === "FINALIZADA" ? "Iniciar outra minuta" : "Voltar ao início" }}
      </button>
      <p v-if="v.estado === 'FINALIZADA'" class="nota pequena">A revisão, a aprovação e a decisão final cabem à autoridade competente.</p>
    </section>

    <section v-else class="painel">
      <h1 v-if="emRevisao">Minuta pronta para revisão</h1>
      <h1 v-else class="visualmente-oculto">Processando os autos</h1>

      <Transition name="folha" mode="out-in">
        <CartaoPergunta v-if="v.pergunta" :key="v.pergunta.id" :pergunta="v.pergunta" :etapa="etapaAtiva" @abrir="abrir" />
      </Transition>

      <!-- Parada com erro: o que houve e o que fazer -->
      <section v-if="v.parada && v.erro && v.erro.codigo !== 'TAR-004'" class="cartao-parada" role="alert"
        :data-teste="v.retomavel ? 'parada-recuperavel' : 'parada-erro'">
        <span class="selo-parada" aria-hidden="true"><Icone nome="alerta" :tamanho="30" /></span>
        <div class="texto-parada">
          <span class="chapeu">{{ v.retomavel ? "A tarefa parou · dá para continuar" : "A tarefa parou" }}</span>
          <h2>{{ v.erro.titulo }}</h2>
          <p>{{ v.erro.mensagem }}</p>
          <p v-if="v.erro.acao" class="o-que-fazer"><strong>O que fazer:</strong> {{ v.erro.acao }}
            <template v-if="v.retomavel"> Suas respostas e as etapas concluídas continuam guardadas nesta tarefa.</template></p>
          <p v-if="v.erro.detalhe" class="detalhe">Diagnóstico ({{ v.erro.codigo }}): {{ v.erro.detalhe }}</p>
          <div class="acoes">
            <button v-if="v.retomavel" class="botao primario" :disabled="estado.ocupado" data-teste="retomar" @click="retomar">Tentar novamente</button>
            <button v-if="erroDeIA" class="botao" data-teste="abrir-config-ia" @click="abrirConfigIA">Abrir configurações de IA</button>
            <span class="espaco"></span>
            <span class="codigo">Código {{ v.erro.codigo }}</span>
          </div>
        </div>
      </section>
      <div v-else-if="v.parada && v.retomavel" class="retomar" data-teste="parada-recuperavel">
        <div class="acoes">
          <button class="botao primario" :disabled="estado.ocupado" data-teste="retomar" @click="retomar">Tentar novamente</button>
        </div>
      </div>
      <div v-else-if="v.parada" class="callout info" role="status" data-teste="parada">
        <Icone nome="info" class="callout-icone" />
        <h3>As próximas etapas ainda não estão disponíveis</h3>
        <p>
          A redação da minuta deste procedimento chega nas próximas versões do aplicativo.
          Você pode conferir a análise, alterar respostas, consultar os autos e cancelar a tarefa quando terminar.
        </p>
      </div>

      <div v-else-if="emRevisao" class="callout" role="status" data-teste="em-revisao">
        <Icone nome="alerta" class="callout-icone" />
        <h3>Leia a minuta inteira antes de usar</h3>
        <p>
          Ela passou pela validação automática, que não substitui a sua revisão. Você pode salvar o DOCX e o PDF, pedir
          ajustes quantas vezes precisar e alterar respostas. A tarefa só termina quando você mandar finalizar.
        </p>
      </div>

      <div v-else-if="!v.pergunta" class="agora" aria-live="polite" data-teste="estado-atual">
        <Figura :tipo="figura" :tamanho="104" />
        <div class="texto">
          <span class="quem" :class="figura">
            <template v-if="figura === 'robo'">A IA está trabalhando</template>
            <template v-else>Trabalhando neste computador</template>
          </span>
          <p class="descricao">{{ v.descricao }}</p>
          <p v-if="v.atividade" class="sub" data-teste="atividade-texto">{{ v.atividade }}</p>
          <div v-if="percentual !== null" class="medida" data-teste="progresso">
            <div class="barra" role="progressbar" :aria-valuenow="percentual" aria-valuemin="0" aria-valuemax="100" :aria-label="v.descricao">
              <span :style="{ width: percentual + '%' }"></span>
            </div>
            <p class="conta-progresso">{{ v.progresso!.atual }} de {{ v.progresso!.total }} {{ v.progresso!.unidade }}</p>
          </div>
          <div v-else class="atividade" role="status" data-teste="atividade">
            <span class="ponto"></span><span class="ponto"></span><span class="ponto"></span>
            <span class="visualmente-oculto">Em andamento</span>
          </div>
        </div>
      </div>

      <AvisoErro v-if="v.erro && !v.parada" :erro="v.erro" tipo="aviso" />
      <AvisoErro v-if="v.aviso" :erro="v.aviso" tipo="aviso" />
      <AvisoErro v-if="estado.erro" :erro="estado.erro" />

      <MinutaPrevia v-if="v.minuta" :minuta="v.minuta" :em-revisao="emRevisao" @abrir="abrir" />
    </section>

    <Janela v-if="v.painel && !v.terminal" :aberta="verAnalise" titulo="Análise" tipo="gaveta" largura="g" @fechar="verAnalise = false">
      <PainelAnalise :painel="v.painel" @abrir="abrirFonteDe" />
    </Janela>

    <Janela v-if="totalRespostas && !v.terminal" :aberta="verRespostas" titulo="Suas respostas" tipo="gaveta" @fechar="verRespostas = false">
      <section class="respostas" aria-label="Respostas dadas" data-teste="respostas">
        <p class="nota">Ao alterar uma resposta, as etapas que dependem dela são refeitas.</p>
        <ul role="list">
          <li v-for="r in v.respostas" :key="r.perguntaId">
            <div>
              <span class="t">{{ r.titulo }}</span>
              <span class="r">{{ r.resumo }}</span>
            </div>
            <button v-if="r.revisavel" class="botao" :disabled="estado.ocupado" :data-teste="'revisar-' + r.perguntaId" @click="alterar(r.perguntaId)">
              Alterar
            </button>
          </li>
        </ul>
      </section>
    </Janela>

    <Janela v-if="v.autos && !v.terminal" :aberta="verAutos" titulo="Autos lidos" largura="g" @fechar="verAutos = false">
      <PainelAutos :autos="v.autos" @abrir="abrirFonteDe" />
    </Janela>

    <VisualizadorFontes v-if="fonte" :pagina="fonte.pagina" :inicio="fonte.inicio" :fim="fonte.fim" @fechar="fonte = null" />
  </main>
</template>

<style scoped>
.proc {
  max-width: 1300px;
  margin: 0 auto;
  padding: 1.5rem 2rem 2.4rem;
  display: grid;
  gap: 1.3rem;
}
.painel { display: grid; gap: 1.2rem; min-width: 0; }
.painel > h1 { font-size: var(--t-xl); }
.espaco { flex: 1; }

/* ---- andamento ---- */
.andamento {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--r-md);
  box-shadow: var(--sombra-1);
  padding: 0.85rem 1.2rem 1rem;
  display: grid;
  gap: 0.9rem;
}
.topo-and { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem 0.6rem; }
.topo-and > strong { font-size: var(--t-sm); }
.topo-and .sub { font-size: var(--t-sm); color: var(--tinta-suave); }
.topo-and .sub.parou { color: var(--erro); }
.ferramenta { min-height: 2.15rem; padding: 0.3rem 0.75rem; font-size: var(--t-sm); border-color: var(--linha); }
.ferramenta.perigo { border-color: var(--erro-linha); }
.conta {
  font-size: var(--t-xs); font-weight: 600; line-height: 1.5;
  background: var(--acento); color: #fff; border-radius: 999px; padding: 0 0.45rem;
}
.confirma { display: flex; flex-wrap: wrap; align-items: center; gap: 0.6rem; background: var(--erro-claro); border: 1px solid var(--erro-linha); border-radius: var(--r-sm); padding: 0.6rem 0.8rem; }
.confirma p { color: var(--erro); flex: 1 1 20rem; font-size: var(--t-sm); }

.trilha {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
}
.trilha li {
  flex: 1 1 0;
  min-width: 0;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  text-align: center;
  font-size: var(--t-xs);
  color: var(--tinta-suave);
  padding: 0 0.2rem;
}
.trilha li::before, .trilha li::after {
  content: "";
  position: absolute;
  top: 0.8rem;
  height: 2px;
  width: 50%;
  background: var(--linha);
  transition: background var(--dur-3) var(--ease);
}
.trilha li::before { left: 0; }
.trilha li::after { right: 0; }
.trilha li:first-child::before, .trilha li:last-child::after { display: none; }
.trilha li.concluida::before, .trilha li.concluida::after, .trilha li.ativa::before, .trilha li.falhou::before { background: var(--acento); }
.trilha li.falhou::before { background: var(--erro); }
.marca {
  position: relative;
  z-index: 1;
  width: 1.7rem;
  height: 1.7rem;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-weight: 600;
  border: 2px solid var(--linha);
  background: var(--superficie);
}
.check { width: 0.95rem; height: 0.95rem; fill: none; stroke: #fff; stroke-width: 3; stroke-linecap: round; stroke-linejoin: round;
  stroke-dasharray: 20; animation: desenha 420ms var(--ease) both; }
@keyframes desenha { from { stroke-dashoffset: 20; } to { stroke-dashoffset: 0; } }
.trilha li.concluida .marca { background: var(--acento); border-color: var(--acento); }
.trilha li.ativa { color: var(--acento); font-weight: 600; }
.trilha li.ativa .marca { border-color: var(--acento); color: var(--acento); animation: chama 1.8s ease-in-out infinite; }
.trilha.trabalhando li.ativa .marca { animation: none; }
.trilha.trabalhando li.ativa .marca::after {
  content: "";
  position: absolute;
  inset: -5px;
  border-radius: 50%;
  border: 2px solid transparent;
  border-top-color: var(--acento);
  border-right-color: var(--acento);
  animation: gira-arco 1s linear infinite;
}
.trilha li.falhou, .trilha.parada li.ativa { color: var(--erro); font-weight: 600; }
.trilha.parada li.ativa .marca { border-color: var(--erro); color: var(--erro); animation: none; }
.trilha.parada li.ativa::before { background: var(--erro); }
.trilha li.falhou .marca { border-color: var(--erro); background: var(--erro); color: #fff; }
.selo-etapa { font-size: 0.7rem; background: var(--acento-claro); color: var(--acento); border-radius: 999px; padding: 0 0.5rem; }
.selo-etapa.parou { background: var(--erro-claro); color: var(--erro); }
@keyframes gira-arco { to { transform: rotate(360deg); } }
@keyframes chama {
  0%, 100% { box-shadow: 0 0 0 0 var(--acento-claro); }
  50% { box-shadow: 0 0 0 6px var(--acento-claro); }
}

/* ---- espera ---- */
.agora {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--r-md);
  box-shadow: var(--sombra-1);
  padding: 1.4rem 1.8rem;
  display: flex;
  align-items: center;
  gap: 1.8rem;
}
.agora .texto { display: grid; gap: 0.55rem; flex: 1; min-width: 0; }
.quem {
  justify-self: start; font-size: var(--t-xs); font-weight: 600; letter-spacing: 0.02em;
  border-radius: 999px; padding: 0.1rem 0.6rem; background: var(--superficie-2); color: var(--tinta-suave); border: 1px solid var(--linha);
}
.quem.robo { background: var(--acento-claro); color: var(--acento); border-color: transparent; }
.descricao { font-size: var(--t-lg); font-family: var(--serif); font-weight: 600; }
.sub { color: var(--tinta-suave); }
.barra { height: 10px; background: var(--superficie-2); border: 1px solid var(--linha); border-radius: 999px; overflow: hidden; }
.barra span {
  display: block; height: 100%; border-radius: 999px; transition: width 400ms var(--ease);
  background: var(--acento) linear-gradient(110deg, transparent 30%, rgba(255, 255, 255, 0.35) 50%, transparent 70%) 0 0 / 200% 100%;
  animation: brilho 1.8s linear infinite;
}
@keyframes brilho { from { background-position: 100% 0; } to { background-position: -100% 0; } }
.conta-progresso { margin-top: 0.35rem; font-size: var(--t-sm); color: var(--tinta-suave); }
.atividade { display: flex; gap: 0.4rem; height: 8px; align-items: center; }
.ponto { width: 8px; height: 8px; border-radius: 50%; background: var(--acento); animation: pulsa 1.2s infinite ease-in-out; }
.ponto:nth-child(2) { animation-delay: 0.2s; }
.ponto:nth-child(3) { animation-delay: 0.4s; }
@keyframes pulsa { 0%, 80%, 100% { opacity: 0.2; transform: scale(0.8); } 40% { opacity: 1; transform: scale(1); } }

/* ---- parada ---- */
.cartao-parada {
  background: var(--superficie);
  border: 1px solid var(--erro-linha);
  border-top: 4px solid var(--erro);
  border-radius: var(--r-md);
  box-shadow: var(--sombra-2);
  padding: 1.8rem;
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
}
.selo-parada { width: 4rem; height: 4rem; border-radius: 50%; background: var(--erro-claro); color: var(--erro); display: grid; place-items: center; flex: none; }
.texto-parada { display: grid; gap: 0.7rem; flex: 1; min-width: 0; }
.chapeu { font-size: var(--t-xs); font-weight: 600; color: var(--erro); text-transform: uppercase; letter-spacing: 0.03em; }
.texto-parada h2 { font-size: var(--t-xl); }
.o-que-fazer { background: var(--superficie-2); border: 1px solid var(--linha); border-radius: var(--r-sm); padding: 0.7rem 0.9rem; }
.detalhe { font-size: var(--t-xs); color: var(--tinta-suave); word-break: break-word; }
.codigo { font-size: var(--t-xs); color: var(--tinta-suave); }
.retomar { display: grid; gap: 0.8rem; }
.nota { color: var(--tinta-suave); }
.acoes { display: flex; flex-wrap: wrap; gap: 0.8rem; align-items: center; }

/* ---- fim ---- */
.proc.fim { max-width: 760px; padding-top: 3.5rem; }
.cartao-fim {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: 16px;
  box-shadow: var(--sombra-2);
  padding: 2.5rem;
  display: grid;
  justify-items: center;
  gap: 1.1rem;
  text-align: center;
}
.cartao-fim h1 { font-size: 1.9rem; }
.selo-fim { width: 5.2rem; height: 5.2rem; border-radius: 50%; display: grid; place-items: center; background: var(--superficie-2); color: var(--tinta-suave); }
.finalizada .selo-fim { background: var(--ok-claro); color: var(--ok); }
.falha .selo-fim { background: var(--erro-claro); color: var(--erro); }
.check-fim { width: 2.9rem; height: 2.9rem; fill: none; stroke: currentColor; stroke-width: 2.4; stroke-linecap: round; stroke-linejoin: round;
  stroke-dasharray: 40; animation: desenha-fim 600ms var(--ease) both; }
@keyframes desenha-fim { from { stroke-dashoffset: 40; } to { stroke-dashoffset: 0; } }
.cartao-fim .nota { max-width: 34rem; }
.cartao-fim :deep(.aviso) { text-align: left; width: 100%; }
.blocos-fim { list-style: none; margin: 0; padding: 0; width: 100%; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; text-align: left; }
.blocos-fim li { border: 1px solid var(--linha); border-radius: var(--r-md); padding: 0.85rem; display: grid; gap: 0.1rem; }
.blocos-fim span { font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); text-transform: uppercase; }
.botao.grande { min-height: 3rem; padding-inline: 1.6rem; }
.nota.pequena { font-size: var(--t-xs); }

/* ---- respostas (gaveta) ---- */
.respostas { display: grid; gap: 0.8rem; }
.respostas .nota { font-size: var(--t-sm); }
.respostas ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 0.5rem; }
.respostas li {
  display: flex; gap: 0.8rem; align-items: center; justify-content: space-between;
  padding: 0.7rem 0.85rem; border: 1px solid var(--linha); border-radius: var(--r-sm); background: var(--superficie-2);
}
.respostas li > div { display: grid; gap: 0.15rem; min-width: 0; }
.respostas .t { font-weight: 600; font-size: var(--t-sm); }
.respostas .r { color: var(--tinta-suave); font-size: var(--t-sm); overflow-wrap: anywhere; }
.respostas .botao { min-height: 2rem; padding: 0.3rem 0.8rem; font-size: var(--t-sm); flex: none; }

@media (max-width: 900px) {
  .proc { padding: 1rem 1rem 2rem; }
  .trilha li:not(.ativa):not(.falhou) .rotulo { display: none; }
  .agora, .cartao-parada { flex-direction: column; text-align: center; align-items: center; }
  .agora .texto { justify-items: center; width: 100%; }
  .quem { justify-self: center; }
  .blocos-fim { grid-template-columns: 1fr; }
}
</style>
