// Package frontend embute a interface compilada (frontend/dist) no executável.
package frontend

import "embed"

// Dist contém o build do Vite. Gere com `npm run build` antes de compilar o app.
//
//go:embed all:dist
var Dist embed.FS
