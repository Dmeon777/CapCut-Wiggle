package main

// Вызовы Win32 напрямую через syscall — без сторонних библиотек.

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pShowWindow           = user32.NewProc("ShowWindow")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pBeginPaint           = user32.NewProc("BeginPaint")
	pEndPaint             = user32.NewProc("EndPaint")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pSetCursor            = user32.NewProc("SetCursor")
	pLoadImageW           = user32.NewProc("LoadImageW")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pGetDpiForWindow      = user32.NewProc("GetDpiForWindow")
	pSetWindowPos         = user32.NewProc("SetWindowPos")
	pGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	pDestroyWindow        = user32.NewProc("DestroyWindow")
	pDrawTextW            = user32.NewProc("DrawTextW")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pGetWindowRect        = user32.NewProc("GetWindowRect")
	pTrackMouseEvent      = user32.NewProc("TrackMouseEvent")
	pSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	pSetCapture           = user32.NewProc("SetCapture")
	pReleaseCapture       = user32.NewProc("ReleaseCapture")
	pSystemParametersInfo = user32.NewProc("SystemParametersInfoW")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pBitBlt             = gdi32.NewProc("BitBlt")
	pCreateFontW        = gdi32.NewProc("CreateFontW")
	pSetTextColor       = gdi32.NewProc("SetTextColor")
	pSetBkMode          = gdi32.NewProc("SetBkMode")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW     = kernel32.NewProc("CreateMutexW")

	pSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")

	pCoInitializeEx = ole32.NewProc("CoInitializeEx")
	pCoTaskMemFree  = ole32.NewProc("CoTaskMemFree")

	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	WM_DESTROY     = 0x0002
	WM_SIZE        = 0x0005
	WM_PAINT       = 0x000F
	WM_CLOSE       = 0x0010
	WM_ERASEBKGND  = 0x0014
	WM_SETCURSOR   = 0x0020
	WM_SETICON     = 0x0080
	WM_NCHITTEST   = 0x0084
	WM_KEYDOWN     = 0x0100
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_MOUSELEAVE  = 0x02A3
	WM_DPICHANGED  = 0x02E0
	WM_APP         = 0x8000

	HTCLIENT  = 1
	HTCAPTION = 2

	WS_POPUP        = 0x80000000
	WS_SYSMENU      = 0x00080000
	WS_MINIMIZEBOX  = 0x00020000
	WS_EX_APPWINDOW = 0x00040000

	CS_HREDRAW    = 0x0002
	CS_VREDRAW    = 0x0001
	CS_DROPSHADOW = 0x00020000

	SW_SHOW     = 5
	SW_MINIMIZE = 6

	IDC_ARROW = 32512
	IDC_HAND  = 32649

	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	SRCCOPY = 0x00CC0020

	DT_CENTER        = 0x0001
	DT_RIGHT         = 0x0002
	DT_VCENTER       = 0x0004
	DT_WORDBREAK     = 0x0010
	DT_SINGLELINE    = 0x0020
	DT_NOPREFIX      = 0x0800
	DT_PATH_ELLIPSIS = 0x4000
	DT_END_ELLIPSIS  = 0x8000

	VK_RETURN = 0x0D
	VK_ESCAPE = 0x1B

	TME_LEAVE = 0x0002

	IMAGE_ICON = 1

	SPI_GETWORKAREA = 0x0030
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

type BROWSEINFOW struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

func u16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func loword(v uintptr) int32 { return int32(int16(v & 0xffff)) }
func hiword(v uintptr) int32 { return int32(int16((v >> 16) & 0xffff)) }

func ptr[T any](v *T) uintptr { return uintptr(unsafe.Pointer(v)) }
