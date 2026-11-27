package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	githubOwner = "SnakeTheDev"
	githubRepo  = "XrayProxy"
	userAgent   = "HomeReality-UpdateCheck/1.0"
)

type Channel string

const (
	ChannelStable  Channel = "stable"
	ChannelNightly Channel = "nightly"
)

type Info struct {
	Available bool
	Channel   Channel
	Label     string // human label e.g. "v1.2.3" or run id
	URL       string // page or asset to open
}

// Check looks for a newer build on GitHub.
//
// stable  → latest GitHub Release (prefer tags / assets from go-rewrite builds)
// nightly → latest successful workflow run of "Build" on branch go-rewrite
//
// current is the running AppVersion ("dev" always counts as outdated for display).
func Check(ch Channel, current string) (*Info, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	switch ch {
	case ChannelStable:
		return checkStable(client, current)
	case ChannelNightly:
		return checkNightly(client, current)
	default:
		return nil, fmt.Errorf("unknown channel %q", ch)
	}
}

func checkStable(client *http.Client, current string) (*Info, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return &Info{Available: false, Channel: ChannelStable, Label: "no releases yet"}, nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github releases: %s — %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return &Info{Available: false, Channel: ChannelStable, Label: "no releases yet"}, nil
	}

	tag := strings.TrimPrefix(rel.TagName, "v")
	cur := strings.TrimPrefix(current, "v")
	available := current == "dev" || current == "" || tag != cur

	info := &Info{
		Available: available,
		Channel:   ChannelStable,
		Label:     rel.TagName,
		URL:       rel.HTMLURL,
	}
	// Prefer a platform-matching asset if present.
	want := platformAssetHint()
	for _, a := range rel.Assets {
		if want != "" && strings.Contains(strings.ToLower(a.Name), want) {
			info.URL = a.BrowserDownloadURL
			break
		}
	}
	return info, nil
}

func checkNightly(client *http.Client, current string) (*Info, error) {
	// Latest successful run of the Build workflow on go-rewrite (or main).
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/build-go-rewrite.yml/runs?branch=go-rewrite&status=success&per_page=1",
		githubOwner, githubRepo,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Fallback: any successful run on the repo workflow file name
		url2 := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/actions/runs?status=success&per_page=5",
			githubOwner, githubRepo,
		)
		req2, _ := http.NewRequest(http.MethodGet, url2, nil)
		req2.Header.Set("User-Agent", userAgent)
		req2.Header.Set("Accept", "application/vnd.github+json")
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return nil, fmt.Errorf("github actions: %s", resp.Status)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github actions: %s", resp2.Status)
		}
		resp = resp2
	}

	var payload struct {
		WorkflowRuns []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			HTMLURL    string `json:"html_url"`
			HeadBranch string `json:"head_branch"`
			HeadSHA    string `json:"head_sha"`
			CreatedAt  string `json:"created_at"`
			Path       string `json:"path"`
		} `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.WorkflowRuns) == 0 {
		return &Info{Available: false, Channel: ChannelNightly, Label: "no successful runs"}, nil
	}

	run := payload.WorkflowRuns[0]
	// Prefer a run of our build workflow when listing unfiltered.
	for _, r := range payload.WorkflowRuns {
		if strings.Contains(r.Path, "build-go-rewrite") || strings.EqualFold(r.Name, "Build") {
			run = r
			break
		}
	}
	short := run.HeadSHA
	if len(short) > 7 {
		short = short[:7]
	}
	label := fmt.Sprintf("nightly %s (%s)", short, run.HeadBranch)
	// Nightly is always "available" vs a release tag; open the run page so
	// the user can download artifacts.
	return &Info{
		Available: true,
		Channel:   ChannelNightly,
		Label:     label,
		URL:       run.HTMLURL,
	}, nil
}

func platformAssetHint() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return ""
	}
}
