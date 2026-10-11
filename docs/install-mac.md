# Desk Buddy for Mac

The Mac app lives in your menu bar (a small Desk Buddy). It runs Desk Buddy's server, tells the Kindle when you're at your Mac, sends your calendar and music, and opens apps when you tap a Deck tile.

Needs macOS 13 Ventura or newer.

## Before you download

- **About 50 MB** to download (the exact size is on the release page). It's one app for every Mac, Apple Silicon and Intel, with Desk Buddy's server built in for both, which is most of the size. Installed, it takes about 150 MB.
- **It isn't signed by Apple (yet).** Signing needs a paid Apple Developer membership (99 USD a year), which this hobby project doesn't have. So the first time you open it, macOS warns that it can't check who made it, and you allow it once (below). That warning is about the missing paid signature, not about anything the app does: the code is all here to read, and [docs/privacy.md](privacy.md) explains what it touches.
- Because it isn't signed, macOS also treats each new version as a new app and **may ask for its permissions again after an update**.

## Install

1. Download **Desk Buddy for Mac** (`.zip`) from [Releases](../../../releases) and open the zip (double-click it in Downloads).
2. Open **Desk Buddy**. The first time, macOS warns that it can't check the app (see [Before you download](#before-you-download)). Allow it once:
   - **macOS 15 Sequoia and newer:** click **Done** on the warning, then open **System Settings › Privacy & Security**, scroll down to *"Desk Buddy" was blocked…* and click **Open Anyway** (it asks for your password). Open Desk Buddy again and click **Open**.
   - **macOS 13–14:** **right-click** (or Control-click) Desk Buddy › **Open**, then **Open** again.
3. Desk Buddy asks to **move itself to your Applications folder**: click **Move to Applications**. It opens again from there. (No dragging needed. You can also drag it into Applications yourself first.)
4. The **welcome** walks you through it:
   - what Desk Buddy does,
   - the permissions (each one is optional and only adds what it says),
   - connecting your Kindle.
5. macOS asks once whether Desk Buddy may **accept incoming network connections**: click **Allow**. That's how the Kindle and your phone reach it.

Desk Buddy starts by itself when you log in. To stop that, use **About › Start at login** in its window.

## Permissions

| Permission | What it's for | Without it |
|---|---|---|
| **Calendars** | Your events on the Kindle (today, month view, meeting heads-up) | No events from the Mac (calendar links on the phone page still work) |
| **Controlling Spotify / Music** | What's playing, covers, play / pause / skip | No music on the Kindle |
| **Bluetooth** | Connecting your headphones when you pick them on the Kindle | Picking Bluetooth headphones doesn't connect them |

You can change these any time in the window's **Permissions** page, or in System Settings › Privacy & Security.

## The window

<p align="center"><img src="mac-home.png" width="80%" alt="Desk Buddy for Mac: Home, the task list"></p>

Click Desk Buddy in the menu bar (or open it from Applications):

- **Home**: your task list (the one the Kindle shows). Tick, add, rename, drag to reorder, add subtasks.
- **Kindle**: all the Kindle's settings, from here.
- **This Mac**: pause Desk Buddy, sync the calendar, and **AI assistants**: instructions so Claude Code or Codex can manage your tasks ([more](ai-tasks.md)).
- **Server**: where Desk Buddy runs (**This Mac** or your own **Cloud**), the **phone page QR code**, and the guide to your own cloud server.
- **Permissions**, **Performance** (how little it uses), **About**.

**Pause** (top right) stops this Mac from telling the Kindle anything until you resume.

## Update

**About › Updates** shows your version and what's new in it, and looks for a newer one. To update, download the new version from [Releases](../../../releases) and replace the old one in Applications. Your tasks and settings are kept. (Installing updates from inside the app comes once Desk Buddy's releases are public.)

## Uninstall

**About › Uninstall…** in the window. It stops Desk Buddy, removes it from login and deletes the app. Your data is in `~/Library/Application Support/DeskBuddy` if you want to remove that too.
