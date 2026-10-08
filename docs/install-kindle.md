# Desk Buddy on the Kindle

Tested on the **Kindle Paperwhite 2** (6th gen, 2013) with firmware 5.12.x. Other models aren't supported yet (see [Compatibility](../README.md#compatibility)).

## 1. Jailbreak the Kindle

Follow [kindlemodding.org](https://kindlemodding.org) for your Kindle and firmware, **including the hotfix**. Desk Buddy uses three things the jailbreak and hotfix set up:

- **FBInk** (draws on the e-ink screen),
- the **root helper** (`/var/local/mkk/su`),
- the **script launcher** that shows `.sh` files as items in your library.

You don't need KUAL.

## 2. Copy Desk Buddy onto the Kindle (once)

1. Download **Desk Buddy for Kindle** (`.zip`) from [Releases](../../../releases) and unzip it. Inside are a `desk` folder and `documents/Desk Buddy.sh`.
2. Connect the Kindle to your Mac with USB. It shows up as a drive called **Kindle**.
3. Copy the **`desk`** folder to the **top level** of the Kindle drive (next to `documents`).
4. Copy **`Desk Buddy.sh`** into the Kindle's **`documents`** folder.
5. **Eject** the Kindle (right-click › Eject) before unplugging.

**USB tips (especially on macOS 26):** Kindles and newer macOS don't always get along: the drive can disconnect and reconnect, or files can look copied but be gone after ejecting. If that happens:
- Make sure Desk Buddy isn't running on the Kindle while it's plugged in.
- Plug the Kindle straight into the Mac (not through a hub) and wait until the drive has stayed connected for ~20 seconds before copying.
- To stop Spotlight indexing the Kindle (which makes it worse), create an empty file named `.metadata_never_index` at the top level of the Kindle drive (in Terminal: `touch /Volumes/Kindle/.metadata_never_index`).
- After ejecting, plug it back in and check `desk` and `documents/Desk Buddy.sh` are really there.

You only do this once. After this, updates come over Wi-Fi.

## 3. Open it and connect

1. On the Kindle, go to the **Library**. **Desk Buddy** appears there as a book (it can take a moment). Tap it.
2. The screen switches to Desk Buddy, sideways. (Stand the Kindle on its long side. If everything is upside down, see [Settings file](#the-settings-file).)
3. A new Kindle doesn't know your Mac yet: tap **Find it**, then pick your Mac.
4. On the Mac, Desk Buddy asks "A Kindle wants to connect": check the code matches the Kindle's, and click **Allow**.

That's it. From now on it connects by itself, and follows your Mac if you move Desk Buddy to the cloud.

Desk Buddy also gives itself a nicer cover in the library after the first start.

## Using it

- **Back to the normal Kindle**: press the **power button**, or **hold the clock for 3 seconds**.
- **The tour** (what everything does): Settings › About › Show the tour.
- **If it ever freezes**: hold the power button for about 15 seconds; the Kindle restarts. Then open Desk Buddy from the library again.
- Keep the Kindle **plugged in** if it's on your desk all day: Desk Buddy keeps the screen awake.

## Updates

New versions arrive over Wi-Fi. When one is ready, an **Update** button appears at the top; tap it to see what's new, then **Install**. It never installs by itself. Settings › Updates lets you check by hand, or hide the button.

## The settings file

`desk/config.json` on the Kindle (you normally never need it):

| | |
|---|---|
| `server` | Where Desk Buddy runs. Filled in when you connect. |
| `key` | Your Desk Buddy key. Filled in when you click Allow. |
| `rotation` | `90` or `270`: which way the Kindle stands. Swap it if the screen is upside down. |
| `full_refresh_every` | How many quiet redraws before a full refresh (Settings › Screen › Screen refresh is easier). |

## Remove it

Connect over USB and delete the `desk` folder and `documents/Desk Buddy.sh` (and `documents/Desk Buddy.sdr` if it's there). Your Kindle goes back to normal.
