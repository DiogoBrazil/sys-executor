//go:build !windows

package main

import "os/exec"

func excluirDoWER() {}

func prepararFaxina(*exec.Cmd) {}
