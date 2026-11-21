package main

// Холст для Windows: DIB-секция (пиксели доступны напрямую) и текст через GDI — ClearType, Segoe UI.

import (
	"syscall"
	"unsafe"
)

const (
	alignLeft = iota
	alignCenter
	alignRight
)

type TextOpts struct {
	Size                                  float64
	Weight                                int
	Col                                   RGB
	Align                                 int
	VCenter, Wrap, Ellipsis, PathEllipsis bool
}

type canvasPlatform struct {
	dc, bmp, old uintptr
	fonts        map[[2]int]uintptr
}

func newCanvas(w, h int) *Canvas {
	c := &Canvas{W: w, H: h}
	dc, _, _ := pCreateCompatibleDC.Call(0)
	bi := BITMAPINFO{Header: BITMAPINFOHEADER{BiSize: uint32(unsafe.Sizeof(BITMAPINFOHEADER{})), BiWidth: int32(w), BiHeight: -int32(h), BiPlanes: 1, BiBitCount: 32}}
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(dc, ptr(&bi), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	old, _, _ := pSelectObject.Call(dc, bmp)
	c.Pix = unsafe.Slice((*byte)(bits), w*h*4)
	c.plat = canvasPlatform{dc: dc, bmp: bmp, old: old, fonts: map[[2]int]uintptr{}}
	pSetBkMode.Call(dc, 1) // TRANSPARENT
	return c
}

func (c *Canvas) free() {
	p := c.plat
	for _, f := range p.fonts {
		pDeleteObject.Call(f)
	}
	pSelectObject.Call(p.dc, p.old)
	pDeleteObject.Call(p.bmp)
	pDeleteDC.Call(p.dc)
}

func (c *Canvas) font(px float64, weight int) uintptr {
	if weight == 0 {
		weight = 400
	}
	key := [2]int{int(px*10 + 0.5), weight}
	if f, ok := c.plat.fonts[key]; ok {
		return f
	}
	h := int32(px + 0.5)
	f, _, _ := pCreateFontW.Call(uintptr(-h), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(u16("Segoe UI"))))
	c.plat.fonts[key] = f
	return f
}

func (c *Canvas) Text(s string, x, y, w, h float64, o TextOpts) {
	if s == "" {
		return
	}
	dc := c.plat.dc
	pSelectObject.Call(dc, c.font(o.Size, o.Weight))
	pSetTextColor.Call(dc, uintptr(o.Col.R)|uintptr(o.Col.G)<<8|uintptr(o.Col.B)<<16)
	flags := uintptr(DT_NOPREFIX)
	switch o.Align {
	case alignCenter:
		flags |= DT_CENTER
	case alignRight:
		flags |= DT_RIGHT
	}
	if o.Wrap {
		flags |= DT_WORDBREAK
	} else {
		flags |= DT_SINGLELINE
		if o.VCenter {
			flags |= DT_VCENTER
		}
	}
	if o.Ellipsis {
		flags |= DT_END_ELLIPSIS
	}
	if o.PathEllipsis {
		flags |= DT_PATH_ELLIPSIS
	}
	r := RECT{int32(x), int32(y), int32(x + w), int32(y + h)}
	str, _ := syscall.UTF16FromString(s)
	pDrawTextW.Call(dc, uintptr(unsafe.Pointer(&str[0])), uintptr(len(str)-1), ptr(&r), flags)
}
