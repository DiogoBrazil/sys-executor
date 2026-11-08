//go:build !windows

package ocr

import "os/exec"

const nomeExecutavel = "tesseract"

func ocultarJanela(*exec.Cmd) {}
