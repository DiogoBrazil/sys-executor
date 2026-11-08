//go:build !windows

package inference

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

// Linux (desenvolvimento e CI): /proc/meminfo e /proc/cpuinfo.
func memoriaSistema() (total, livre uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Fields(sc.Text())
		if len(c) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(c[1], 10, 64)
		switch c[0] {
		case "MemTotal:":
			total = v << 10
		case "MemAvailable:":
			livre = v << 10
		}
	}
	if total == 0 {
		return 0, 0, errors.New("MemTotal ausente")
	}
	return total, livre, nil
}

func nucleosFisicos() int {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	nucleos := map[string]bool{}
	var fisico string
	for _, l := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "physical id":
			fisico = strings.TrimSpace(v)
		case "core id":
			nucleos[fisico+"/"+strings.TrimSpace(v)] = true
		}
	}
	return len(nucleos)
}
