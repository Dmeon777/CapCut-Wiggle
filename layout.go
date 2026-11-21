package main

// Экраны установщика: что где нарисовано и какие места кликабельны. Размеры — в логических
// пикселях, s — масштаб экрана (1 = 96 dpi).

import (
	"fmt"
	"math"
	"sync"
)

var (
	colBg      = hex(0x1c1c1c)
	colTitle   = hex(0x161616)
	colLine    = hex(0x2c2c2c)
	colField   = hex(0x141414)
	colBorder  = hex(0x3a3a3a)
	colText    = hex(0xececec)
	colMuted   = hex(0x9a9a9a)
	colFaint   = hex(0x6b6b6b)
	colAccent  = hex(0xeea83a)
	colAccentH = hex(0xf6bc5c)
	colOnAcc   = hex(0x1a1a1a)
	colGhost   = hex(0x2a2a2a)
	colGhostH  = hex(0x343434)
	colError   = hex(0xef6a5f)
	colCloseH  = hex(0xc42b1c)
)

const (
	logicalW = 620.0
	logicalH = 430.0
	titleH   = 40.0
)

type Screen int

const (
	scrWelcome Screen = iota
	scrProgress
	scrDone
	scrError
	scrUnConfirm
	scrUnProgress
	scrUnDone
)

type State struct {
	mu                        sync.Mutex
	Screen                    Screen
	Uninstall                 bool
	Dir                       string
	Desktop, Launch, KeepData bool
	Update                    bool
	Version, OldVersion       string
	Step, Detail              string
	Frac                      float64
	Err                       string
	Launched                  bool
	Hover, Pressed            string
}

type Hit struct {
	X, Y, W, H float64
	ID         string
}

func (h Hit) has(x, y float64) bool { return x >= h.X && y >= h.Y && x < h.X+h.W && y < h.Y+h.H }

