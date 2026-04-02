package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const upgradeRepo = "anthonygacis/git-master-tool"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func runUpgrade() {
	exe, err := resolveExePath()
	if err != nil {
		fatalf("could not determine binary path: %v", err)
	}

	fmt.Printf("\n%s%sUpgrade gitmt%s\n\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s→%s Checking latest release...\n", colorDim, colorReset)

	latest, err := fetchLatestTag()
	if err != nil {
		fatalf("could not fetch latest release: %v", err)
	}

	if version != "dev" && latest == version {
		fmt.Printf("%s✓%s Already on the latest version (%s)\n\n", colorGreen, colorReset, version)
		return
	}

	fmt.Printf("  current : %s\n", version)
	fmt.Printf("  latest  : %s%s%s\n\n", colorBold, latest, colorReset)

	asset := releaseAsset()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", upgradeRepo, latest, asset)

	fmt.Printf("%s→%s Downloading %s...\n", colorDim, colorReset, asset)
	tmp, err := downloadToTemp(url)
	if err != nil {
		fatalf("download failed: %v", err)
	}
	defer os.Remove(tmp)

	if err := os.Chmod(tmp, 0755); err != nil {
		fatalf("could not set permissions: %v", err)
	}

	fmt.Printf("%s→%s Installing to %s...\n", colorDim, colorReset, exe)
	if err := installBinary(tmp, exe); err != nil {
		fatalf("install failed: %v\n       try running with sudo", err)
	}

	fmt.Printf("%s✓%s Updated to %s\n\n", colorGreen, colorReset, latest)
}

func resolveExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", upgradeRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "gitmt/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no releases found at github.com/%s", upgradeRepo)
	}
	return rel.TagName, nil
}

func releaseAsset() string {
	switch runtime.GOOS {
	case "darwin":
		return "gitmt-darwin-universal"
	case "windows":
		return fmt.Sprintf("gitmt-windows-%s.exe", runtime.GOARCH)
	default:
		return fmt.Sprintf("gitmt-%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}

func downloadToTemp(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "gitmt/"+version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned %d — check that the release asset exists", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "gitmt-upgrade-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	tmp.Close()
	return name, nil
}

func installBinary(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".gitmt-upgrade-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	in, err := os.Open(src)
	if err != nil {
		tmp.Close()
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		in.Close()
		tmp.Close()
		return err
	}
	in.Close()
	tmp.Close()

	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}
