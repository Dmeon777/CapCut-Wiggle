package main

// Ярлыки (IShellLink), запись в «Приложения» (реестр) и закрытие программы — прямыми вызовами Windows,
// без PowerShell: так быстрее, и антивирусы не принимают установщик за подозрительный.

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	pRegCreateKeyExW            = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW             = advapi32.NewProc("RegSetValueExW")
	pRegCloseKey                = advapi32.NewProc("RegCloseKey")
	pRegDeleteTreeW             = advapi32.NewProc("RegDeleteTreeW")
	pCoCreateInstance           = ole32.NewProc("CoCreateInstance")
	pCoUninitialize             = ole32.NewProc("CoUninitialize")
	pSHGetKnownFolderPath       = shell32.NewProc("SHGetKnownFolderPath")
	pQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

type GUID struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}

var (
	clsidShellLink = GUID{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidShellLinkW  = GUID{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPersistFile = GUID{0x0000010B, 0x0000, 0x0000, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	folderDesktop  = GUID{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
	folderPrograms = GUID{0xA77F5D77, 0x2E2B, 0x44C3, [8]byte{0xA6, 0xA2, 0xAB, 0xA6, 0x01, 0x05, 0x4A, 0x51}}
)

const uninstSubkey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\CapCutWiggle`

// comThread — COM нужен в том же системном потоке, где вызываются его методы.
func comThread() func() {
	runtime.LockOSThread()
	pCoInitializeEx.Call(0, 2)
	return func() { pCoUninitialize.Call(); runtime.UnlockOSThread() }
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	var s []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + i))
		if c == 0 {
			break
		}
		s = append(s, c)
	}
	return string(utf16.Decode(s))
}

func knownFolder(id *GUID) (string, error) {
	var p *uint16
	hr, _, _ := pSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(id)), 0, 0, uintptr(unsafe.Pointer(&p)))
	if hr != 0 {
		return "", fmt.Errorf("SHGetKnownFolderPath: 0x%x", uint32(hr))
	}
	defer pCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
	return utf16PtrToString(p), nil
}

// comCall — вызвать метод COM-объекта по номеру в таблице методов.
func comCall(obj uintptr, index int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func createShortcut(lnk, target, args, workdir, icon, desc string) error {
	var sl uintptr
	hr, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, 1, /*CLSCTX_INPROC_SERVER*/
		uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&sl)))
	if hr != 0 || sl == 0 {
		return fmt.Errorf("IShellLink: 0x%x", uint32(hr))
	}
	defer comCall(sl, 2) // Release
	t, a, w, i, d := u16(target), u16(args), u16(workdir), u16(icon), u16(desc)
	comCall(sl, 20, uintptr(unsafe.Pointer(t)))    // SetPath
	comCall(sl, 11, uintptr(unsafe.Pointer(a)))    // SetArguments
	comCall(sl, 9, uintptr(unsafe.Pointer(w)))     // SetWorkingDirectory
	comCall(sl, 17, uintptr(unsafe.Pointer(i)), 0) // SetIconLocation
	comCall(sl, 7, uintptr(unsafe.Pointer(d)))     // SetDescription
	runtime.KeepAlive([]*uint16{t, a, w, i, d})
	var pf uintptr
	if hr := comCall(sl, 0, uintptr(unsafe.Pointer(&iidPersistFile)), uintptr(unsafe.Pointer(&pf))); hr != 0 || pf == 0 {
		return fmt.Errorf("IPersistFile: 0x%x", uint32(hr))
	}
	defer comCall(pf, 2)
	l := u16(lnk)
	hr = comCall(pf, 6, uintptr(unsafe.Pointer(l)), 1) // Save
	runtime.KeepAlive(l)
	if hr != 0 {
		return fmt.Errorf("не сохранился ярлык %s: 0x%x", lnk, uint32(hr))
	}
	return nil
}

type regValue struct {
	name  string
	str   string
	dword uint32
	isDW  bool
}

func regWriteUninstall(vals []regValue) error {
	var h syscall.Handle
	key := u16(uninstSubkey)
	r, _, _ := pRegCreateKeyExW.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(key)), 0, 0, 0,
		0x20006 /*KEY_WRITE*/, 0, uintptr(unsafe.Pointer(&h)), 0)
	if r != 0 {
		return fmt.Errorf("реестр: код %d", r)
	}
	defer pRegCloseKey.Call(uintptr(h))
	for _, v := range vals {
		name := u16(v.name)
		if v.isDW {
			d := v.dword
			pRegSetValueExW.Call(uintptr(h), uintptr(unsafe.Pointer(name)), 0, 4 /*REG_DWORD*/, uintptr(unsafe.Pointer(&d)), 4)
		} else {
			data := utf16.Encode([]rune(v.str + "\x00"))
			pRegSetValueExW.Call(uintptr(h), uintptr(unsafe.Pointer(name)), 0, 1 /*REG_SZ*/, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
		}
		runtime.KeepAlive(name)
	}
	return nil
}

func regDeleteUninstall() {
	key := u16(uninstSubkey)
	pRegDeleteTreeW.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(key)))
	runtime.KeepAlive(key)
}

// stopRunning — закрыть CapCut Wiggle, если он запущен из этой папки (иначе файлы заняты).
func stopRunning(dir string) {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer syscall.CloseHandle(snap)
	prefix := strings.ToLower(strings.TrimRight(dir, `\`) + `\`)
	killed := false
	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = syscall.Process32First(snap, &pe); err == nil; err = syscall.Process32Next(snap, &pe) {
		name := strings.ToLower(syscall.UTF16ToString(pe.ExeFile[:]))
		if name != "python.exe" && name != "pythonw.exe" {
			continue
		}
		h, err := syscall.OpenProcess(0x1000|0x0001, false, pe.ProcessID) // QUERY_LIMITED_INFORMATION | TERMINATE
		if err != nil {
			continue
		}
		buf := make([]uint16, 1024)
		n := uint32(len(buf))
		if r, _, _ := pQueryFullProcessImageNameW.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r != 0 {
			if strings.HasPrefix(strings.ToLower(syscall.UTF16ToString(buf[:n])), prefix) {
				syscall.TerminateProcess(h, 1)
				killed = true
			}
		}
		syscall.CloseHandle(h)
	}
	if killed {
		time.Sleep(600 * time.Millisecond)
	}
}
