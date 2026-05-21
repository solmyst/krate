package daemon

import (
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"krate/internal/image"
	"krate/internal/state"
)

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	stdjson.NewEncoder(w).Encode(v)
}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}

func Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.FileServer(http.Dir("/var/lib/krate/web")).ServeHTTP)

	// list containers
	mux.HandleFunc("/containers", cors(func(w http.ResponseWriter, r *http.Request) {
		containers, _ := state.List()
		if containers == nil {
			containers = []state.Container{}
		}
		writeJSON(w, containers, 200)
	}))

	// run container
	mux.HandleFunc("/containers/run", cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
			return
		}
		var body struct {
			Image    string   `json:"image"`
			Cmd      []string `json:"cmd"`
			MemoryMB int64    `json:"memory_mb"`
			CPUPct   int      `json:"cpu_pct"`
			Hostname string   `json:"hostname"`
		}
		if err := stdjson.NewDecoder(r.Body).Decode(&body); err != nil || body.Image == "" {
			writeJSON(w, map[string]string{"error": "invalid body"}, 400)
			return
		}
		if len(body.Cmd) == 0 {
			body.Cmd = []string{"/bin/sh"}
		}
		if body.MemoryMB == 0 {
			body.MemoryMB = 100
		}
		if body.CPUPct == 0 {
			body.CPUPct = 50
		}
		if body.Hostname == "" {
			body.Hostname = "krate-box"
		}
		go func() {
			args := []string{"run",
				"--memory", fmt.Sprintf("%d", body.MemoryMB),
				"--cpu", fmt.Sprintf("%d", body.CPUPct),
				"--name", body.Hostname,
				body.Image,
			}
			args = append(args, body.Cmd...)
			cmd := exec.Command("/proc/self/exe", args...)
			cmd.Start()
		}()
		writeJSON(w, map[string]string{"status": "started"}, 200)
	}))

	// stop container
	mux.HandleFunc("/containers/stop/", cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
			return
		}
		id := r.URL.Path[len("/containers/stop/"):]
		c, err := state.Get(id)
		if err != nil {
			writeJSON(w, map[string]string{"error": "container not found"}, 404)
			return
		}
		proc, _ := os.FindProcess(c.PID)
		proc.Signal(syscall.SIGTERM)
		state.Delete(c.ID)
		writeJSON(w, map[string]string{"status": "stopped"}, 200)
	}))

	// logs
	mux.HandleFunc("/containers/logs/", cors(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/containers/logs/"):]
		c, err := state.Get(id)
		if err != nil {
			writeJSON(w, map[string]string{"error": "not found"}, 404)
			return
		}
		data, _ := os.ReadFile(c.LogFile)
		writeJSON(w, map[string]string{"logs": string(data)}, 200)
	}))

	// list images
	mux.HandleFunc("/images", cors(func(w http.ResponseWriter, r *http.Request) {
		imgs, _ := image.List()
		if imgs == nil {
			imgs = []image.ImageInfo{}
		}
		writeJSON(w, imgs, 200)
	}))

	fmt.Printf("krate daemon listening on http://localhost:%s\n", port)
	return http.ListenAndServe(":"+port, mux)
}