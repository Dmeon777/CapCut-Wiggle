//go:build !windows

package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"
)

const testReleaseURL = "https://github.com/Dmeon777/CapCut-Wiggle/releases/latest/download/capcut-wiggle.zip"

func TestReleaseDownloadAndExtract(t *testing.T) {
	var last int64
	data, err := download(testReleaseURL, 90*time.Second, func(got, total int64) { last = got })
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	t.Logf("downloaded %d bytes (progress %d), hasApp=%v version=%s", len(data), last, zipHasApp(data), zipVersion(data))
	if !zipHasApp(data) {
		t.Fatal("no app.py in release zip")
	}
	dir := t.TempDir()
	if err := extractApp(data, dir); err != nil {
		t.Fatal(err)
	}
	var files []string
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			r, _ := filepath.Rel(dir, p)
			files = append(files, r)
		}
		return nil
	})
	sort.Strings(files)
	t.Logf("%d files: %v", len(files), files)
	for _, must := range []string{"app.py", "corner.py", "ui/index.html", "ui/thumb.js", "icon.ico", "VERSION"} {
		if _, err := os.Stat(filepath.Join(dir, must)); err != nil {
			t.Errorf("missing %s", must)
		}
	}
	for _, f := range files {
		if strings.Contains(f, ".git") || strings.HasPrefix(f, "docs") || strings.HasPrefix(f, "installer") {
			t.Errorf("unexpected %s", f)
		}
	}
	// встроенная копия тоже годится
	if !zipHasApp(embeddedTestApp(t)) {
		t.Error("embedded app.zip broken")
	}
}

func embeddedTestApp(t *testing.T) []byte {
	b, err := os.ReadFile("app.zip")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPythonPth(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{"python312._pth": "python312.zip\r\n.\r\n\r\n# Uncomment to run site.main() automatically\r\n#import site\r\n", "pythonw.exe": "MZ", "Lib/x.txt": "x"} {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	dir := filepath.Join(t.TempDir(), "python")
	if err := extractPython(buf.Bytes(), dir); err != nil {
		t.Fatal(err)
	}
	patchPth(dir)
	patchPth(dir)
	b, _ := os.ReadFile(filepath.Join(dir, "python312._pth"))
	if n := strings.Count(string(b), "\n..\r\n") + strings.Count(string(b), "\n..\n"); n != 1 {
		t.Errorf("'..' lines = %d:\n%q", n, b)
	}
	t.Logf("_pth: %q", b)
}

// размеры структур Win32 (amd64): раскладка Go совпадает на любой ОС
type tPOINT struct{ X, Y int32 }
type tRECT struct{ Left, Top, Right, Bottom int32 }
type tMSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      tPOINT
	Private uint32
}
type tWNDCLASSEXW struct {
	CbSize                           uint32
	Style                            uint32
	LpfnWndProc                      uintptr
	CbClsExtra, CbWndExtra           int32
	HInstance, HIcon, HCursor, HbrBg uintptr
	LpszMenuName, LpszClassName      *uint16
	HIconSm                          uintptr
}
type tPAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     tRECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}
type tTME struct {
	CbSize, DwFlags uint32
	HwndTrack       uintptr
	DwHoverTime     uint32
}
type tBI struct {
	HwndOwner, PidlRoot       uintptr
	PszDisplayName, LpszTitle *uint16
	UlFlags                   uint32
	Lpfn, LParam              uintptr
	IImage                    int32
}

func TestStructSizes(t *testing.T) {
	for name, got := range map[string][2]uintptr{
		"MSG": {unsafe.Sizeof(tMSG{}), 48}, "WNDCLASSEXW": {unsafe.Sizeof(tWNDCLASSEXW{}), 80},
		"PAINTSTRUCT": {unsafe.Sizeof(tPAINTSTRUCT{}), 72}, "TRACKMOUSEEVENT": {unsafe.Sizeof(tTME{}), 24},
		"BROWSEINFOW": {unsafe.Sizeof(tBI{}), 64},
	} {
		if got[0] != got[1] {
			t.Errorf("%s: %d, want %d", name, got[0], got[1])
		}
	}
}
