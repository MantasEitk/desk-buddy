//go:build windows

package main

// What's playing, from Windows' own media controls (the ones behind the volume pop-up and the
// media keys): Spotify, Apple Music, browsers and other players all show up there. Also play /
// pause / skip and the cover. Direct Windows calls only, on the media thread (com_windows.go).

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation/collections"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

type (
	smtcManager = control.GlobalSystemMediaTransportControlsSessionManager
	smtcSession = control.GlobalSystemMediaTransportControlsSession
	smtcProps   = control.GlobalSystemMediaTransportControlsSessionMediaProperties
)

// Only touched on the media thread.
var (
	media       *smtcManager
	mediaFailAt time.Time
	shownID     string // the app id of the session shown last (its cover is read from it)
)

func mediaManager() (*smtcManager, error) {
	if media != nil {
		return media, nil
	}
	if time.Since(mediaFailAt) < time.Minute { // don't keep asking when Windows said no
		return nil, errors.New("Windows' media controls aren't available")
	}
	p, err := await(control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync())
	if err != nil || p == nil {
		mediaFailAt = time.Now()
		if err == nil {
			err = errors.New("no media controls")
		}
		return nil, err
	}
	media = (*smtcManager)(p)
	return media, nil
}

type session struct {
	s    *smtcSession
	id   string // like "Spotify.exe"
	info sessionInfo
}

// sessions lists the apps with media right now (release them with releaseAll), and which
// one Windows itself shows (-1 if none).
func sessions() ([]session, int, error) {
	m, err := mediaManager()
	if err != nil {
		return nil, -1, err
	}
	view, err := listSessions(m)
	if err != nil || view == nil { // e.g. Explorer restarted: ask for the media controls again next time
		m.Release()
		media = nil
		if err == nil {
			err = errors.New("no media sessions")
		}
		return nil, -1, err
	}
	defer view.Release()
	n, err := view.GetSize()
	if err != nil {
		return nil, -1, err
	}
	var list []session
	for i := uint32(0); i < n && i < 20; i++ {
		p, err := view.GetAt(i)
		if err != nil || p == nil {
			continue
		}
		s := (*smtcSession)(p)
		id, _ := s.GetSourceAppUserModelId()
		playing := false
		if pi, err := s.GetPlaybackInfo(); err == nil && pi != nil {
			st, _ := pi.GetPlaybackStatus()
			playing = st == control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying
			pi.Release()
		}
		list = append(list, session{s: s, id: id, info: sessionInfo{App: appName(id), Playing: playing}})
	}
	cur := -1
	if c, err := m.GetCurrentSession(); err == nil && c != nil {
		id, _ := c.GetSourceAppUserModelId()
		c.Release()
		for i := range list {
			if list[i].id == id {
				cur = i
			}
		}
	}
	return list, cur, nil
}

// listSessions asks for the session list (the bindings panic when the media controls have gone
// away, e.g. after Explorer restarted).
func listSessions(m *smtcManager) (v *collections.IVectorView, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("media controls: %v", r)
		}
	}()
	return m.GetSessions()
}

func releaseAll(list []session) {
	for _, s := range list {
		s.s.Release()
	}
}

// pick: the session to show or control (see pickSession), or nil.
func pick(list []session, cur int, pref, last string) *session {
	infos := make([]sessionInfo, len(list))
	for i := range list {
		infos[i] = list[i].info
	}
	if i := pickSession(infos, cur, pref, last); i >= 0 {
		return &list[i]
	}
	return nil
}

func properties(s *smtcSession) (*smtcProps, error) {
	p, err := await(s.TryGetMediaPropertiesAsync())
	if err != nil || p == nil {
		return nil, errors.New("no song details")
	}
	return (*smtcProps)(p), nil
}

// nowPlaying: the app and song, like the Mac's now_playing ({running: false} when no media app
// is open).
func nowPlaying(pref, last string) (st musicState, err error) {
	if e := onCOM(func() {
		var list []session
		var cur int
		list, cur, err = sessions()
		if err != nil {
			return
		}
		defer releaseAll(list)
		s := pick(list, cur, pref, last)
		if s == nil {
			shownID = ""
			return
		}
		shownID = s.id
		st = musicState{App: s.info.App, Running: true, Playing: s.info.Playing}
		p, e := properties(s.s)
		if e != nil {
			return // open, nothing loaded (or it didn't say)
		}
		defer p.Release()
		st.Title, _ = p.GetTitle()
		st.Artist, _ = p.GetArtist()
		st.Album, _ = p.GetAlbumTitle()
		if st.Title == "" {
			st.Playing = false
		}
	}); e != nil {
		return musicState{}, e
	}
	return st, err
}

// mediaControl: play / pause, next or previous in the app shown.
func mediaControl(cmd, pref, last string) (err error) {
	if e := onCOM(func() {
		var list []session
		var cur int
		list, cur, err = sessions()
		if err != nil {
			return
		}
		defer releaseAll(list)
		s := pick(list, cur, pref, last)
		if s == nil {
			err = errNoMusic
			return
		}
		switch cmd {
		case "next":
			_, err = await(s.s.TrySkipNextAsync())
		case "previous":
			_, err = await(s.s.TrySkipPreviousAsync())
		default:
			_, err = await(s.s.TryTogglePlayPauseAsync())
		}
	}); e != nil {
		return e
	}
	return err
}

// coverArt: the cover of the song shown, as Windows has it (usually a JPEG or PNG).
func coverArt() (img []byte, err error) {
	if e := onCOM(func() {
		var list []session
		list, _, err = sessions()
		if err != nil {
			return
		}
		defer releaseAll(list)
		var s *smtcSession
		for i := range list {
			if list[i].id == shownID && shownID != "" {
				s = list[i].s
			}
		}
		if s == nil {
			err = errNoMusic
			return
		}
		p, e := properties(s)
		if e != nil {
			err = e
			return
		}
		defer p.Release()
		ref, e := p.GetThumbnail()
		if e != nil || ref == nil {
			err = errors.New("no cover")
			return
		}
		defer ref.Release()
		stream, e := await(ref.OpenReadAsync())
		if e != nil || stream == nil {
			err = errors.New("couldn't open the cover")
			return
		}
		defer comRelease(stream)
		img, err = readStream(stream)
	}); e != nil {
		return nil, e
	}
	return img, err
}

var (
	shcore                                 = syscall.NewLazyDLL("shcore.dll")
	procCreateStreamOverRandomAccessStream = shcore.NewProc("CreateStreamOverRandomAccessStream")
	iidIStream                             = ole.NewGUID("{0000000C-0000-0000-C000-000000000046}")
)

// readStream reads a Windows picture stream (up to 2 MB) through the plain stream Windows makes
// over it.
func readStream(stream unsafe.Pointer) ([]byte, error) {
	if err := procCreateStreamOverRandomAccessStream.Find(); err != nil {
		return nil, err
	}
	var is unsafe.Pointer
	hr, _, _ := procCreateStreamOverRandomAccessStream.Call(uintptr(stream), uintptr(unsafe.Pointer(iidIStream)), uintptr(unsafe.Pointer(&is)))
	if int32(hr) < 0 || is == nil {
		return nil, ole.NewError(hr)
	}
	defer comRelease(is)
	var out []byte
	buf := make([]byte, 64*1024)
	for len(out) < 2_000_000 {
		var n uint32
		if err := comCall(is, 3, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n))); err != nil { // Read
			return nil, err
		}
		if n == 0 {
			break
		}
		out = append(out, buf[:n]...)
	}
	return out, nil
}
