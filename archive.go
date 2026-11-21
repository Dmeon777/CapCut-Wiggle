package main

// Работа с архивами и сетью — без системных вызовов, поэтому проверяется тестами и не на Windows.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func readVersion(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

/* ---------- архив программы ---------- */

// appRoot — папка внутри архива, где лежит app.py (у релиза это «capcut-wiggle/»).
func appRoot(zr *zip.Reader) (string, bool) {
	best, ok := "", false
	for _, f := range zr.File {
		if path.Base(f.Name) != "app.py" || strings.Contains(f.Name, ".git/") {
			continue
		}
		root := strings.TrimSuffix(f.Name, "app.py")
		if !ok || len(root) < len(best) {
			best, ok = root, true
		}
	}
	return best, ok
}

func openZip(data []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

func zipHasApp(data []byte) bool {
	zr, err := openZip(data)
	if err != nil {
		return false
	}
	_, ok := appRoot(zr)
	return ok
}

func zipVersion(data []byte) string {
	zr, err := openZip(data)
	if err != nil {
		return "1.0"
	}
	root, _ := appRoot(zr)
	for _, f := range zr.File {
		if f.Name == root+"VERSION" {
			if r, err := f.Open(); err == nil {
				b, _ := io.ReadAll(r)
				r.Close()
				if v := strings.TrimSpace(string(b)); v != "" {
					return v
				}
			}
		}
	}
	return "1.0"
}

var keepTop = map[string]bool{
	"app.py": true, "capcut_wiggle.py": true, "corner.py": true, "wiggle_project.py": true, "VERSION": true,
	"README.md": true, "CHANGELOG.md": true, "LICENSE": true, "Start.bat": true, "Start (debug).bat": true,
}

func writeZipFile(f *zip.File, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	var lastErr error
	for i := 0; i < 5; i++ { // файл может быть ещё занят закрывающейся программой
		w, err := os.Create(dst)
		if err != nil {
			lastErr = err
			time.Sleep(400 * time.Millisecond)
			continue
		}
		_, err = io.Copy(w, r)
		w.Close()
		return err
	}
	return lastErr
}

func extractApp(data []byte, dir string) error {
	zr, err := openZip(data)
	if err != nil {
		return err
	}
	root, ok := appRoot(zr)
	if !ok {
		return fmt.Errorf("в архиве нет app.py")
	}
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, root) || strings.HasSuffix(f.Name, "/") {
			continue
		}
		rel := f.Name[len(root):]
		var dst string
		switch {
		case keepTop[rel], strings.HasPrefix(rel, "ui/"):
			dst = rel
		case rel == "installer/icon.ico", rel == "icon.ico":
			dst = "icon.ico"
		default:
			continue
		}
		if strings.Contains(dst, "..") {
			continue
		}
		if err := writeZipFile(f, filepath.Join(dir, filepath.FromSlash(dst))); err != nil {
			return err
		}
	}
	return nil
}

func extractPython(data []byte, pyDir string) error {
	zr, err := openZip(data)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") || strings.Contains(f.Name, "..") {
			continue
		}
		if err := writeZipFile(f, filepath.Join(pyDir, filepath.FromSlash(f.Name))); err != nil {
			return err
		}
	}
	return nil
}

// patchPth — встроенный Python видит только пути из ._pth; программа лежит на уровень выше.
func patchPth(pyDir string) {
	m, _ := filepath.Glob(filepath.Join(pyDir, "python*._pth"))
	for _, p := range m {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n") {
			if strings.TrimSpace(line) == ".." {
				return
			}
		}
		os.WriteFile(p, append(b, []byte("\r\n..\r\n")...), 0o644)
	}
}

/* ---------- сеть ---------- */

func download(url string, timeout time.Duration, prog func(got, total int64)) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CapCutWiggle-Setup")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("сервер ответил %s", resp.Status)
	}
	var buf bytes.Buffer
	chunk := make([]byte, 64<<10)
	var got int64
	for {
		n, err := resp.Body.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			got += int64(n)
			prog(got, resp.ContentLength)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func sizeText(got, total int64) string {
	if total > 0 {
		return fmt.Sprintf("%.1f из %.1f МБ", float64(got)/1e6, float64(total)/1e6)
	}
	return fmt.Sprintf("%.1f МБ", float64(got)/1e6)
}

func fracOf(got, total int64) float64 {
	if total <= 0 {
		return 0.5
	}
	return float64(got) / float64(total)
}
