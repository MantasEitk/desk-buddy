<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/buddy-dark.png">
    <img src="docs/buddy.png" width="140" alt="Desk Buddy, a smiling book on little feet">
  </picture>
</p>

<h1 align="center">Desk Buddy</h1>

<p align="center"><b>Turn an old jailbroken Kindle into a calm desk companion.</b><br>Your tasks, calendar, weather, music and a clock, on e-ink, next to your computer.</p>

<p align="center">
<img src="docs/tasks.png" width="49%" alt="Tasks with today's calendar beside them"> <img src="docs/clock.png" width="49%" alt="Standby clock with the next event">
</p>

<p align="center"><a href="#what-you-need">What you need</a> · <a href="#set-up">Set up</a> · <a href="#compatibility">Compatibility</a> · <a href="#known-issues">Known issues</a> · <a href="docs/troubleshooting.md">Troubleshooting</a> · <a href="#privacy">Privacy</a></p>

---

## What it does

- **Tasks** with subtasks, a pinned **Now working on** card and a **Pomodoro** focus timer. Ticking a task ticks its subtasks.
- **A list per computer** (Mac / Windows) that follows the one you're using, or one list for everything.
- **Calendar**: today beside your tasks, a month view, and a heads-up before meetings with a **Join** button (Zoom, Meet, Teams, Webex and more).
- **Weather** for your town (today and the week).
- **Standby clocks** and **8 themes**: Plain, Garden, Sketch, Classic, Retro, Notebook, Typewriter, Game Boy.
- **Deck**: tap a tile on the Kindle to open an app on your computer.
- **Music**: what's playing in Spotify or Apple Music, album covers, play / pause / skip, volume, and which speakers it plays on.
- **Phone page**: see and change your tasks from your phone. Scan a QR code once.
- **Calm e-ink**: light and dark, four fonts, gentle screen refreshes (no needless flashing), a first-run tour.
- **Updates itself over Wi-Fi.** You tap Install; it never updates on its own.
- **Works offline**: the Kindle keeps showing your last screen, and changes you make wait until the server is back.

<p align="center"><img src="docs/calendar.png" width="49%" alt="Calendar tab"> <img src="docs/weather.png" width="49%" alt="Weather tab: now, the next 12 hours and the week"></p>

## What you need

