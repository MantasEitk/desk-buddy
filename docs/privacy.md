# Privacy: what Desk Buddy sees, and where it goes

Giving an app access to your calendar and music, and a jailbroken Kindle on your desk, can feel like a lot. This page explains exactly what Desk Buddy reads, why, and where it goes, so you don't have to take it on trust. The code is all here to check.

## The short version

- **It's a closed loop.** Desk Buddy is three parts that only talk to each other: your **Kindle**, **Desk Buddy for Mac**, and **your own Desk Buddy server**. Your phone can join in if you scan the QR code.
- **The server is yours.** By default it runs **on your Mac**, so your data never leaves your home Wi-Fi. If you choose, it runs in **your own** Cloudflare account instead. There's no Desk Buddy company server, no account to make, nothing to sign up for.
- **No ads, no tracking, no analytics, no telemetry, no crash reports sent anywhere.** Nothing is sold or shared, because there's nobody to send it to.
- **Two small exceptions, on purpose:** the weather (your town goes to Open-Meteo) and some album covers (from Spotify). Details below.

```
                         your home Wi-Fi
   ┌──────────────────────────────────────────────────────────┐
   │                                                          │
   │   Kindle  ⇄  Desk Buddy server  ⇄  Desk Buddy for Mac    │
   │                 (on your Mac)                            │
   │                      ⇅                                   │
   │                 your phone (optional)                    │
   └──────────────────────────────────────────────────────────┘
          only outside: weather (Open-Meteo), some covers (Spotify)
```

With your own cloud server, the middle box moves into your Cloudflare account; the rest is the same.

## Where your data lives

| Where Desk Buddy runs | Your data is stored in |
|---|---|
| **This Mac** (the default) | `~/Library/Application Support/DeskBuddy/` on your Mac (a small database file) |
| **Your own cloud** (optional) | A D1 database in **your** Cloudflare account. Only you can see it, and the server only answers requests that carry your key. |

The Kindle keeps a copy of its last screen and its own settings on itself, so it can show something while offline.

## Each permission, and why

macOS asks before Desk Buddy can do any of these. Every one is optional: say no and only that feature is missing. You can change your mind any time in Desk Buddy › **Permissions**, or System Settings › Privacy & Security.

### Calendars

- **Why:** to show today's events beside your tasks, the month view, and the heads-up before a meeting (with a Join button).
- **What it reads:** your events from 40 days back to 100 days ahead, from the calendars in the Mac's Calendar app.
- **What it sends, and where:** each event's title, time, place, notes, calendar name and link, **to your own Desk Buddy server only**. The notes are read so it can find meeting links (Zoom, Meet, Teams…).
- **What it never does:** change, add or delete anything in your calendars. Desk Buddy only reads them.

### Controlling Spotify and Music (Automation)

- **Why:** to show what's playing on the Kindle, with the cover, and to make its play / pause / skip and volume buttons work.
- **What it reads:** the song, artist, album, whether it's playing, the volume, and the cover. **Only while the Music screen is open on the Kindle**; otherwise it doesn't look at all.
- **What it sends:** that information, to your own Desk Buddy server.
- **What it never does:** read your library, playlists or history, or control anything you didn't tap.

### Bluetooth

- **Why:** when you pick your headphones on the Kindle's music screen, Desk Buddy can connect them.
- **What it reads:** the names of your Mac's sound outputs and paired Bluetooth audio devices.
- **What it never does:** scan for, pair with or connect to anything you didn't pick.

### Incoming connections (a one-time macOS question)

- **Why:** with Desk Buddy running on your Mac, the Kindle and your phone need to reach it over your Wi-Fi.
- It only answers devices that have **your key**. A new Kindle can only get the key if you click **Allow** on your Mac (and the code on both screens matches).

### What it never asks for

No microphone, camera, screen recording, contacts, photos, files, location (the weather uses the town you type), passwords, or "Accessibility" control of your Mac.

## Other things it does on your Mac

- **"Are you at this Mac?"** It checks how long ago you last touched the keyboard or mouse (just a number of seconds, never what you typed) and whether the Mac is in light or dark mode, so the Kindle shows the right list and theme. It tells your server when that changes, and every 2 minutes.
- **The Deck:** when you tap a tile on the Kindle, it opens **that app by name**, or a meeting link from Zoom, Meet, Teams and similar services. It never runs commands or opens anything else.
- **App icons for the Deck:** it sends the names and small greyscale icons of the apps in your Applications folders to your server, so the Kindle can draw them.

## What leaves your home, and why

| Goes to | What | When |
|---|---|---|
| [Open-Meteo](https://open-meteo.com) (free weather service) | Your town's coordinates, or the name you type when you search for a place | Every so often, for the forecast |
| Spotify | The track's ID or cover link, to fetch its cover picture | Only while the Music screen is open, once per song |
| The calendar links you add yourself (.ics) | A normal download of that calendar | Every 10 minutes |
| **Your own** Cloudflare account | Everything above | Only if you set up your own cloud server |

The Kindle downloads its updates from your own Desk Buddy server (and, once releases move there, from this GitHub project). The Mac app only opens Cloudflare's or Node.js's website if you click those buttons in the cloud guide.

## The key

Your Kindle, your Mac and your phone share one **key**, a long random password made on your Mac. Every request needs it.

- The Kindle gets it when you click **Allow** on your Mac. Your phone gets it from the QR code on your Mac (don't share that QR).
- With your own cloud server, everything travels encrypted (https). On your home Wi-Fi (This Mac), the Kindle talks to your Mac over plain http, like many home devices do, so use Desk Buddy on a home network you trust.

## Check it yourself

- **It's all open source:** the Mac app is in `menubar/` and `backend/helper.py`, the server in `worker/`, the Kindle app in `kindle/desk/`. Search the code for `https://` to see every outside address it can talk to.
- **Desk Buddy › Permissions** shows each permission with a "Shares:" line saying what it sends.
- Tools like [Little Snitch](https://www.obdev.at/products/littlesnitch/) or [LuLu](https://objective-see.org/products/lulu.html) show every connection an app makes, if you want to watch it.

## Deleting everything

- **Mac:** Desk Buddy › About › **Uninstall…**, then delete `~/Library/Application Support/DeskBuddy/`.
- **Your cloud server:** in Cloudflare, delete the `desk-buddy` Worker and the `desk-buddy` D1 database.
- **Kindle:** over USB, delete the `desk` folder and `documents/Desk Buddy.sh`.
