//go:build !windows

package documento

import "os/exec"

const nomeTypst = "typst"

func ocultarJanela(*exec.Cmd) {}
