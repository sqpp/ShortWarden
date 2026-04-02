package httpapi

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type updateState struct {
	Running      bool      `json:"running"`
	LastStarted  time.Time `json:"last_started,omitempty"`
	LastFinished time.Time `json:"last_finished,omitempty"`
	ExitCode     int       `json:"exit_code,omitempty"`
	Output       string    `json:"output,omitempty"`
	Error        string    `json:"error,omitempty"`
}

var (
	updateMu sync.Mutex
	upd      updateState
)

func defaultUpdateCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/update.ps1"
	}
	return "sh ./scripts/update.sh"
}

func resolvedUpdateCommand() string {
	v := strings.TrimSpace(os.Getenv("SHORTWARDEN_UPDATE_COMMAND"))
	if v != "" {
		return v
	}
	return defaultUpdateCommand()
}

func runUpdateInBackground(cmdLine string) {
	// Long timeout because pulling containers can take a while.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", cmdLine)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-lc", cmdLine)
	}
	out, err := cmd.CombinedOutput()

	updateMu.Lock()
	defer updateMu.Unlock()
	upd.Running = false
	upd.LastFinished = time.Now().UTC()
	upd.Output = string(out)
	if len(upd.Output) > 12000 {
		upd.Output = upd.Output[len(upd.Output)-12000:]
	}
	if err != nil {
		upd.Error = err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			upd.ExitCode = ee.ExitCode()
		} else {
			upd.ExitCode = 1
		}
		return
	}
	upd.Error = ""
	upd.ExitCode = 0
}

func (h *Handler) TriggerUpdate(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	if _, ok := requireUserID(r); !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	updateMu.Lock()
	if upd.Running {
		updateMu.Unlock()
		writeError(w, http.StatusConflict, "update already running")
		return
	}
	upd.Running = true
	upd.LastStarted = time.Now().UTC()
	upd.Error = ""
	upd.ExitCode = 0
	upd.Output = ""
	cmd := resolvedUpdateCommand()
	updateMu.Unlock()

	go runUpdateInBackground(cmd)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"message": "update started",
		"command": cmd,
	})
}

func (h *Handler) GetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUserID(r); !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	updateMu.Lock()
	s := upd
	updateMu.Unlock()
	writeJSON(w, http.StatusOK, s)
}

