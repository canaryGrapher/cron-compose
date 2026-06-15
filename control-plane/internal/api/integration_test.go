//go:build integration
// +build integration

// Integration test: exercises the REST API against a real Postgres. Opt in by setting
// INTEGRATION_DB_URL to a writable DSN (the test applies the schema fresh).
//
//	INTEGRATION_DB_URL=postgres://croncompose:croncompose@localhost:5432/croncompose_it?sslmode=disable \
//	  go test -tags integration ./internal/api -run TestRESTRoundTrip -v
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/agentgw"
	"github.com/croncompose/croncompose/control-plane/internal/auth"
	"github.com/croncompose/croncompose/control-plane/internal/cryptobox"
	"github.com/croncompose/croncompose/control-plane/internal/pki"
)

func TestRESTRoundTrip(t *testing.T) {
	dsn := os.Getenv("INTEGRATION_DB_URL")
	if dsn == "" {
		t.Skip("set INTEGRATION_DB_URL to a writable postgres DSN to run this")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	applyMigrations(t, ctx, pool)

	log := slog.Default()

	// Seed admin.
	hash, _ := auth.Hash("password123")
	if _, err := auth.NewStore(pool).Upsert(ctx, "admin@test.local", "Admin", "owner", hash); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	bundle, err := pki.LoadOrCreate(t.TempDir(), []string{"localhost"})
	if err != nil {
		t.Fatalf("pki: %v", err)
	}
	box, err := cryptobox.New(strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("crypto: %v", err)
	}
	gw := agentgw.New(":0", log, pool, bundle, nil)

	app := New(Deps{
		Log:              log,
		Pool:             pool,
		Gateway:          gw,
		PKI:              bundle,
		GRPCAddr:         ":0",
		SessionSecret:    []byte("integration-test-secret-key"),
		PublicHTTPURL:    "http://localhost:8080/api/v1",
		PublicGRPCAddr:   "localhost:9090",
		InstallScriptURL: "https://example.invalid/install.sh",
		Crypto:           box,
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := app.Test(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. login
	if _, err := postJSON(t, client, srv.URL+"/api/v1/auth/login",
		`{"email":"admin@test.local","password":"password123"}`, http.StatusOK); err != nil {
		t.Fatal(err)
	}

	// 2. /me
	if code := getStatus(t, client, srv.URL+"/api/v1/me"); code != 200 {
		t.Fatalf("/me status: %d", code)
	}

	// 3. create server
	created := struct {
		Server struct {
			ID string `json:"id"`
		} `json:"server"`
	}{}
	body, err := postJSON(t, client, srv.URL+"/api/v1/servers",
		`{"name":"test-pi","description":"integration"}`, http.StatusCreated)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if created.Server.ID == "" {
		t.Fatal("server id missing")
	}

	// 4. create job
	jobBody, _ := json.Marshal(map[string]any{
		"server_id":     created.Server.ID,
		"name":          "echo",
		"schedule_cron": "* * * * *",
		"script_body":   "echo hi",
	})
	if _, err := postJSON(t, client, srv.URL+"/api/v1/jobs", string(jobBody), http.StatusCreated); err != nil {
		t.Fatal(err)
	}

	// 5. audit log should have entries
	auditRes, _ := client.Get(srv.URL + "/api/v1/audit?limit=10")
	if auditRes.StatusCode != 200 {
		t.Fatalf("audit: %d", auditRes.StatusCode)
	}
	defer auditRes.Body.Close()
	var auditOut struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.NewDecoder(auditRes.Body).Decode(&auditOut)
	if len(auditOut.Items) < 2 {
		t.Errorf("expected at least 2 audit entries, got %d", len(auditOut.Items))
	}
}

// --- helpers ---

func postJSON(t *testing.T, client *http.Client, url, body string, wantStatus int) ([]byte, error) {
	t.Helper()
	res, err := client.Post(url, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		return nil, &httpError{url: url, got: res.StatusCode, want: wantStatus, body: string(out)}
	}
	return out, nil
}

func getStatus(t *testing.T, client *http.Client, url string) int {
	t.Helper()
	res, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()
	return res.StatusCode
}

type httpError struct {
	url       string
	got, want int
	body      string
}

func (e *httpError) Error() string {
	return e.url + " status=" + itoa(e.got) + " want=" + itoa(e.want) + " body=" + e.body
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	root := findMigrationsDir(t)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			t.Fatalf("apply %s: %v", e.Name(), err)
		}
	}
}

// findMigrationsDir walks up from the package dir to find ../migrations.
func findMigrationsDir(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(wd, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		wd = filepath.Dir(wd)
	}
	t.Fatal("could not find migrations/ dir")
	return ""
}
