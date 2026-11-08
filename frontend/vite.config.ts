import { defineConfig, type Plugin } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import { cpSync, createReadStream, existsSync } from "node:fs";
import { join, normalize } from "node:path";

// Recursos do PDF.js servidos pelo próprio app (funciona offline):
// fontes padrão (PDFs sem fontes embutidas) e decodificadores WebAssembly.
const PDFJS = join(__dirname, "node_modules", "pdfjs-dist");
const PASTAS = ["standard_fonts", "wasm"];

function recursosPdfjs(): Plugin {
  return {
    name: "recursos-pdfjs",
    configureServer(server) {
      server.middlewares.use("/pdfjs/", (req, res, next) => {
        const rel = normalize(decodeURIComponent((req.url ?? "").split("?")[0])).replace(/^([/\\])+/, "");
        const [pasta] = rel.split(/[/\\]/);
        const arq = join(PDFJS, rel);
        if (!PASTAS.includes(pasta) || !arq.startsWith(PDFJS) || !existsSync(arq)) return next();
        createReadStream(arq).pipe(res);
      });
    },
    closeBundle() {
      for (const p of PASTAS) cpSync(join(PDFJS, p), join(__dirname, "dist", "pdfjs", p), { recursive: true });
    },
  };
}

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    // Sem sourcemaps no produto (nada de código-fonte extra no executável).
    sourcemap: false,
    // App desktop: os arquivos são locais; o PDF.js (carregado sob demanda) é grande.
    chunkSizeWarningLimit: 2000,
  },
  plugins: [vue(), wails("./bindings"), recursosPdfjs()],
});
