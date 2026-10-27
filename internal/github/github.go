package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Release struct {
	TagName string
	repo    string
}

var ErrNotFound = errors.New("GitHub file not found")

type Client struct {
	http *http.Client
}

func NewClient() Client {
	return Client{http: &http.Client{Timeout: 60 * time.Second}}
}

func (c Client) LatestRelease(ctx context.Context, repo string) (Release, error) {
	latestURL := fmt.Sprintf("https://github.com/%s/releases/latest", strings.TrimPrefix(repo, "https://github.com/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, latestURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "obsidian-preference-sync")

	resp, err := c.http.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("fetch latest release for %s: %w", repo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Release{}, fmt.Errorf("fetch latest release for %s: HTTP %s", repo, resp.Status)
	}
	// Resolve latest once so all assets belong to the same release, even if a new one is published mid-install.
	repoPath, tag, ok := strings.Cut(resp.Request.URL.EscapedPath(), "/releases/tag/")
	if !ok || tag == "" || resp.Request.URL.Host != "github.com" || resp.Request.URL.Scheme != "https" {
		return Release{}, fmt.Errorf("resolve latest release for %s: unexpected URL %s", repo, resp.Request.URL)
	}
	tag, err = url.PathUnescape(tag)
	if err != nil {
		return Release{}, fmt.Errorf("parse release tag for %s: %w", repo, err)
	}
	return Release{TagName: tag, repo: strings.TrimPrefix(repoPath, "/")}, nil
}

func (r Release) AssetURL(name string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", r.repo, url.PathEscape(r.TagName), url.PathEscape(name))
}

func (c Client) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "obsidian-preference-sync")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("download %s: %w", url, ErrNotFound)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: HTTP %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download %s: %w", url, err)
	}
	return data, nil
}

func (c Client) DownloadRepoFile(ctx context.Context, repo string, path string) ([]byte, error) {
	fileURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/HEAD/%s", strings.TrimPrefix(repo, "https://github.com/"), path)
	return c.Download(ctx, fileURL)
}
