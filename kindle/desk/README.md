# desk — the Kindle app

A full-screen Go app for the Kindle Paperwhite 2 (jailbroken). It draws `/api/desk` in landscape straight to the e-ink panel (through `fbink`), reads taps from `/dev/input/event1`, and sends them to the backend.

## Build and install

```sh
./build.sh
```

Plug the Kindle into the Mac, then copy:

- `dist/desk/` → the Kindle's root, so it becomes `/mnt/us/desk/`
- `dist/documents/Desk Buddy.sh` → the Kindle's `documents/` folder

Eject, then open **Desk Buddy** from the library.

## Use

The first time, a new Kindle looks for the computer running Desk Buddy on the Wi-Fi: tap **Find it**, pick your computer and click **Allow** on it. After that it connects by itself, follows your computer if you move Desk Buddy to the cloud, and updates over Wi-Fi (Settings › Updates).

- **Power button**, or **hold the clock for 3 s**: back to the normal Kindle.
- Hard escape if anything ever hangs: hold the power button ~15 s to restart the Kindle.

## Settings

`/mnt/us/desk/config.json`: `server` (backend URL), `key` (filled in when you click Allow), `rotation` (`90` or `270`, depending on which way the Kindle stands), `full_refresh_every` (partial redraws between flashing full ones).

## How it hands over the screen (`run.sh`)

Before: screensaver off, top bar hidden, Kindle UI (`cvm`) paused. After the app exits for any reason, including a crash: all three put back, then Settings → Home to make the Kindle redraw. Log: `/mnt/us/desk/desk.log`.

## Test on the Mac

```sh
go run . -preview out.png -server http://127.0.0.1:8787   # from the live backend
go run . -preview out.png -json sample.json               # from saved /api/desk JSON
```

`fonts/Literata-{Medium,Bold,Display}.ttf` are static cuts of the variable `Literata.ttf` (Go's renderer can't pick weights in a variable font).
