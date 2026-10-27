package install

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "obsidian-preference-sync/internal/github"
	"obsidian-preference-sync/internal/registry"
	"obsidian-preference-sync/internal/vault"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInstallPluginDirectReleaseAssets(t *testing.T) {
	for _, tc := range []struct {
		name      string
		missing   string
		status    int
		wantError bool
	}{
		{name: "all assets"},
		{name: "optional CSS absent", missing: "styles.css", status: http.StatusNotFound},
		{name: "required asset absent", missing: "main.js", status: http.StatusNotFound, wantError: true},
		{name: "optional CSS forbidden", missing: "styles.css", status: http.StatusForbidden, wantError: true},
		{name: "optional CSS server error", missing: "styles.css", status: http.StatusInternalServerError, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			latestRequests := 0
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "github.com" {
					t.Fatalf("unexpected host: %s", r.URL)
				}
				status := http.StatusOK
				header := make(http.Header)
				body := ""
				switch r.URL.Path {
				case "/owner/plugin/releases/latest":
					latestRequests++
					status = http.StatusFound
					header.Set("Location", "https://github.com/owner/plugin/releases/tag/1.2.3")
				case "/owner/plugin/releases/tag/1.2.3":
				default:
					const prefix = "/owner/plugin/releases/download/1.2.3/"
					if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, prefix) {
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
					}
					name := strings.TrimPrefix(r.URL.Path, prefix)
					body = "release data for " + name
					if name == tc.missing {
						status = tc.status
					}
				}
				return &http.Response{Request: r, StatusCode: status, Status: http.StatusText(status), Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			v := vault.Vault{PluginsDir: filepath.Join(t.TempDir(), "plugins")}
			_, err := InstallPlugin(context.Background(), v, gh.NewClient(), registry.Plugin{ID: "plugin", Repo: "owner/plugin"})
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if latestRequests != 1 {
				t.Fatalf("resolved latest %d times", latestRequests)
			}
			if tc.wantError {
				if _, err := os.Stat(v.PluginDir("plugin")); !os.IsNotExist(err) {
					t.Fatalf("failed download left a plugin directory: %v", err)
				}
				return
			}
			for _, name := range []string{"manifest.json", "main.js", "styles.css"} {
				data, err := os.ReadFile(filepath.Join(v.PluginDir("plugin"), name))
				if name == tc.missing {
					if !os.IsNotExist(err) {
						t.Fatalf("missing optional asset was written: %v", err)
					}
				} else if err != nil || string(data) != "release data for "+name {
					t.Fatalf("asset %s: data=%q, err=%v", name, data, err)
				}
			}
		})
	}
}
