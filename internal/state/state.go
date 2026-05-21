package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateDir = "/var/lib/krate/state"

type Container struct {
	ID        string    `json:"id"`
	Image     string    `json:"image"`
	Cmd       []string  `json:"cmd"`
	Hostname  string    `json:"hostname"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	LogFile   string    `json:"log_file"`
	RootFS    string    `json:"rootfs"`
}

func Save(c Container) error {
	os.MkdirAll(stateDir, 0755)
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stateDir, c.ID+".json"), data, 0644)
}

func Delete(id string) {
	os.Remove(filepath.Join(stateDir, id+".json"))
}

func Get(id string) (*Container, error) {
	entries, _ := os.ReadDir(stateDir)
	for _, e := range entries {
		name := e.Name()[:len(e.Name())-5]
		if len(name) >= len(id) && name[:len(id)] == id {
			id = name
			break
		}
	}
	data, err := os.ReadFile(filepath.Join(stateDir, id+".json"))
	if err != nil {
		return nil, err
	}
	var c Container
	return &c, json.Unmarshal(data, &c)
}

func List() ([]Container, error) {
	entries, err := os.ReadDir(stateDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var containers []Container
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(stateDir, e.Name()))
		if err != nil {
			continue
		}
		var c Container
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		if !isAlive(c.PID) {
			os.Remove(filepath.Join(stateDir, e.Name()))
			continue
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func isAlive(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}