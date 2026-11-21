package main

// Окно установщика: своя строка заголовка, отрисовка через холст, мышь и клавиатура.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type App struct {
	hwnd     uintptr
	st       *State
	scale    float64
	canvas   *Canvas
	hits     []Hit
	tracking bool
	busy     bool
	pressed  string
	arrow    uintptr
	hand     uintptr
}

var app *App

func (a *App) invalidate() { pInvalidateRect.Call(a.hwnd, 0, 0) }

func (a *App) post() { pPostMessageW.Call(a.hwnd, WM_APP, 0, 0) }

func (a *App) setScaleFromWindow() {
	a.scale = 1
	if pGetDpiForWindow.Find() == nil {
		if dpi, _, _ := pGetDpiForWindow.Call(a.hwnd); dpi > 0 {
			a.scale = float64(dpi) / 96
		}
	}
}

func (a *App) hitAt(x, y float64) string {
	for i := len(a.hits) - 1; i >= 0; i-- {
		if a.hits[i].has(x, y) {
			return a.hits[i].ID
		}
	}
	return ""
}

func (a *App) paint(hdc uintptr) {
	var rc RECT
	pGetClientRect.Call(a.hwnd, ptr(&rc))
	w, h := int(rc.Right), int(rc.Bottom)
	if w <= 0 || h <= 0 {
		return
	}
	if a.canvas == nil || a.canvas.W != w || a.canvas.H != h {
		if a.canvas != nil {
			a.canvas.free()
		}
		a.canvas = newCanvas(w, h)
	}
	a.st.mu.Lock()
	a.hits = paint(a.canvas, a.st, a.scale)
	a.st.mu.Unlock()
	pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), a.canvas.plat.dc, 0, 0, SRCCOPY)
}

func (a *App) progress(step, detail string, frac float64) {
	a.st.mu.Lock()
	a.st.Step, a.st.Detail, a.st.Frac = step, detail, frac
	a.st.mu.Unlock()
	a.post()
}

func (a *App) startInstall() {
	st := a.st
	st.mu.Lock()
	st.Screen, st.Frac, st.Step, st.Detail, st.Err = scrProgress, 0, "Начинаю", "", ""
	dir, desktop, launch := st.Dir, st.Desktop, st.Launch
	st.mu.Unlock()
	a.busy = true
	go func() {
		defer comThread()()
		version, err := doInstall(dir, desktop, a.progress)
		launched := false
		if err == nil && launch {
			launched = launchApp(dir) == nil
		}
		st.mu.Lock()
		if err != nil {
			st.Screen, st.Err = scrError, err.Error()
		} else {
			st.Screen, st.Launched = scrDone, launched
			if version != "" {
				st.Version = version
			}
		}
		st.mu.Unlock()
		a.busy = false
		a.post()
	}()
	a.invalidate()
}

func (a *App) startUninstall() {
	st := a.st
	st.mu.Lock()
	st.Screen, st.Frac, st.Step, st.Detail, st.Err = scrUnProgress, 0, "Начинаю", "", ""
	dir, keep := st.Dir, st.KeepData
	st.mu.Unlock()
	a.busy = true
	go func() {
		defer comThread()()
		err := doUninstall(dir, keep, a.progress)
		st.mu.Lock()
		if err != nil {
			st.Screen, st.Err = scrError, err.Error()
		} else {
			st.Screen = scrUnDone
		}
		st.mu.Unlock()
		a.busy = false
		a.post()
	}()
	a.invalidate()
}

func (a *App) browse() {
	name := make([]uint16, 260)
	bi := BROWSEINFOW{HwndOwner: a.hwnd, PszDisplayName: &name[0], LpszTitle: u16("Куда установить CapCut Wiggle"),
		UlFlags: 0x0001 | 0x0040} // BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE
	pidl, _, _ := pSHBrowseForFolderW.Call(ptr(&bi))
	if pidl == 0 {
		return
	}
	buf := make([]uint16, 1024)
	pSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	pCoTaskMemFree.Call(pidl)
	p := syscall.UTF16ToString(buf)
	if p == "" {
		return
	}
	if !strings.EqualFold(filepath.Base(p), appName) {
		p = filepath.Join(p, appName)
	}
	a.st.mu.Lock()
	a.st.Dir = p
	refreshUpdate(a.st)
	a.st.mu.Unlock()
}

