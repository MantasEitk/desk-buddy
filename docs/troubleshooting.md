# Troubleshooting

## The Kindle can't find my Mac ("No computer found")

- Desk Buddy is open on the Mac (Desk Buddy in the menu bar) and set to **This Mac** (window › Server).
- The Kindle and the Mac are on the **same Wi-Fi**. Guest networks and "client isolation" on some routers stop devices seeing each other.
- macOS asked whether Desk Buddy may accept incoming connections: it must be **Allowed**. Check System Settings › Network › Firewall › Options.
- Then tap **Find it** again on the Kindle (or tap "No computer found" to look again).

## The Kindle says it's offline (a crossed-out cloud at the top)

Tap the icon: the Kindle says what it thinks is wrong.

- **On This Mac**: the Mac is asleep, off the Wi-Fi, or Desk Buddy isn't open. It catches up as soon as the Mac is back; changes you made on the Kindle meanwhile are kept and sent then.
- **Wi-Fi**: see below.

## The Kindle won't join the Wi-Fi, or keeps dropping it

- The Paperwhite 2 needs **2.4 GHz** Wi-Fi with **WPA2**. If your router went 5 GHz-only or WPA3-only (some do after an update), turn on a 2.4 GHz / compatibility option.
- Move it closer to the router to test. A weak signal at the desk is the most common cause.
- **Restart the router.** It has fixed this more than once.
- On the Kindle: airplane mode on and off; or Settings › Wi-Fi Networks › forget the network and join again.
- Restart the Kindle: hold the power button about 40 seconds.

## Updates say "Couldn't check just now"

The Kindle can't reach its server at that moment (see the two sections above). Once it's connected, Settings › Updates › **Check for updates** works.

## Desk Buddy isn't in the Kindle's library

The `Desk Buddy.sh` file must be in the Kindle's `documents` folder, and the jailbreak's hotfix must be installed (it's what lists `.sh` files as books). Give the library a minute after copying.

## The screen is upside down

Change `rotation` in `desk/config.json` on the Kindle from `90` to `270` (or back).

## The screen looks ghosty or flashes too much

Settings › Screen › **Screen refresh**: Gentle, Normal or Clean. The refresh button at the top always cleans the screen.

## The Kindle froze

Hold the power button about 15 seconds until it restarts, then open Desk Buddy from the library.

## No music on the Kindle

- Desk Buddy may control Spotify / Music: window › **Permissions**.
- Spotify sometimes reports nothing playing: play / pause once.
- Music playing on another device (Spotify Connect) doesn't show.

## The Mac asks for permissions again

The app isn't signed with an Apple Developer ID yet, so macOS sees each new version as a new app. Allow them again on the Permissions page.

## The phone page asks for a key

Scan the QR code again (Mac app › Server › Phone page). It changes when you switch between This Mac and the cloud.

## Still stuck?

[Open an issue](../../../issues) with what you see. The Kindle's log is at `desk/desk.log` on the Kindle drive.
