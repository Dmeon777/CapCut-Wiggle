//go:build !windows

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestPreviewScreens(t *testing.T) {
	for _, sc := range []struct {
		name string
		st   *State
		s    float64
	}{
		{"welcome", &State{Screen: scrWelcome, Dir: `C:\Users\User\AppData\Local\Programs\CapCut Wiggle`, Desktop: true, Launch: true, Version: "1.0.0"}, 1},
		{"welcome_hidpi", &State{Screen: scrWelcome, Dir: `C:\Users\User\AppData\Local\Programs\CapCut Wiggle`, Desktop: true, Launch: false, Version: "1.0.0", Hover: "install"}, 1.5},
		{"progress", &State{Screen: scrProgress, Step: "Скачиваю Python", Detail: "4.1 из 10.7 МБ", Frac: 0.46, Version: "1.0.0"}, 1},
		{"done", &State{Screen: scrDone, Desktop: true, Version: "1.0.0"}, 1},
		{"error", &State{Screen: scrError, Err: "Нет связи с python.org. Проверь интернет и нажми «Повторить».", Version: "1.0.0"}, 1},
		{"uninstall", &State{Screen: scrUnConfirm, Uninstall: true, KeepData: true, OldVersion: "1.0.0"}, 1},
	} {
		w, h := int(logicalW*sc.s), int(logicalH*sc.s)
		c := &Canvas{W: w, H: h, Pix: make([]byte, w*h*4)}
		paint(c, sc.st, sc.s)
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < w*h; i++ {
			img.Set(i%w, i/w, color.RGBA{c.Pix[i*4+2], c.Pix[i*4+1], c.Pix[i*4], 255})
		}
		f, _ := os.Create("/tmp/rt/setup_" + sc.name + ".png")
		png.Encode(f, img)
		f.Close()
	}
}