- **A jailbroken touch Kindle.** Tested on the **Kindle Paperwhite 2** (see [Compatibility](#compatibility)). Jailbreak it with [kindlemodding.org](https://kindlemodding.org), including the hotfix.
- **A Mac** with macOS 13 Ventura or newer. (Windows is on the way.)
- **A USB cable, once**, to put Desk Buddy on the Kindle. After that it updates over Wi-Fi.
- **That's it.** No account, no subscription. Desk Buddy runs its own small server on your Mac.

Optional: your own free **Cloudflare** server, so the Kindle and your phone keep working while your Mac sleeps or you're away. See [docs/cloud.md](docs/cloud.md).

## Set up

About 15 minutes, once.

1. **Mac**: download *Desk Buddy for Mac* from [Releases](../../releases), move it to Applications and open it. It walks you through what it needs. → [docs/install-mac.md](docs/install-mac.md)
2. **Kindle**: copy Desk Buddy onto the Kindle over USB, eject, and open **Desk Buddy** from the library. → [docs/install-kindle.md](docs/install-kindle.md)
3. **Connect them**: on the Kindle tap **Find it**, pick your Mac, and click **Allow** on the Mac.

From then on: everything happens on the Mac (Desk Buddy in the menu bar) or on the Kindle itself, and new versions arrive over Wi-Fi.

## Compatibility

| | |
|---|---|
| **Kindle Paperwhite 2** (6th gen, 2013), firmware 5.12.x | ✅ Tested, daily use |
| Other touch Kindles (Paperwhite 3 / 4 / 5, Oasis, Basic) | ❌ Not yet. The layout is made for the PW2's screen (1024 × 758) and touch panel. Support for other screens is planned. |
| Kindles without a touchscreen | ❌ No |
| Jailbreak | The [kindlemodding.org](https://kindlemodding.org) jailbreak **with its hotfix** (it provides FBInk, the root helper and the script launcher Desk Buddy uses). KUAL isn't needed. |
| Kindle Wi-Fi | 2.4 GHz, WPA2 (the Paperwhite 2 can't join 5 GHz-only or WPA3-only networks) |
| **Mac** | macOS 13 Ventura or newer. Apple Silicon: tested. Intel: the release build is universal, but untested. |
| **Windows** | 🚧 Coming |
| Phone page | Any modern phone browser (iPhone, Android) |
| Music | Spotify and Apple Music desktop apps on the Mac |
| Calendar | Everything in the Mac's Calendar app (iCloud, Google, Exchange, … accounts added there), plus calendar links (.ics) added on the phone page |
| Own cloud server | Cloudflare's free plan (Workers + D1) |

## How it works

```
 Kindle (draws the screen)  ⇄  Desk Buddy server  ⇄  Desk Buddy for Mac (menu bar)
                                on your Mac, or in       tells it which computer you're on,
                                your own Cloudflare      sends your calendar and music,
                                account                  opens Deck apps
```

The server keeps everything (tasks, settings, calendar, weather). The Kindle asks it for one screen at a time and sends your taps back; the Mac app connects your computer. Switching between your Mac and the cloud happens in the Mac app, and the Kindle follows by itself.

## Known issues

- **Only the Paperwhite 2** works for now (see [Compatibility](#compatibility)).
- **The first install needs USB.** On macOS 26, USB with Kindles can be flaky (the drive disconnects, or files you copied don't stick). Copy, eject properly, and check the files are there; tips in [docs/install-kindle.md](docs/install-kindle.md).
- **The Mac app isn't signed** with an Apple Developer ID yet: the first time you open it, right-click › **Open**. macOS may also ask for its permissions again after each update.
- **On "This Mac"** (the default), the Kindle only updates while your Mac is awake, on the same Wi-Fi, with Desk Buddy open. It shows your last screen meanwhile and catches up when the Mac is back. Want it always on? Use your own [cloud server](docs/cloud.md).
- **After the Kindle restarts**, open Desk Buddy from the library again (it doesn't start by itself).
- **Weak Wi-Fi at the desk** makes the Kindle slow or offline. Restarting the router has fixed it more than once.
- **Spotify** sometimes reports nothing playing through its AppleScript; play / pause once and it catches up. Music playing on another device (Spotify Connect) doesn't show.
- **Apple Music covers** only show for songs that have artwork stored on your Mac.
- **Speaker switching** changes your Mac's sound output, not Spotify Connect devices.
- **Windows** isn't ready yet.

Something else? [Open an issue](../../issues).

## Privacy

Your data only goes to **your own** Desk Buddy server: on your Mac (the default), or in your own Cloudflare account if you set one up. There's no Desk Buddy company server, no ads, no tracking, no analytics, and nothing is sold or shared.

Two things leave your network on purpose: the weather asks [Open-Meteo](https://open-meteo.com) for your town's forecast, and some Spotify covers come from Spotify's public link preview.

## More

- [docs/install-mac.md](docs/install-mac.md): installing and using the Mac app
- [docs/install-kindle.md](docs/install-kindle.md): putting Desk Buddy on the Kindle, using it, updating, removing it
- [docs/cloud.md](docs/cloud.md): your own free cloud server (optional)
- [docs/troubleshooting.md](docs/troubleshooting.md): when something doesn't work
- [docs/building.md](docs/building.md): building it yourself, and how the code is laid out
- [CONTRIBUTING.md](CONTRIBUTING.md): ideas, bugs and pull requests

## Credits

[FBInk](https://github.com/NiLuJe/FBInk) by NiLuJe, [KindleModding](https://kindlemodding.org) (jailbreak and the script launcher), [Open-Meteo](https://open-meteo.com), [ical.js](https://github.com/kewisch/ical.js), [go-qrcode](https://github.com/skip2/go-qrcode). Fonts, all under the SIL Open Font License (texts in `kindle/desk/fonts/`): Atkinson Hyperlegible, Literata, Inter, Bitter, Courier Prime, Press Start 2P. Built with [Claude Code](https://claude.com/claude-code).

"Kindle" is a trademark of Amazon; Spotify and Apple Music belong to their owners. Desk Buddy isn't affiliated with or endorsed by any of them.

## License

[MIT](LICENSE)