func refreshUpdate(st *State) {
	st.Update = fileExists(filepath.Join(st.Dir, "app.py"))
	st.OldVersion = readVersion(st.Dir)
}

func (a *App) close() {
	if a.busy {
		return
	}
	pDestroyWindow.Call(a.hwnd)
}

func (a *App) action(id string) {
	st := a.st
	st.mu.Lock()
	screen, uninstall, dir := st.Screen, st.Uninstall, st.Dir
	st.mu.Unlock()
	switch id {
	case "close", "cancel", "done":
		a.close()
	case "min":
		pShowWindow.Call(a.hwnd, SW_MINIMIZE)
	case "browse":
		a.browse()
	case "desktop", "launch", "keep":
		st.mu.Lock()
		switch id {
		case "desktop":
			st.Desktop = !st.Desktop
		case "launch":
			st.Launch = !st.Launch
		case "keep":
			st.KeepData = !st.KeepData
		}
		st.mu.Unlock()
	case "install":
		a.startInstall()
	case "retry":
		if uninstall {
			a.startUninstall()
		} else {
			a.startInstall()
		}
	case "uninstall":
		a.startUninstall()
	case "run":
		launchApp(dir)
		a.close()
	}
	_ = screen
	a.invalidate()
}

// Enter — главная кнопка экрана, Esc — отмена.
func (a *App) key(vk uintptr) {
	a.st.mu.Lock()
	screen, launched := a.st.Screen, a.st.Launched
	a.st.mu.Unlock()
	if vk == VK_ESCAPE {
		a.close()
		return
	}
	if vk != VK_RETURN {
		return
	}
	switch screen {
	case scrWelcome:
		a.action("install")
	case scrDone:
		if launched {
			a.action("done")
		} else {
			a.action("run")
		}
	case scrError:
		a.action("retry")
	case scrUnConfirm:
		a.action("uninstall")
	case scrUnDone:
		a.action("done")
	}
}

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	a := app
	switch msg {
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(hwnd, ptr(&ps))
		a.paint(hdc)
		pEndPaint.Call(hwnd, ptr(&ps))
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_NCHITTEST:
		var wr RECT
		pGetWindowRect.Call(hwnd, ptr(&wr))
		x, y := float64(loword(lp)-wr.Left), float64(hiword(lp)-wr.Top)
		if y >= 0 && y < math.Round(titleH*a.scale) {
			if id := a.hitAt(x, y); id != "min" && id != "close" {
				return HTCAPTION
			}
		}
		return HTCLIENT
	case WM_MOUSEMOVE:
		if !a.tracking {
			tme := TRACKMOUSEEVENT{CbSize: uint32(unsafe.Sizeof(TRACKMOUSEEVENT{})), DwFlags: TME_LEAVE, HwndTrack: hwnd}
			pTrackMouseEvent.Call(ptr(&tme))
			a.tracking = true
		}
		id := a.hitAt(float64(loword(lp)), float64(hiword(lp)))
		a.st.mu.Lock()
		changed := a.st.Hover != id
		a.st.Hover = id
		a.st.mu.Unlock()
		if changed {
			a.invalidate()
		}
		return 0
	case WM_MOUSELEAVE:
		a.tracking = false
		a.st.mu.Lock()
		a.st.Hover = ""
		a.st.mu.Unlock()
		a.invalidate()
		return 0
	case WM_SETCURSOR:
		if loword(lp) == HTCLIENT {
			a.st.mu.Lock()
			hover := a.st.Hover
			a.st.mu.Unlock()
			if hover != "" {
				pSetCursor.Call(a.hand)
			} else {
				pSetCursor.Call(a.arrow)
			}
			return 1
		}
	case WM_LBUTTONDOWN:
		a.pressed = a.hitAt(float64(loword(lp)), float64(hiword(lp)))
		pSetCapture.Call(hwnd)
		return 0
	case WM_LBUTTONUP:
		pReleaseCapture.Call()
		id := a.hitAt(float64(loword(lp)), float64(hiword(lp)))
		if id != "" && id == a.pressed {
			a.action(id)
		}
		a.pressed = ""
		return 0
	case WM_KEYDOWN:
		a.key(wp)
		return 0
	case WM_DPICHANGED:
		r := (*RECT)(unsafe.Pointer(lp))
		a.scale = float64(loword(wp)) / 96
		pSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), SWP_NOZORDER|SWP_NOACTIVATE)
		a.invalidate()
		return 0
	case WM_APP:
		a.invalidate()
		return 0
	case WM_CLOSE:
		a.close()
		return 0
	case WM_DESTROY:
		a.st.mu.Lock()
		unDone, dir := a.st.Screen == scrUnDone, a.st.Dir
		a.st.mu.Unlock()
		if unDone {
			removeSelfLater(dir)
		}
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func runUI(st *State) {
	hInst, _, _ := pGetModuleHandleW.Call(0)
	arrow, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	hand, _, _ := pLoadCursorW.Call(0, IDC_HAND)
	app = &App{st: st, scale: 1, arrow: arrow, hand: hand}
	// значок из ресурсов .exe (id 1): большой и маленький
	iconBig, _, _ := pLoadImageW.Call(hInst, 1, IMAGE_ICON, 0, 0, 0x0040|0x8000) // LR_DEFAULTSIZE | LR_SHARED
	smx, _, _ := pGetSystemMetrics.Call(49)                                      // SM_CXSMICON
	iconSm, _, _ := pLoadImageW.Call(hInst, 1, IMAGE_ICON, smx, smx, 0x8000)
	cls := u16("CapCutWiggleSetup")
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW | CS_DROPSHADOW,
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		HIcon:         iconBig,
		HCursor:       arrow,
		LpszClassName: cls,
		HIconSm:       iconSm,
	}
	pRegisterClassExW.Call(ptr(&wc))
	title := "Установка CapCut Wiggle"
	if st.Uninstall {
		title = "Удаление CapCut Wiggle"
	}
	hwnd, _, _ := pCreateWindowExW.Call(WS_EX_APPWINDOW, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16(title))),
		WS_POPUP|WS_SYSMENU|WS_MINIMIZEBOX, 0, 0, uintptr(logicalW), uintptr(logicalH), 0, 0, hInst, 0)
	if hwnd == 0 {
		os.Exit(1)
	}
	app.hwnd = hwnd
	app.setScaleFromWindow()
	// размер под масштаб экрана и по центру рабочей области
	w, h := int32(math.Round(logicalW*app.scale)), int32(math.Round(logicalH*app.scale))
	var work RECT
	pSystemParametersInfo.Call(SPI_GETWORKAREA, 0, ptr(&work), 0)
	x := work.Left + (work.Right-work.Left-w)/2
	y := work.Top + (work.Bottom-work.Top-h)/2
	pSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), SWP_NOZORDER)
	// скруглённые углы окна в Windows 11 (в Windows 10 вызов просто ничего не сделает)
	if pDwmSetWindowAttribute.Find() == nil {
		pref := int32(2) // DWMWCP_ROUND
		pDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&pref)), 4)
	}
	pShowWindow.Call(hwnd, SW_SHOW)
	pSetForegroundWindow.Call(hwnd)

	var m MSG
	for {
		r, _, _ := pGetMessageW.Call(ptr(&m), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(ptr(&m))
		pDispatchMessageW.Call(ptr(&m))
	}
}
