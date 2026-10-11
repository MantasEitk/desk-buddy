package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestTidyApps(t *testing.T) {
	got := tidyApps([][2]string{
		{"Spotify", "SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify"},
		{"Calculator", "Microsoft.WindowsCalculator_8wekyb3d8bbwe!App"},
		{"Google Chrome", "Chrome"},
		{"Google Chrome", "{6D809377-6AF0-444B-8957-A3773F02200E}\\Google\\Chrome\\Application\\chrome.exe"}, // a second one
		{"Uninstall Foo", "{7C5A40EF-A0FB-4BFC-874A-C0F2E0B9FA8E}\\Foo\\uninst.exe"},
		{"Foo Readme", "{7C5A40EF-A0FB-4BFC-874A-C0F2E0B9FA8E}\\Foo\\readme.txt"},
		{"Foo Website", "https://foo.example.com"},
		{"Foo on the Web", "{7C5A40EF-A0FB-4BFC-874A-C0F2E0B9FA8E}\\Foo\\Foo.url"},
		{"Foo Guide", "{7C5A40EF-A0FB-4BFC-874A-C0F2E0B9FA8E}\\Foo\\guide.pdf"},
		{"Get Help", "Microsoft.GetHelp_8wekyb3d8bbwe!App"},
		{"Nameless", ""},
		{"Ünïcode", "x!App"}, // the server only takes plain names
		{"  Notepad ", "Microsoft.WindowsNotepad_8wekyb3d8bbwe!App"},
	})
	want := map[string]string{
		"Spotify":       `shell:AppsFolder\SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify`,
		"Calculator":    `shell:AppsFolder\Microsoft.WindowsCalculator_8wekyb3d8bbwe!App`,
		"Google Chrome": `shell:AppsFolder\Chrome`,
		"Notepad":       `shell:AppsFolder\Microsoft.WindowsNotepad_8wekyb3d8bbwe!App`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for n, w := range want {
		if got[n] != w {
			t.Errorf("%s: got %q, want %q", n, got[n], w)
		}
	}
}

func TestMatchApp(t *testing.T) {
	names := []string{"Calculator", "Discord", "File Explorer", "Google Chrome", "Microsoft Edge", "Microsoft Teams (work or school)", "Outlook (new)", "Settings", "Spotify", "Steam", "WhatsApp"}
	for want, n := range map[string]string{
		"spotify":       "Spotify",
		"spotify:":      "Spotify",
		"Chrome":        "Google Chrome",
		"chrome.exe":    "Google Chrome",
		"google chrome": "Google Chrome",
		"Files":         "File Explorer",
		"explorer.exe":  "File Explorer",
		"msedge.exe":    "Microsoft Edge",
		"Edge":          "Microsoft Edge",
		"Teams":         "Microsoft Teams (work or school)",
		"ms-teams:":     "Microsoft Teams (work or school)",
		"Outlook":       "Outlook (new)",
		"ms-settings:":  "Settings",
		"whatsapp:":     "WhatsApp",
		"steam:":        "Steam",
		"discord:":      "Discord",
		"Firefox":       "",
		"":              "",
		"Chrom":         "",
	} {
		got, ok := matchApp(names, want)
		if got != n || ok != (n != "") {
			t.Errorf("%q: got %q %v, want %q", want, got, ok, n)
		}
	}
}

func TestTileTarget(t *testing.T) {
	apps := map[string]string{"Spotify": `shell:AppsFolder\Spotify!App`, "Google Chrome": `shell:AppsFolder\Chrome`}
	for _, c := range []struct {
		label, win, want string
	}{
		{"Spotify", "spotify:", `shell:AppsFolder\Spotify!App`}, // the list first
		{"Chrome", "chrome.exe", `shell:AppsFolder\Chrome`},
		{"Google Chrome", "Google Chrome", `shell:AppsFolder\Chrome`}, // from "All apps"
		{"Discord", "discord:", "discord:"},                           // not on the list: the bare link
		{"Notepad++", "notepad++.exe", ""},                            // not a plain program name
		{"Paint", "mspaint.exe", ""},                                  // only apps on the list or known app links
		{"Steam", "steam://run/123", ""},                              // a link with something after it
		{"X", "Spotify: --x", ""},
		{"X", "C:", ""},
		{"X", `C:\Windows\notepad.exe`, ""},
		{"X", "cmd.exe", ""}, // never a shell
		{"X", "PowerShell.exe", ""},
		{"X", "ms-msdt:", ""},
		{"X", "search-ms:", ""},
		{"X", "file:", ""},
		{"X", "notepad.exe /p x", ""},
		{"X", "a.exe & calc", ""},
		{"Nothing", "", ""},
	} {
		if got := tileTarget(deckItem{Label: c.label, Win: c.win}, apps); got != c.want {
			t.Errorf("%s / %s: got %q, want %q", c.label, c.win, got, c.want)
		}
	}
}

func TestBitmapImage(t *testing.T) {
	// premultiplied: half-transparent red is (128, 0, 0, 128)
	img := bitmapImage([]byte{0, 0, 128, 128}, 1, 1)
	if _, ok := img.(*image.RGBA); !ok {
		t.Fatalf("premultiplied: got %T", img)
	}
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c.R < 250 || c.A != 128 {
		t.Fatalf("premultiplied red: %v", c)
	}
	// straight: a colour brighter than its alpha
	if c := color.NRGBAModel.Convert(bitmapImage([]byte{0, 0, 255, 128, 0, 0, 0, 0}, 2, 1).At(0, 0)).(color.NRGBA); c.R != 255 || c.A != 128 {
		t.Fatalf("straight red: %v", c)
	}
	// no alpha at all: opaque
	if _, _, _, a := bitmapImage([]byte{10, 20, 30, 0}, 1, 1).At(0, 0).RGBA(); a != 0xffff {
		t.Fatal("no alpha should be opaque")
	}
	if bitmapImage([]byte{1, 2, 3}, 1, 1) != nil {
		t.Fatal("too few bytes")
	}
}

func decode(t *testing.T, p []byte) *image.Gray {
	t.Helper()
	if p == nil {
		t.Fatal("no icon")
	}
	img, err := png.Decode(bytes.NewReader(p))
	if err != nil {
		t.Fatal(err)
	}
	g, ok := img.(*image.Gray)
	if !ok || g.Rect.Dx() != iconSide || g.Rect.Dy() != iconSide {
		t.Fatalf("want a %d×%d greyscale PNG, got %T %v", iconSide, iconSide, img, img.Bounds())
	}
	return g
}

func TestIconPNG(t *testing.T) {
	// a red square with a see-through margin, at Windows' biggest size: cropped, fitted,
	// darkest tone black, the margin white
	src := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 64; y < 192; y++ {
		for x := 64; x < 192; x++ {
			src.Set(x, y, color.NRGBA{200, 30, 30, 255})
		}
	}
	g := decode(t, iconPNG(src))
	if g.GrayAt(48, 48).Y != 0 || g.GrayAt(iconMargin+1, iconMargin+1).Y != 0 {
		t.Errorf("the square should fill the inside, in black: %d %d", g.GrayAt(48, 48).Y, g.GrayAt(iconMargin+1, iconMargin+1).Y)
	}
	if g.GrayAt(2, 2).Y != 255 || g.GrayAt(iconSide-3, iconSide-3).Y != 255 {
		t.Error("the margin should be white")
	}
	// a white shape made for a coloured tile: its shape, in black
	white := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			white.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	if g := decode(t, iconPNG(white)); g.GrayAt(48, 48).Y != 0 || g.GrayAt(2, 2).Y != 255 {
		t.Error("a white icon should come out black on white")
	}
	// grey tones keep their order
	grad := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			grad.Set(x, y, color.NRGBA{uint8(60 + x*2), uint8(60 + x*2), uint8(60 + x*2), 255})
		}
	}
	g = decode(t, iconPNG(grad))
	if !(g.GrayAt(20, 48).Y < g.GrayAt(48, 48).Y && g.GrayAt(48, 48).Y < g.GrayAt(75, 48).Y) {
		t.Error("a gradient should stay a gradient")
	}
	if iconPNG(image.NewNRGBA(image.Rect(0, 0, 16, 16))) != nil || iconPNG(nil) != nil {
		t.Error("nothing drawn: no icon")
	}
}