// paint рисует текущий экран и возвращает кликабельные места (в пикселях экрана).
func paint(c *Canvas, st *State, s float64) []Hit {
	L := func(v float64) float64 { return math.Round(v * s) }
	W, H := float64(c.W), float64(c.H)
	var hits []Hit
	hit := func(id string, x, y, w, h float64) { hits = append(hits, Hit{x, y, w, h, id}) }

	c.Fill(colBg)
	// строка заголовка: значок, название, свернуть и закрыть
	c.RoundRect(0, 0, W, L(titleH), 0, colTitle)
	c.RoundRect(0, L(titleH), W, math.Max(1, L(1)), 0, colLine)
	c.Logo(L(14), L(11), L(18), false)
	title := "Установка CapCut Wiggle"
	if st.Uninstall {
		title = "Удаление CapCut Wiggle"
	}
	c.Text(title, L(42), 0, L(300), L(titleH), TextOpts{Size: 12.5 * s, Col: colMuted, VCenter: true})
	bw := L(46)
	for i, id := range []string{"min", "close"} {
		x := W - bw*float64(2-i)
		if st.Hover == id {
			col := colGhostH
			if id == "close" {
				col = colCloseH
			}
			c.RoundRect(x, 0, bw, L(titleH), 0, col)
		}
		cx, cy := x+bw/2, L(titleH)/2
		ic := colMuted
		if st.Hover == id {
			ic = colText
		}
		if id == "close" {
			d := L(5)
			c.Polyline([][2]float64{{cx - d, cy - d}, {cx + d, cy + d}}, math.Max(1, 1.2*s), ic)
			c.Polyline([][2]float64{{cx + d, cy - d}, {cx - d, cy + d}}, math.Max(1, 1.2*s), ic)
		} else {
			c.Polyline([][2]float64{{cx - L(5), cy}, {cx + L(5), cy}}, math.Max(1, 1.2*s), ic)
		}
		hit(id, x, 0, bw, L(titleH))
	}

	// шапка: логотип, название, описание
	c.Logo(L(32), L(68), L(72), true)
	c.Text("CapCut Wiggle", L(124), L(66), L(460), L(36), TextOpts{Size: 26 * s, Weight: 600, Col: colText, VCenter: true})
	c.Text("Рысканье, углы, эффекты и обложки для CapCut", L(125), L(102), L(460), L(20), TextOpts{Size: 13.5 * s, Col: colMuted, VCenter: true})
	ver := "Версия " + st.Version
	if st.Update && st.OldVersion != "" && st.OldVersion != st.Version {
		ver = fmt.Sprintf("Обновление %s → %s", st.OldVersion, st.Version)
	} else if st.Update {
		ver = "Переустановка " + st.Version
	}
	if st.Uninstall {
		ver = "Установлена версия " + st.OldVersion
	}
	c.Text(ver, L(125), L(122), L(460), L(18), TextOpts{Size: 12 * s, Col: colFaint, VCenter: true})

	button := func(id, label string, x, y, w float64, primary, enabled bool) {
		h := L(34)
		fill, tc := colGhost, colText
		if primary {
			fill, tc = colAccent, colOnAcc
		}
		if enabled && st.Hover == id {
			if primary {
				fill = colAccentH
			} else {
				fill = colGhostH
			}
		}
		if !enabled {
			fill, tc = colGhost, colFaint
		}
		c.RoundRect(x, y, w, h, L(4), fill)
		weight := 400
		if primary {
			weight = 600
		}
		c.Text(label, x, y, w, h, TextOpts{Size: 13 * s, Weight: weight, Col: tc, Align: alignCenter, VCenter: true})
		if enabled {
			hit(id, x, y, w, h)
		}
	}
	check := func(id, label string, x, y float64, on bool) {
		b := L(18)
		if on {
			c.RoundRect(x, y, b, b, L(3), colAccent)
			k := b / 18
			c.Polyline([][2]float64{{x + 4.5*k, y + 9.2*k}, {x + 7.6*k, y + 12.3*k}, {x + 13.6*k, y + 5.8*k}}, 2*k, colOnAcc)
		} else {
			bc := colBorder
			if st.Hover == id {
				bc = colMuted
			}
			c.Box(x, y, b, b, L(3), math.Max(1, s), bc, colField)
		}
		c.Text(label, x+b+L(10), y-L(3), L(360), b+L(6), TextOpts{Size: 13.5 * s, Col: colText, VCenter: true})
		hit(id, x, y-L(4), b+L(10)+L(300), b+L(8))
	}
	footer := func() float64 {
		y := H - L(64)
		c.RoundRect(0, y, W, math.Max(1, L(1)), 0, colLine)
		return y + L(15)
	}
	bar := func(y float64) {
		c.RoundRect(L(32), y, W-L(64), L(6), L(3), colGhost)
		if f := math.Max(0, math.Min(1, st.Frac)); f > 0 {
			c.RoundRect(L(32), y, math.Max(L(6), (W-L(64))*f), L(6), L(3), colAccent)
		}
	}

	switch st.Screen {
	case scrWelcome:
		c.Text("Папка установки", L(32), L(176), L(300), L(18), TextOpts{Size: 12 * s, Col: colMuted, VCenter: true})
		fw := W - L(64) - L(112)
		c.Box(L(32), L(198), fw, L(34), L(4), math.Max(1, s), colBorder, colField)
		c.Text(st.Dir, L(44), L(198), fw-L(24), L(34), TextOpts{Size: 13 * s, Col: colText, VCenter: true, PathEllipsis: true})
		button("browse", "Изменить…", W-L(32)-L(100), L(198), L(100), false, true)
		check("desktop", "Ярлык на рабочем столе", L(32), L(256), st.Desktop)
		check("launch", "Запустить после установки", L(32), L(290), st.Launch)
		y := footer()
		c.Text("Скачает программу и Python (~12 МБ)", L(32), y, L(300), L(34), TextOpts{Size: 12 * s, Col: colFaint, VCenter: true})
		label := "Установить"
		if st.Update {
			label = "Обновить"
		}
		button("install", label, W-L(32)-L(132), y, L(132), true, true)
		button("cancel", "Отмена", W-L(32)-L(132)-L(10)-L(96), y, L(96), false, true)

	case scrProgress, scrUnProgress:
		c.Text(st.Step, L(32), L(184), W-L(64)-L(60), L(24), TextOpts{Size: 15 * s, Weight: 600, Col: colText, VCenter: true})
		c.Text(fmt.Sprintf("%d%%", int(st.Frac*100+0.5)), W-L(32)-L(60), L(184), L(60), L(24), TextOpts{Size: 13 * s, Col: colMuted, VCenter: true, Align: alignRight})
		bar(L(220))
		c.Text(st.Detail, L(32), L(236), W-L(64), L(20), TextOpts{Size: 12 * s, Col: colMuted, VCenter: true, Ellipsis: true})
		y := footer()
		c.Text("Не закрывай окно — это займёт меньше минуты.", L(32), y, W-L(64), L(34), TextOpts{Size: 12 * s, Col: colFaint, VCenter: true})

	case scrDone:
		c.Text("Готово", L(32), L(176), L(400), L(32), TextOpts{Size: 22 * s, Weight: 600, Col: colAccent, VCenter: true})
		msg := "CapCut Wiggle установлен в меню «Пуск»"
		if st.Desktop {
			msg += " и на рабочий стол"
		}
		msg += "."
		if st.Launched {
			msg += " Окно программы сейчас откроется."
		}
		c.Text(msg, L(32), L(214), W-L(64), L(48), TextOpts{Size: 13.5 * s, Col: colText, Wrap: true})
		c.Text("ffmpeg для рендера программа скачает сама при первом открытии вкладки «Углы».", L(32), L(262), W-L(64), L(40), TextOpts{Size: 12 * s, Col: colMuted, Wrap: true})
		y := footer()
		if st.Launched {
			button("done", "Закрыть", W-L(32)-L(132), y, L(132), true, true)
		} else {
			button("run", "Запустить", W-L(32)-L(132), y, L(132), true, true)
			button("done", "Закрыть", W-L(32)-L(132)-L(10)-L(96), y, L(96), false, true)
		}

	case scrError:
		c.Text("Не получилось", L(32), L(176), L(400), L(30), TextOpts{Size: 20 * s, Weight: 600, Col: colError, VCenter: true})
		c.Text(st.Err, L(32), L(212), W-L(64), L(110), TextOpts{Size: 13 * s, Col: colText, Wrap: true})
		y := footer()
		button("retry", "Повторить", W-L(32)-L(132), y, L(132), true, true)
		button("cancel", "Закрыть", W-L(32)-L(132)-L(10)-L(96), y, L(96), false, true)

	case scrUnConfirm:
		c.Text("Удалить CapCut Wiggle с компьютера?", L(32), L(176), W-L(64), L(28), TextOpts{Size: 16 * s, Weight: 600, Col: colText, VCenter: true})
		c.Text("Уберу программу, ярлыки и скачанный ffmpeg. Проекты CapCut и готовые рендеры не трогаю.", L(32), L(208), W-L(64), L(40), TextOpts{Size: 12.5 * s, Col: colMuted, Wrap: true})
		check("keep", "Оставить мои пресеты и настройки", L(32), L(262), st.KeepData)
		y := footer()
		button("uninstall", "Удалить", W-L(32)-L(132), y, L(132), true, true)
		button("cancel", "Отмена", W-L(32)-L(132)-L(10)-L(96), y, L(96), false, true)

	case scrUnDone:
		c.Text("CapCut Wiggle удалён", L(32), L(176), W-L(64), L(32), TextOpts{Size: 20 * s, Weight: 600, Col: colAccent, VCenter: true})
		msg := "Спасибо, что пользовался!"
		if st.KeepData {
			msg = "Пресеты и настройки остались в папке программы — поставишь заново, и они подхватятся."
		}
		c.Text(msg, L(32), L(214), W-L(64), L(48), TextOpts{Size: 13 * s, Col: colText, Wrap: true})
		y := footer()
		button("done", "Закрыть", W-L(32)-L(132), y, L(132), true, true)
	}
	return hits
}
