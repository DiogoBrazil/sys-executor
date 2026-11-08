//go:build windows

// PoC F0-02 (Windows) — llama-server.exe como sidecar dentro de um Job Object.
// Valida: KILL_ON_JOB_CLOSE (sidecar morre com o app, inclusive em kill -9 do app),
// pico de memória pelo Job, limite de memória do Job, json_schema, e detecção de hardware
// (AVX2 via x/sys/cpu, RAM via GlobalMemoryStatusEx).
//
// Uso: sidecar.exe -server llama-server.exe -model m.gguf [-memlimit MB] [-hold]
//
//	-hold: após o teste, fica parado (para o teste de kill do processo pai).
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/cpu"
	"golang.org/x/sys/windows"
)

var (
	server   = flag.String("server", "", "llama-server.exe")
	model    = flag.String("model", "", "GGUF")
	memlimit = flag.Int("memlimit", 0, "limite de memória do Job em MB (0 = sem limite)")
	hold     = flag.Bool("hold", false, "não sair ao final")
	ctx      = flag.String("ctx", "4096", "contexto")
)

type memStatusEx struct {
	Length, MemoryLoad                                 uint32
	TotalPhys, AvailPhys, TotalPageFile, AvailPageFile uint64
	TotalVirtual, AvailVirtual, AvailExtendedVirtual   uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func ram() (total, livre uint64) {
	var m memStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m)))
	return m.TotalPhys >> 20, m.AvailPhys >> 20
}

func main() {
	flag.Parse()
	tot, livre := ram()
	fmt.Printf("hardware: AVX2=%v AVX512F=%v RAM total=%d MB livre=%d MB\n", cpu.X86.HasAVX2, cpu.X86.HasAVX512F, tot, livre)

	job, err := windows.CreateJobObject(nil, nil)
	must(err)
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if *memlimit > 0 {
		info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
		info.JobMemoryLimit = uintptr(*memlimit) << 20
	}
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	must(err)

	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	kb := make([]byte, 16)
	rand.Read(kb)
	key := hex.EncodeToString(kb)
	cmd := exec.Command(*server, "-m", *model, "--host", "127.0.0.1", "--port", fmt.Sprint(port), "--api-key", key,
		"--no-webui", "--offline", "-c", *ctx, "-ngl", "0", "-np", "1", "--reasoning", "off", "--log-disable")
	// stdout/stderr descartados (sem logs persistidos); sem janela de console.
	cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	t0 := time.Now()
	must(cmd.Start())
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	must(err)
	must(windows.AssignProcessToJobObject(job, h))
	fmt.Printf("sidecar pid=%d no Job (atribuído %v após o start)\n", cmd.Process.Pid, time.Since(t0))

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for {
		select {
		case err := <-exited:
			fmt.Printf("sidecar terminou durante a carga: %v (%.1fs) pico Job=%d MB\n", err, time.Since(t0).Seconds(), peak(job))
			os.Exit(3)
		default:
		}
		if r, err := http.Get(base + "/health"); err == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				break
			}
		}
		if time.Since(t0) > 3*time.Minute {
			fmt.Println("timeout de carga")
			os.Exit(1)
		}
		time.Sleep(200 * time.Millisecond)
		continue
	}
	fmt.Printf("carga: %.1fs, pico Job após carga=%d MB\n", time.Since(t0).Seconds(), peak(job))

	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"artigo", "inciso", "natureza"},
		"properties": map[string]any{"artigo": map[string]any{"type": "integer"}, "inciso": map[string]any{"type": "string"},
			"natureza": map[string]any{"type": "string", "enum": []string{"LEVE", "MEDIA", "GRAVE"}}}}
	body, _ := json.Marshal(map[string]any{
		"messages":    []map[string]string{{"role": "user", "content": "Extraia o dispositivo: 'conduta prevista no art. 16, inciso XVI, do RDPM/RO (natureza média)'. Responda só JSON."}},
		"temperature": 0, "max_tokens": 100,
		"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "d", "strict": true, "schema": schema}},
	})
	rq, _ := http.NewRequest("POST", base+"/v1/chat/completions", bytes.NewReader(body))
	rq.Header.Set("Authorization", "Bearer "+key)
	rq.Header.Set("Content-Type", "application/json")
	t1 := time.Now()
	rs, err := http.DefaultClient.Do(rq)
	must(err)
	var out struct {
		Choices []struct{ Message struct{ Content string } }
		Timings struct {
			PredictedPerSecond float64 `json:"predicted_per_second"`
		}
	}
	json.NewDecoder(rs.Body).Decode(&out)
	rs.Body.Close()
	content := ""
	if len(out.Choices) > 0 {
		content = strings.TrimSpace(out.Choices[0].Message.Content)
	}
	var m map[string]any
	fmt.Printf("json_schema: %s valido=%v (%.1fs, %.1f tok/s)\n", content, json.Unmarshal([]byte(content), &m) == nil, time.Since(t1).Seconds(), out.Timings.PredictedPerSecond)
	fmt.Printf("pico de memória do Job: %d MB\n", peak(job))

	if *hold {
		fmt.Println("HOLD: aguardando ser encerrado à força")
		select {}
	}
	// Sai SEM matar o sidecar: o fechamento do handle do Job (fim do processo) deve derrubá-lo.
	fmt.Println("saindo sem matar o sidecar explicitamente")
}

func peak(job windows.Handle) uint64 {
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil)
	return uint64(info.PeakJobMemoryUsed) >> 20
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}
