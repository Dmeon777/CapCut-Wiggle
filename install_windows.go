package main

// Установка и удаление: скачать программу и Python, распаковать, ярлыки, запись в «Приложения».

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	appName     = "CapCut Wiggle"
	repoURL     = "https://github.com/Dmeon777/CapCut-Wiggle"
	releaseURL  = repoURL + "/releases/latest/download/capcut-wiggle.zip"
	pyVersion   = "3.12.10"
	pyURL       = "https://www.python.org/ftp/python/" + pyVersion + "/python-" + pyVersion + "-embed-amd64.zip"
	description = "Рысканье, углы, эффекты и обложки для CapCut"
	noWindow    = 0x08000000 // CREATE_NO_WINDOW
)

//go:embed app.zip
var embeddedApp []byte

//go:embed icon.ico
var embeddedIcon []byte

type progressFn func(step, detail string, frac float64)

func defaultDir() string {
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(la, "Programs", appName)
}

// pythonFor — чем запускать программу: свой Python из папки, иначе системный.
func pythonFor(dir string) string {
	if p := filepath.Join(dir, "python", "pythonw.exe"); fileExists(p) {
		return p
	}
	for _, name := range []string{"pyw.exe", "pythonw.exe"} {
		if p, err := exec.LookPath(name); err == nil && !strings.Contains(strings.ToLower(p), `\windowsapps\`) {
			return p
		}
	}
	return ""
}

func dirSizeKB(dir string) int64 {
	var n int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			n += info.Size()
		}
		return nil
	})
	return n / 1024
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o755)
}

/* ---------- установка ---------- */

func doInstall(dir string, desktop bool, p progressFn) (string, error) {
	p("Готовлю папку", dir, 0.01)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("Не получилось создать папку %s.\n%v", dir, err)
	}
	probe := filepath.Join(dir, ".write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return "", fmt.Errorf("Нет прав на запись в папку %s. Выбери другую папку (кнопка «Изменить…» на первом экране).", dir)
	}
	os.Remove(probe)
	stopRunning(dir)

	// программа: свежая из релиза на GitHub, без интернета — копия внутри установщика
	appData, source := embeddedApp, "копия из установщика"
	p("Скачиваю CapCut Wiggle", "github.com/Dmeon777/CapCut-Wiggle", 0.03)
	if data, err := download(releaseURL, 90*time.Second, func(got, total int64) {
		p("Скачиваю CapCut Wiggle", sizeText(got, total), 0.03+0.15*fracOf(got, total))
	}); err == nil && zipHasApp(data) {
		appData, source = data, "свежая версия с GitHub"
	}
	version := zipVersion(appData)

	// свой Python, чтобы ничего не ставить в систему
	pyDir := filepath.Join(dir, "python")
	if !fileExists(filepath.Join(pyDir, "pythonw.exe")) {
		p("Скачиваю Python "+pyVersion, "python.org", 0.2)
		data, err := download(pyURL, 10*time.Minute, func(got, total int64) {
			p("Скачиваю Python "+pyVersion, sizeText(got, total), 0.2+0.55*fracOf(got, total))
		})
		if err != nil {
			if pythonFor(dir) == "" {
				return "", fmt.Errorf("Не получилось скачать Python с python.org.\n%v\n\nПроверь интернет и нажми «Повторить».", err)
			}
		} else {
			p("Распаковываю Python", pyDir, 0.77)
			if err := extractPython(data, pyDir); err != nil {
				return "", fmt.Errorf("Не получилось распаковать Python.\n%v", err)
			}
		}
	}
	patchPth(pyDir)

	p("Распаковываю CapCut Wiggle "+version, source, 0.82)
	if err := extractApp(appData, dir); err != nil {
		return "", fmt.Errorf("Не получилось распаковать программу.\n%v\n\nЕсли CapCut Wiggle открыт — закрой его и нажми «Повторить».", err)
	}
	icon := filepath.Join(dir, "icon.ico")
	if !fileExists(icon) {
		os.WriteFile(icon, embeddedIcon, 0o644)
	}

	p("Создаю ярлыки", "", 0.9)
	self, _ := os.Executable()
	uninst := filepath.Join(dir, "uninstall.exe")
	if !strings.EqualFold(filepath.Clean(self), filepath.Clean(uninst)) {
		if err := copyFile(self, uninst); err != nil {
			return "", fmt.Errorf("Не получилось записать деинсталлятор.\n%v", err)
		}
	}
	target := pythonFor(dir)
	if target == "" {
		return "", fmt.Errorf("Python не найден. Нажми «Повторить» — установщик скачает его заново.")
	}
	args := `"` + filepath.Join(dir, "app.py") + `"`
	progs, err := knownFolder(&folderPrograms)
	if err == nil {
		err = createShortcut(filepath.Join(progs, appName+".lnk"), target, args, dir, icon, description)
	}
	if err != nil {
		return "", fmt.Errorf("Программа установлена, но ярлык в меню «Пуск» не создался.\n%v", err)
	}
	if desk, err := knownFolder(&folderDesktop); err == nil {
		lnk := filepath.Join(desk, appName+".lnk")
		if desktop {
			if err := createShortcut(lnk, target, args, dir, icon, description); err != nil {
				return "", fmt.Errorf("Программа установлена, но ярлык на рабочем столе не создался.\n%v", err)
			}
		} else {
			os.Remove(lnk)
		}
	}
	p("Записываю в «Приложения» Windows", "", 0.96)
	if err := regWriteUninstall([]regValue{
		{name: "DisplayName", str: appName},
		{name: "DisplayVersion", str: version},
		{name: "Publisher", str: "CapCut Wiggle"},
		{name: "DisplayIcon", str: icon},
		{name: "InstallLocation", str: dir},
		{name: "UninstallString", str: `"` + uninst + `" /uninstall`},
		{name: "URLInfoAbout", str: repoURL},
		{name: "NoModify", dword: 1, isDW: true},
		{name: "NoRepair", dword: 1, isDW: true},
		{name: "EstimatedSize", dword: uint32(dirSizeKB(dir)), isDW: true},
	}); err != nil {
		return "", fmt.Errorf("Программа установлена, но не записалась в «Приложения» Windows.\n%v", err)
	}
	p("Готово", "", 1)
	return version, nil
}

func launchApp(dir string) error {
	py := pythonFor(dir)
	if py == "" {
		return fmt.Errorf("Python не найден")
	}
	cmd := exec.Command(py, filepath.Join(dir, "app.py"))
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: noWindow}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

/* ---------- удаление ---------- */

func doUninstall(dir string, keep bool, p progressFn) error {
	p("Закрываю CapCut Wiggle", "", 0.1)
	stopRunning(dir)
	p("Убираю ярлыки", "", 0.35)
	for _, f := range []*GUID{&folderPrograms, &folderDesktop} {
		if d, err := knownFolder(f); err == nil {
			os.Remove(filepath.Join(d, appName+".lnk"))
		}
	}
	regDeleteUninstall()
	p("Удаляю файлы", dir, 0.6)
	self, _ := os.Executable()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Не получилось открыть папку %s.\n%v", dir, err)
	}
	var failed []string
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if strings.EqualFold(filepath.Clean(full), filepath.Clean(self)) {
			continue
		}
		if keep && (e.Name() == "user_presets.json" || e.Name() == "wiggle_config.json") {
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			failed = append(failed, e.Name())
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("Не получилось удалить: %s.\nЗакрой CapCut Wiggle и нажми «Повторить».", strings.Join(failed, ", "))
	}
	p("Готово", "", 1)
	return nil
}

// removeSelfLater — деинсталлятор не может стереть сам себя, пока работает: это сделает cmd через пару секунд.
func removeSelfLater(dir string) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = `C:\Windows\System32\cmd.exe`
	}
	cmd := exec.Command(comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: noWindow,
		CmdLine: fmt.Sprintf(`"%s" /c ping -n 3 127.0.0.1 >nul & del /f /q "%s" & rmdir "%s"`, comspec, self, dir)}
	cmd.Dir = os.TempDir()
	cmd.Start()
}
