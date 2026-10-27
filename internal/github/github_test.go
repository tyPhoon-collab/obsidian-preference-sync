package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLatestReleasePinsRedirectedTag(t *testing.T) {
	client := Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodHead || r.URL.Host != "github.com" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		header := make(http.Header)
		status := http.StatusOK
		switch r.URL.EscapedPath() {
		case "/owner/plugin/releases/latest":
			status = http.StatusFound
			header.Set("Location", "https://github.com/owner/renamed/releases/tag/release%2F1.2.3")
		case "/owner/renamed/releases/tag/release%2F1.2.3":
		default:
			t.Fatalf("unexpected URL: %s", r.URL)
		}
		return &http.Response{Request: r, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}}
	release, err := client.LatestRelease(context.Background(), "owner/plugin")
	if err != nil {
		t.Fatal(err)
	}
	if release.TagName != "release/1.2.3" {
		t.Fatalf("unexpected tag: %q", release.TagName)
	}
	for _, name := range []string{"manifest.json", "main.js", "styles.css"} {
		want := "https://github.com/owner/renamed/releases/download/release%2F1.2.3/" + name
		if got := release.AssetURL(name); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestLatestReleaseRejectsMissingOrUnresolvedRelease(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{Request: r, StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
			})}}
			if _, err := client.LatestRelease(context.Background(), "owner/plugin"); err == nil {
				t.Fatal("expected failure without a resolved release tag")
			}
		})
	}
}

func TestDownloadRepoFileUsesRawDefaultBranch(t *testing.T) {
	client := Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.String() != "https://raw.githubusercontent.com/owner/theme/HEAD/theme.css" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		return &http.Response{Request: r, StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("body {}"))}, nil
	})}}
	data, err := client.DownloadRepoFile(context.Background(), "https://github.com/owner/theme", "theme.css")
	if err != nil || string(data) != "body {}" {
		t.Fatalf("data=%q, err=%v", data, err)
	}
}

func TestDownloadDistinguishesNotFound(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{Request: r, StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(""))}, nil
			})}}
			_, err := client.Download(context.Background(), "https://github.com/owner/plugin/releases/download/1.2.3/styles.css")
			if err == nil || errors.Is(err, ErrNotFound) != (status == http.StatusNotFound) {
				t.Fatalf("status=%d, err=%v", status, err)
			}
		})
	}
}
