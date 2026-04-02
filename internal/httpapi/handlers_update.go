package httpapi

import (
	"encoding/json"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"shortwarden/internal/buildinfo"
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

type systemVersion struct {
	AppVersion string `json:"app_version"`
	BuildTime  string `json:"build_time,omitempty"`
	GitSHA     string `json:"git_sha,omitempty"`
}

type latestVersionResponse struct {
	Image         string `json:"image"`
	LatestVersion string `json:"latest_version"`
}

func scriptName() string {
	if runtime.GOOS == "windows" {
		return "update.ps1"
	}
	return "update.sh"
}

func findUpdateScriptPath() string {
	// Optional explicit override path.
	if v := strings.TrimSpace(os.Getenv("SHORTWARDEN_UPDATE_SCRIPT")); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	name := scriptName()
	candidates := []string{
		filepath.Join("scripts", name),
		filepath.Join("..", "scripts", name),
		filepath.Join("..", "..", "scripts", name),
	}

	// Also try near the running binary.
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "scripts", name),
			filepath.Join(exeDir, "..", "scripts", name),
			filepath.Join(exeDir, "..", "..", "scripts", name),
		)
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return ""
}

func defaultUpdateCommand() string {
	script := findUpdateScriptPath()
	if script != "" {
		if runtime.GOOS == "windows" {
			return `powershell -NoProfile -ExecutionPolicy Bypass -File "` + script + `"`
		}
		return `sh "` + script + `"`
	}
	return ""
}

func resolvedUpdateCommand() string {
	v := strings.TrimSpace(os.Getenv("SHORTWARDEN_UPDATE_COMMAND"))
	if v != "" {
		return v
	}
	// Use internal default flow unless explicitly overridden.
	return ""
}

func resolveWorkspaceHostPath(dockerPath string) (string, error) {
	cid := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if cid == "" {
		if h, err := os.Hostname(); err == nil {
			cid = strings.TrimSpace(h)
		}
	}
	if cid == "" {
		return "", errors.New("cannot determine current container id")
	}
	cmd := exec.Command(
		dockerPath,
		"inspect",
		cid,
		"--format",
		"{{range .Mounts}}{{if eq .Destination \"/workspace\"}}{{.Source}}{{end}}{{end}}",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to inspect workspace mount: %w", err)
	}
	hostPath := strings.TrimSpace(string(out))
	if hostPath == "" {
		return "", errors.New("workspace host path not found from container mounts")
	}
	return hostPath, nil
}

func resolveDockerExecutable() (string, error) {
	// 1) Standard PATH lookup.
	if p, err := exec.LookPath("docker"); err == nil {
		return p, nil
	}

	// 2) Windows-specific fallback via PowerShell command discovery.
	if runtime.GOOS == "windows" {
		ps := exec.Command("powershell", "-NoProfile", "-Command", "(Get-Command docker -ErrorAction SilentlyContinue).Source")
		out, err := ps.CombinedOutput()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				if _, statErr := os.Stat(p); statErr == nil {
					return p, nil
				}
			}
		}

		// 3) where.exe fallback.
		wh := exec.Command("where.exe", "docker")
		out2, err2 := wh.CombinedOutput()
		if err2 == nil {
			lines := strings.Split(strings.TrimSpace(string(out2)), "\n")
			if len(lines) > 0 {
				p := strings.TrimSpace(lines[0])
				if p != "" {
					if _, statErr := os.Stat(p); statErr == nil {
						return p, nil
					}
				}
			}
		}

		// 4) Common Docker Desktop install locations.
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		candidates := []string{
			filepath.Join(programFiles, "Docker", "Docker", "resources", "bin", "docker.exe"),
			filepath.Join(programFilesX86, "Docker", "Docker", "resources", "bin", "docker.exe"),
			`C:\Program Files\Docker\Docker\resources\bin\docker.exe`,
		}
		for _, c := range candidates {
			if c == "" {
				continue
			}
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
	}

	return "", errors.New(`docker executable not found (process PATH / command discovery)`)
}

