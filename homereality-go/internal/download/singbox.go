package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const Version = "1.14.0"

// Official release SHA-256 for pinned version (windows zips). Darwin/linux verified by presence after extract.
var pinnedHashes = map[string]string{
	"windows/amd64": "3ffb56267da14e287be48bd10cf7e6505260125bad940b75101fbb4d5d58e5d6",
	"windows/arm64": "f58dff882b2feb022da8de41943804b38681ecab5e1f490f23602fc37e9d5dd4",
}

func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "sing-box.exe"
	}
	return "sing-box"
}

func platArch() (plat, arch string) {
	plat = runtime.GOOS
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	default:
		arch = runtime.GOARCH
	}
	if plat == "darwin" {
		plat = "darwin"
	}
	return plat, arch
}

// Ensure places sing-box next to the executable (or in dir). No-op if already present.
func Ensure(dir string, progress func(string)) error {
	dest := filepath.Join(dir, BinaryName())
	if st, err := os.Stat(dest); err == nil && st.Size() > 1000 {
		return nil
	}
	if progress != nil {
		progress("Downloading sing-box…")
	}
	plat, arch := platArch()
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-%s-%s", Version, Version, plat, arch)
	var archivePath string
	var isZip bool
	if plat == "windows" {
		url += ".zip"
		isZip = true
	} else {
		url += ".tar.gz"
	}

	tmpDir, err := os.MkdirTemp("", "hr-sb-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath = filepath.Join(tmpDir, "sb.archive")
	if err := httpDownload(url, archivePath); err != nil {
		// try latest
		if progress != nil {
			progress("Pinned version failed, trying latest…")
		}
		latest, err2 := latestAssetURL(plat, arch, isZip)
		if err2 != nil {
			return fmt.Errorf("download: %v (fallback: %v)", err, err2)
		}
		if err := httpDownload(latest, archivePath); err != nil {
			return err
		}
	} else if h, ok := pinnedHashes[plat+"/"+arch]; ok && isZip {
		sum, err := fileSHA256(archivePath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, h) {
			return fmt.Errorf("checksum mismatch for sing-box (got %s)", sum)
		}
	}

	if isZip {
		if err := extractZipBin(archivePath, tmpDir, BinaryName()); err != nil {
			return err
		}
	} else {
		if err := extractTarGzBin(archivePath, tmpDir, BinaryName()); err != nil {
			return err
		}
	}

	bin := findFile(tmpDir, BinaryName())
	if bin == "" {
		return fmt.Errorf("sing-box binary not found in archive")
	}
	// also copy libcronet.dll on Windows if present
	if runtime.GOOS == "windows" {
		if dll := findFile(tmpDir, "libcronet.dll"); dll != "" {
			_ = copyFile(dll, filepath.Join(dir, "libcronet.dll"))
		}
	}
	if err := copyFile(bin, dest); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dest, 0o755)
	}
	if progress != nil {
		progress("sing-box ready")
	}
	return nil
}

func httpDownload(url, path string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Home-Reality")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("HTTP %d for %s", res.StatusCode, url)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, res.Body)
	return err
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func latestAssetURL(plat, arch string, zip bool) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/SagerNet/sing-box/releases/latest", nil)
	req.Header.Set("User-Agent", "Home-Reality")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var body struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	suffix := fmt.Sprintf("%s-%s", plat, arch)
	if zip {
		suffix += ".zip"
	} else {
		suffix += ".tar.gz"
	}
	for _, a := range body.Assets {
		if strings.Contains(a.Name, suffix) {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset matching %s", suffix)
}

func extractZipBin(zipPath, dest, want string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base != want && base != "libcronet.dll" {
			continue
		}
		out := filepath.Join(dest, base)
		if err := writeZipFile(f, out); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, out string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.Create(out)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, rc)
	return err
}

func extractTarGzBin(path, dest, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		base := filepath.Base(hdr.Name)
		if base != want {
			continue
		}
		out := filepath.Join(dest, base)
		w, err := os.Create(out)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, tr); err != nil {
			w.Close()
			return err
		}
		w.Close()
	}
	return nil
}

func findFile(dir, name string) string {
	var found string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == name {
			found = p
			return io.EOF
		}
		return nil
	})
	return found
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
