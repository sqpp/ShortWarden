package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"shortwarden/internal/config"
)

func TestIntegration_HappyPath(t *testing.T) {
	dsn := os.Getenv("SHORTWARDEN_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SHORTWARDEN_POSTGRES_DSN to run integration tests")
	}

	// Run migrations.
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.New("file://"+filepath.ToSlash(filepath.Join(root, "db", "migrations")), dsn)
	if err != nil {
		t.Fatalf("migrate init: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()

	cfg := config.Config{
		HTTPAddr:           ":0",
		PostgresDSN:        dsn,
		JWTSecret:          "test-secret",
		PublicBaseURL:      "http://localhost:8080",
		CookieDomain:       "",
		CookieSecure:       false,
		CORSAllowedOrigins: "",
	}

	srv := httptest.NewServer(NewServer(cfg, db))
	defer srv.Close()

	email := "test+" + time.Now().UTC().Format("20060102150405") + "@example.com"
	password := "password123"

	// Register.
	{
		resp := mustJSON(t, http.MethodPost, srv.URL+"/v1/auth/register", nil, map[string]any{
			"email":    email,
			"password": password,
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("register status=%d body=%s", resp.StatusCode, mustBody(t, resp))
		}
	}

	// Login -> cookie.
	var cookies []*http.Cookie
	{
		resp := mustJSON(t, http.MethodPost, srv.URL+"/v1/auth/login", nil, map[string]any{
			"email":    email,
			"password": password,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login status=%d body=%s", resp.StatusCode, mustBody(t, resp))
		}
		cookies = resp.Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected cookies")
		}
	}

	// CSRF.
	var csrf string
	{
		resp := mustJSON(t, http.MethodGet, srv.URL+"/v1/auth/csrf", cookies, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("csrf status=%d body=%s", resp.StatusCode, mustBody(t, resp))
		}
		var j struct{ Token string `json:"token"` }
		_ = json.NewDecoder(resp.Body).Decode(&j)
		_ = resp.Body.Close()
		if j.Token == "" {
			t.Fatal("missing csrf token")
		}
		csrf = j.Token
		// merge csrf cookie too
		cookies = append(cookies, resp.Cookies()...)
	}

	// Create link.
	var alias string
	{
		req := httptest.NewRequest(http.MethodPost, srv.URL+"/v1/links", bytes.NewReader(mustMarshal(t, map[string]any{
			"target_url": "https://example.com",
		})))
		req.Header.Set("content-type", "application/json")
		req.Header.Set("X-CSRF-Token", csrf)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rr := httptest.NewRecorder()
		NewServer(cfg, db).ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create link status=%d body=%s", rr.Code, rr.Body.String())
		}
		var j struct {
			Alias string `json:"alias"`
		}
		_ = json.Unmarshal(rr.Body.Bytes(), &j)
		if j.Alias == "" {
			t.Fatal("missing alias")
		}
		alias = j.Alias
	}

	// Redirect should 302.
	{
		resp, err := http.Get(srv.URL + "/r/" + alias)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("redirect status=%d", resp.StatusCode)
		}
	}

	// Export JSON should succeed.
	{
		resp, err := doReq(http.MethodGet, srv.URL+"/v1/links/export?format=json", cookies, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("export status=%d body=%s", resp.StatusCode, mustBody(t, resp))
		}
	}
}

func mustJSON(t *testing.T, method, url string, cookies []*http.Cookie, jsonBody any) *http.Response {
	t.Helper()
	var body []byte
	if jsonBody != nil {
		body = mustMarshal(t, jsonBody)
	}
	resp, err := doReq(method, url, cookies, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func doReq(method, url string, cookies []*http.Cookie, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("content-type", contentType)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return http.DefaultClient.Do(req)
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			return "", errors.New("repo root not found")
		}
		cur = next
	}
}

