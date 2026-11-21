//go:build !windows

package main

// Заглушка для проверки раскладки не на Windows: вместо текста — полоски его примерного размера.

type canvasPlatform struct{}

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

func (c *Canvas) Text(s string, x, y, w, h float64, o TextOpts) {
	n := float64(len([]rune(s)))
	tw := n * o.Size * 0.52
	if tw > w {
		tw = w
	}
	th := o.Size * 0.72
	tx := x
	switch o.Align {
	case alignCenter:
		tx = x + (w-tw)/2
	case alignRight:
		tx = x + w - tw
	}
	ty := y
	if o.VCenter {
		ty = y + (h-th)/2
	}
	c.RoundRect(tx, ty, tw, th, 2, o.Col)
}
