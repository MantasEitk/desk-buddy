package main

// The PC's apps, for the Kindle's "All apps" and the favourite tiles: every app Windows' own
// "All apps" shows (Store apps too), found in apps_windows.go, kept tidy here. Each app has a
// target: "shell:AppsFolder\<its id>", which opens it the way the Start menu does (or, if
// Windows wouldn't list them, the path of its Start menu shortcut).

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The names the server takes (the same rule as its /api/deck/icons).
var safeName = regexp.MustCompile(`^[\w .&'+()-]{1,60}$`)

// Not apps: uninstallers, readmes, help, manuals, web links and documents.
var (
	notAppName = regexp.MustCompile(`(?i)uninstall|readme|read me|\bhelp\b|documentation|manual|release notes|licen[cs]e|website|on the web|support center`)
	notAppFile = regexp.MustCompile(`(?i)\.(url|website|chm|hlp|txt|rtf|pdf|html?|md|ini|log|xml|docx?)$`)
)

// keepApp: whether an entry from the apps folder is an app someone would open from the Deck.
func keepApp(name, id string) bool {
	if !safeName.MatchString(name) || notAppName.MatchString(name) || id == "" {
		return false
	}
	low := strings.ToLower(id)
	return !notAppFile.MatchString(id) && !strings.HasPrefix(low, "http:") && !strings.HasPrefix(low, "https:")
}

// tidyApps keeps the real apps (the first of each name) as name → target.
func tidyApps(found [][2]string) map[string]string {
	out := map[string]string{}
	for _, f := range found {
		name, id := strings.TrimSpace(f[0]), f[1]
		if _, dup := out[name]; dup || !keepApp(name, id) {
			continue
		}
		out[name] = `shell:AppsFolder\` + id
	}
	return out
}

// scanShortcuts: the Start menu's shortcuts (what 0.7 listed), for when Windows won't list
// its apps folder.
func scanShortcuts() map[string]string {
	found := map[string]string{}
	for _, dir := range []string{
		filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs`),
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`),
	} {
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".lnk") {
				return nil
			}
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if _, dup := found[name]; !dup && keepApp(name, p) {
				found[name] = p
			}
			return nil
		})
	}
	return found
}

func sortedNames(apps map[string]string) []string {
	names := make([]string, 0, len(apps))
	for n := range apps {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Other names people (and the old favourites) use for an app.
var appAliases = map[string]string{
	"files": "file explorer", "explorer": "file explorer",
	"msedge": "edge", "msteams": "teams", "olk": "outlook",
	"vs code": "visual studio code", "vscode": "visual studio code", "code": "visual studio code",
}

// appKey: a name without its maker, its "(new)" and the like, ".exe" or a link's ":".
// "Google Chrome", "chrome.exe" and "chrome" all become "chrome".
func appKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(strings.TrimSuffix(s, ":"), ".exe")
	if i := strings.LastIndex(s, " ("); i > 0 && strings.HasSuffix(s, ")") {
		s = s[:i]
	}
	s = strings.TrimSuffix(s, " classic")
	for _, maker := range []string{"google ", "microsoft ", "mozilla ", "ms-"} {
		s = strings.TrimPrefix(s, maker)
	}
	if a, ok := appAliases[s]; ok {
		s = appKey(a)
	}
	return s
}

// matchApp finds an app by name, like the Mac finds "Chrome" as Google Chrome: the exact
// name first (any case), then the same app by its key.
func matchApp(names []string, want string) (string, bool) {
	want = strings.TrimSpace(want)
	if want == "" {
		return "", false
	}
	for _, n := range names {
		if strings.EqualFold(n, want) {
			return n, true
		}
	}
	k := appKey(want)
	if k == "" {
		return "", false
	}
	for _, n := range names {
		if appKey(n) == k {
			return n, true
		}
	}
	return "", false
}

// A favourite's win can also be the link that opens a well-known app ("spotify:"), when the
// app isn't on the list: only these, and never with anything after them. (An allow list on
// purpose: anything else has to be an app on the PC's own list.)
var appLinks = map[string]bool{
	"spotify:": true, "discord:": true, "steam:": true, "whatsapp:": true, "ms-settings:": true,
	"msteams:": true, "slack:": true, "zoommtg:": true,
}

// tileApp: a Deck tile's app from the list (by its win, then its label), as its target.
func tileApp(it deckItem, apps map[string]string) string {
	names := sortedNames(apps)
	for _, want := range []string{it.Win, it.Label} {
		if n, ok := matchApp(names, want); ok {
			return apps[n]
		}
	}
	return ""
}

// tileTarget: what to open for a Deck tile ("" if nothing safe): its app from the list, or
// the well-known app link in its win.
func tileTarget(it deckItem, apps map[string]string) string {
	if t := tileApp(it, apps); t != "" {
		return t
	}
	if w := strings.ToLower(strings.TrimSpace(it.Win)); appLinks[w] {
		return w
	}
	return ""
}
