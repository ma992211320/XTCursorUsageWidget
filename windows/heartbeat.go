package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const reportBase = "https://cursor.kj1001.fun"

func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d%d", time.Now().UnixNano(), os.Getpid())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func deviceID() string {
	dir := appDir()
	p := filepath.Join(dir, "device_id")
	if raw, err := os.ReadFile(p); err == nil {
		s := strings.TrimSpace(string(raw))
		if len(s) >= 8 && len(s) <= 80 {
			return s
		}
	}
	id := newUUIDv4()
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(p, []byte(id), 0600)
	return id
}

func appEdition() string {
	if uninstallerPath() != "" {
		return "installed"
	}
	return "portable"
}

func sendHeartbeatOnce(status string, timeout time.Duration) {
	id := deviceID()
	if id == "" {
		return
	}
	payload, err := json.Marshal(map[string]string{
		"id":      id,
		"version": appVersion,
		"os":      "windows",
		"edition": appEdition(),
		"status":  status,
	})
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, reportBase+"/api/heartbeat", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", appTitle+"/"+appVersion)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	if status != "online" {
		return
	}
	var info updateInfo
	if json.Unmarshal(raw, &info) != nil {
		return
	}
	considerUpdate(info)
}

func sendUninstallHeartbeat() {
	sendHeartbeatOnce("uninstalled", 3*time.Second)
}

func startHeartbeat() {
	sendHeartbeatOnce("online", 8*time.Second)
	defer sendHeartbeatOnce("offline", 2*time.Second)
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for range t.C {
		sendHeartbeatOnce("online", 8*time.Second)
	}
}
