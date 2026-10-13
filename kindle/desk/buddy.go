package main

// Desk Buddy himself (brand/desk-buddy.svg, drawn as buddy.png): on the connecting and
// updating screens, in About and the tour, and on the Kindle library's cover for the app.
//
// The library item: the Kindle lists any .sh in documents/ as a book (the jailbreak's
// sh_integration), and its "# Icon:" header sets the cover. On start, the app draws the
// cover (cover.png) and writes documents/Desk Buddy.sh pointing at it. No USB needed.

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
)

//go:embed buddy.png
var buddyPNG []byte

var (
	buddyOnce sync.Once
	buddyMask *image.Alpha // his shape (black on clear)
)

// drawBuddy draws Desk Buddy h pixels tall, centred on cx with his top at top, in colour c.
func drawBuddy(img *image.Gray, cx, top, h int, c uint8) image.Rectangle {
	buddyOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(buddyPNG))
		if err != nil {
			return
		}
		b := m.Bounds()
		buddyMask = image.NewAlpha(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := m.At(x, y).RGBA()
				buddyMask.Pix[(y-b.Min.Y)*buddyMask.Stride+(x-b.Min.X)] = uint8(a >> 8)
			}
		}
	})
	if buddyMask == nil {
		return image.Rectangle{}
	}
	w := h * buddyMask.Rect.Dx() / buddyMask.Rect.Dy()
	x0 := cx - w/2
	s := scaleOf(img) // drawn at the canvas's resolution
	pw, ph := px(w, s), px(h, s)
	small := image.NewAlpha(image.Rect(0, 0, pw, ph))
	xdraw.CatmullRom.Scale(small, small.Rect, buddyMask, buddyMask.Rect, xdraw.Src, nil)
	ox, oy := px(x0, s), px(top, s)
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			if a := small.Pix[y*small.Stride+x]; a > 0 {
				blendPx(img, ox+x, oy+y, c, float64(a)/255)
			}
		}
	}
	return image.Rect(x0, top, x0+w, top+h)
}

// coverImage: the library cover: Desk Buddy, then "Desk" / "Buddy", on white (portrait).
func coverImage() *image.Gray {
	was := themeName
	themeName = "" // the cover is always in the plain style
	defer func() { themeName = was }()
	const w, h = 600, 960 // a book cover's shape (1 : 1.6), so it's as tall as the others in the library
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = paper
	}
	drawBuddy(img, w/2, 175, 360, ink)
	f := face("uiBold", 110)
	drawCentered(img, f, w/2, 655, "Desk", ink)
	drawCentered(img, f, w/2, 781, "Buddy", ink)
	return img
}

// welcomeImage: the welcome screen shown for a moment each time the app starts (landscape),
// before the last-known screen. Always the plain style; inverted in dark mode.
func welcomeImage(dark bool) *image.Gray {
	was := themeName
	themeName = ""
	defer func() { themeName = was }()
	img := newCanvas(LW, LH)
	drawBuddy(img, LW/2, 120, 300, ink)
	drawCentered(img, face("ui", 24), LW/2, 498, "Welcome to", grey2)
	drawCentered(img, face("uiBold", 56), LW/2, 568, "Desk Buddy", ink)
	if dark {
		invertRect(img, img.Rect)
	}
	return img
}

//go:embed kindle-files/launcher.sh
var launcherSH string

// ensureLibraryItem keeps the Kindle's library item for the app named Desk Buddy, with its
// cover. Only on the Kindle (documents/ exists). Safe order: new item first, old one after.
func ensureLibraryItem(appDir string) {
	docs := "/mnt/us/documents"
	if _, err := os.Stat(docs); err != nil {
		return
	}
	cover := filepath.Join(appDir, "cover.png")
	var buf bytes.Buffer
	if png.Encode(&buf, coverImage()) == nil {
		if old, _ := os.ReadFile(cover); !bytes.Equal(old, buf.Bytes()) {
			if err := os.WriteFile(cover, buf.Bytes(), 0o644); err != nil {
				log.Println("cover:", err)
				return
			}
		}
	}
	// the item: the launcher, with the cover line after its name and author
	item := strings.Replace(launcherSH, "# DontUseFBInk", "# Icon: "+cover+"\n# DontUseFBInk", 1)
	path := filepath.Join(docs, "Desk Buddy.sh")
	if cur, _ := os.ReadFile(path); string(cur) != item {
		if err := os.WriteFile(path+".new", []byte(item), 0o755); err != nil {
			log.Println("library item:", err)
			return
		}
		if err := os.Rename(path+".new", path); err != nil {
			log.Println("library item:", err)
			return
		}
		syncFS()
	}
}
