package main

// Рисование без системных библиотек: заливка, скруглённые прямоугольники и линии со сглаживанием.
// Буфер BGRA сверху вниз — его же Windows показывает через DIB-секцию.

import "math"

type RGB struct{ R, G, B uint8 }

func hex(v uint32) RGB { return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)} }

type Canvas struct {
	W, H int
	Pix  []byte
	plat canvasPlatform // текст рисует система (ClearType)
}

func (c *Canvas) Fill(col RGB) {
	for i := 0; i+3 < len(c.Pix); i += 4 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = col.B, col.G, col.R, 255
	}
}

func (c *Canvas) blend(x, y int, col RGB, a float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := (y*c.W + x) * 4
	c.Pix[i] = uint8(float64(c.Pix[i])*(1-a) + float64(col.B)*a + 0.5)
	c.Pix[i+1] = uint8(float64(c.Pix[i+1])*(1-a) + float64(col.G)*a + 0.5)
	c.Pix[i+2] = uint8(float64(c.Pix[i+2])*(1-a) + float64(col.R)*a + 0.5)
	c.Pix[i+3] = 255
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (c *Canvas) bbox(x, y, w, h float64) (int, int, int, int) {
	return clampInt(int(math.Floor(x)), 0, c.W), clampInt(int(math.Floor(y)), 0, c.H),
		clampInt(int(math.Ceil(x+w)), 0, c.W), clampInt(int(math.Ceil(y+h)), 0, c.H)
}

func sdRoundRect(px, py, x, y, w, h, r float64) float64 {
	cx, cy := x+w/2, y+h/2
	qx := math.Abs(px-cx) - (w/2 - r)
	qy := math.Abs(py-cy) - (h/2 - r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// RoundRect — залитый прямоугольник со скруглением r и мягким краем.
func (c *Canvas) RoundRect(x, y, w, h, r float64, col RGB) {
	if w <= 0 || h <= 0 {
		return
	}
	r = math.Min(r, math.Min(w, h)/2)
	x0, y0, x1, y1 := c.bbox(x, y, w, h)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			if a := 0.5 - sdRoundRect(float64(px)+0.5, float64(py)+0.5, x, y, w, h, r); a > 0 {
				c.blend(px, py, col, a)
			}
		}
	}
}

// Box — скруглённый прямоугольник с рамкой толщиной bw.
func (c *Canvas) Box(x, y, w, h, r, bw float64, border, fill RGB) {
	c.RoundRect(x, y, w, h, r, border)
	c.RoundRect(x+bw, y+bw, w-2*bw, h-2*bw, math.Max(0, r-bw), fill)
}

func distSeg(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l := dx*dx + dy*dy
	t := 0.0
	if l > 0 {
		t = math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l))
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

// Polyline — линия толщиной width с круглыми концами и сглаживанием.
func (c *Canvas) Polyline(pts [][2]float64, width float64, col RGB) {
	if len(pts) < 2 {
		return
	}
	minx, miny, maxx, maxy := pts[0][0], pts[0][1], pts[0][0], pts[0][1]
	for _, p := range pts {
		minx, miny = math.Min(minx, p[0]), math.Min(miny, p[1])
		maxx, maxy = math.Max(maxx, p[0]), math.Max(maxy, p[1])
	}
	pad := width/2 + 1
	x0, y0, x1, y1 := c.bbox(minx-pad, miny-pad, maxx-minx+2*pad, maxy-miny+2*pad)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			d := math.MaxFloat64
			for i := 1; i < len(pts); i++ {
				d = math.Min(d, distSeg(fx, fy, pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1]))
			}
			if a := width/2 - d + 0.5; a > 0 {
				c.blend(px, py, col, a)
			}
		}
	}
}

func bezier(p0, p1, p2, p3 [2]float64, n int) [][2]float64 {
	out := make([][2]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		a, b, cc, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		out = append(out, [2]float64{a*p0[0] + b*p1[0] + cc*p2[0] + d*p3[0], a*p0[1] + b*p1[1] + cc*p2[1] + d*p3[1]})
	}
	return out
}

// Logo — значок Wiggle (волна из icon.svg) в квадрате size×size.
func (c *Canvas) Logo(x, y, size float64, tile bool) {
	k := size / 32
	if tile {
		c.Box(x, y, size, size, 7*k, math.Max(1, k*0.6), hex(0x3a3a3a), hex(0x1c1c1c))
	}
	P := func(px, py float64) [2]float64 { return [2]float64{x + px*k, y + py*k} }
	segs := [][4][2]float64{
		{P(4, 17), P(7, 9), P(8.5, 9), P(10.5, 17)},
		{P(10.5, 17), P(12.5, 25), P(14, 25), P(16, 17)},
		{P(16, 17), P(18, 9), P(19.5, 9), P(21.5, 17)},
		{P(21.5, 17), P(23.5, 25), P(25, 25), P(28, 17)},
	}
	var pts [][2]float64
	for i, s := range segs {
		b := bezier(s[0], s[1], s[2], s[3], 14)
		if i > 0 {
			b = b[1:]
		}
		pts = append(pts, b...)
	}
	c.Polyline(pts, 2.6*k, colAccent)
}