func runUpdateInBackground(cmdLine string) {
	// Long timeout because pulling containers can take a while.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	run := func(name string, args ...string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			updateMu.Lock()
			upd.Output += string(out)
			if len(upd.Output) > 12000 {
				upd.Output = upd.Output[len(upd.Output)-12000:]
			}
			updateMu.Unlock()
		}
		return err
	}

	var err error
	if cmdLine != "" {
		// Explicit override command.
		if runtime.GOOS == "windows" {
			err = run("powershell", "-NoProfile", "-Command", cmdLine)
		} else {
			err = run("sh", "-lc", cmdLine)
		}
	} else {
		// Default Docker flow without hardcoded docker path.
		dockerPath, lookErr := resolveDockerExecutable()
		if lookErr != nil {
			err = lookErr
		} else {
			projectName := strings.TrimSpace(os.Getenv("SHORTWARDEN_COMPOSE_PROJECT_NAME"))
			if projectName == "" {
				projectName = "shortwarden"
			}
			workspaceHost, hostErr := resolveWorkspaceHostPath(dockerPath)
			if hostErr != nil {
				err = hostErr
			} else {
				updaterImage := strings.TrimSpace(os.Getenv("SHORTWARDEN_UPDATER_IMAGE"))
				if updaterImage == "" {
					updaterImage = "docker:27-cli"
				}
				helperCmd := fmt.Sprintf(`set -eu
docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace pull api
if docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace config --services 2>/dev/null | grep -qx screenshot; then
  docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace build screenshot
  docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace up -d --no-deps --force-recreate api screenshot
else
  docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace up -d --no-deps --force-recreate api
fi
docker compose -p %s -f /workspace/docker-compose.nginx.yml --project-directory /workspace restart nginx
docker image prune -f
`,
					projectName, projectName, projectName, projectName, projectName, projectName,
				)
				if e := run(
					dockerPath,
					"run", "--rm",
					"-v", "/var/run/docker.sock:/var/run/docker.sock",
					"-v", workspaceHost+":/workspace",
					"-w", "/workspace",
					updaterImage,
					"sh", "-lc", helperCmd,
				); e != nil {
					err = e
				}
			}
		}
	}

	updateMu.Lock()
	defer updateMu.Unlock()
	upd.Running = false
	upd.LastFinished = time.Now().UTC()
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

func (h *Handler) GetSystemVersion(w http.ResponseWriter, r *http.Request) {
	v := strings.TrimSpace(buildinfo.Version)
	if v == "" || v == "0.0.0" {
		v = strings.TrimSpace(os.Getenv("SHORTWARDEN_APP_VERSION"))
	}
	if v == "" {
		v = "0.0.0"
	}
	bt := strings.TrimSpace(buildinfo.BuildTime)
	if bt == "" {
		bt = strings.TrimSpace(os.Getenv("SHORTWARDEN_BUILD_TIME"))
	}
	sha := strings.TrimSpace(buildinfo.GitSHA)
	if sha == "" {
		sha = strings.TrimSpace(os.Getenv("SHORTWARDEN_GIT_SHA"))
	}
	writeJSON(w, http.StatusOK, systemVersion{
		AppVersion: v,
		BuildTime:  bt,
		GitSHA:     sha,
	})
}

func parseDockerImage(v string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(v), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseSemver(v string) (int, int, int, bool) {
	clean := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(v), "v"))
	ps := strings.Split(clean, ".")
	if len(ps) != 3 {
		return 0, 0, 0, false
	}
	maj, err1 := strconv.Atoi(ps[0])
	min, err2 := strconv.Atoi(ps[1])
	pat, err3 := strconv.Atoi(ps[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}

func newerTag(a, b string) bool {
	amj, ami, apa, aok := parseSemver(a)
	bmj, bmi, bpa, bok := parseSemver(b)
	if !aok || !bok {
		return false
	}
	if amj != bmj {
		return amj > bmj
	}
	if ami != bmi {
		return ami > bmi
	}
	return apa > bpa
}

func (h *Handler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	image := strings.TrimSpace(os.Getenv("SHORTWARDEN_DOCKER_IMAGE"))
	if image == "" {
		image = "sqpp/shortwarden"
	}
	ns, repo, ok := parseDockerImage(image)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid SHORTWARDEN_DOCKER_IMAGE")
		return
	}

	res, err := http.Get("https://hub.docker.com/v2/repositories/" + ns + "/" + repo + "/tags?page_size=100")
	if err != nil {
		writeError(w, http.StatusBadGateway, "docker hub unreachable")
		return
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "docker hub returned "+res.Status)
		return
	}
	var payload struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadGateway, "invalid docker hub response")
		return
	}
	best := ""
	for _, t := range payload.Results {
		name := strings.TrimSpace(t.Name)
		if _, _, _, ok := parseSemver(name); !ok {
			continue
		}
		if best == "" || newerTag(name, best) {
			best = name
		}
	}
	writeJSON(w, http.StatusOK, latestVersionResponse{
		Image:         image,
		LatestVersion: best,
	})
}

