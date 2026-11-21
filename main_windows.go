package main

// CapCut Wiggle — установщик и деинсталлятор в одном файле.
//   CapCut-Wiggle-Setup.exe              — установка или обновление
//   uninstall.exe /uninstall             — удаление (так его зовёт «Приложения» Windows)

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func init() { runtime.LockOSThread() } // окно и его сообщения — в одном потоке

func main() {
	pCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED — нужно окну выбора папки
	uninstall := false
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "/uninstall") || a == "--uninstall" {
			uninstall = true
		}
	}
	st := &State{Desktop: true, Launch: true, KeepData: true, Version: zipVersion(embeddedApp)}
	if uninstall {
		self, _ := os.Executable()
		st.Dir = filepath.Dir(self)
		st.Uninstall = true
		st.Screen = scrUnConfirm
		st.OldVersion = readVersion(st.Dir)
		if st.OldVersion == "" {
			st.OldVersion = st.Version
		}
	} else {
		st.Dir = defaultDir()
		refreshUpdate(st)
	}
	runUI(st)
}