func TestCollectIcons(t *testing.T) {
	apps := map[string]string{"Google Chrome": "t:chrome", "Spotify": "t:spotify", "Notes": "t:none", strings.Repeat("Long", 11): "t:long"}
	asked := map[string]int{}
	icon := func(target string) []byte {
		asked[target]++
		if target == "t:none" {
			return nil
		}
		return []byte(target)
	}
	tiles := []deckItem{{Label: "Chrome", Win: "chrome.exe"}, {Label: "Discord", Win: "discord:"}, {Label: "Join", URL: ptr("https://zoom.us/j/1")}}
	got := collectIcons(tiles, apps, icon)
	want := map[string]string{"Chrome": "t:chrome", "Google Chrome": "t:chrome", "Spotify": "t:spotify"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != base64.StdEncoding.EncodeToString([]byte(v)) {
			t.Errorf("%s: got %q", k, got[k])
		}
	}
	if asked["discord:"] != 0 || asked["t:long"] != 0 {
		t.Error("only apps on the list, with names the server keeps")
	}
	// the total stays under the server's limit, favourites first
	big := func(string) []byte { return make([]byte, iconMax*3/4-10) }
	many := map[string]string{}
	for i := 0; i < 100; i++ {
		many[string(rune('A'+i%26))+strings.Repeat("x", i/26)] = "t"
	}
	got = collectIcons([]deckItem{{Label: "Fav", Win: "Axx"}}, many, big)
	total := 0
	for _, v := range got {
		total += len(v)
	}
	if total > iconsTotal || got["Fav"] == "" || len(got) > iconsCount {
		t.Fatalf("total %d, %d icons, favourite %v", total, len(got), got["Fav"] != "")
	}
}
