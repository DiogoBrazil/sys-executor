// Pacote policies embute os policy packs (REGRAS_IA_ESTRATEGIA.md §2.2):
// o núcleo (sempre presente) e as instruções de cada etapa por fluxo. As
// regras da etapa vêm do catálogo (internal/rules/catalog), nunca daqui.
package policies

import "embed"

//go:embed nucleo/*.md pads/*.md sr/*.md
var FS embed.FS
