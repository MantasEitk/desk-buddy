# Desk Buddy for Mac

The Mac app lives in your menu bar (a small Desk Buddy). It runs Desk Buddy's server, tells the Kindle when you're at your Mac, sends your calendar and music, and opens apps when you tap a Deck tile.

Needs macOS 13 Ventura or newer.

## Install

1. Download **Desk Buddy for Mac** (`.zip`) from [Releases](../../../releases) and open the zip.
2. Drag **Desk Buddy** into your **Applications** folder.
3. Open it. The first time, macOS may say it can't check the app (it isn't signed with an Apple Developer ID yet): **right-click › Open**, then **Open**. (On newer macOS: System Settings › Privacy & Security › **Open Anyway**.)
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

Click Desk Buddy in the menu bar (or open it from Applications):

- **Home**: your task list (the one the Kindle shows). Tick, add, rename, drag to reorder, add subtasks.
- **Kindle**: all the Kindle's settings, from here.
- **This Mac**: pause Desk Buddy, sync the calendar.
- **Server**: where Desk Buddy runs (**This Mac** or your own **Cloud**), the **phone page QR code**, and the guide to your own cloud server.
- **Permissions**, **Performance** (how little it uses), **About**.

**Pause** (top right) stops this Mac from telling the Kindle anything until you resume.

## Update

Download the new version from [Releases](../../../releases) and replace the old one in Applications. Your tasks and settings are kept. (An Update button in the app is coming.)

## Uninstall

**About › Uninstall…** in the window. It stops Desk Buddy, removes it from login and deletes the app. Your data is in `~/Library/Application Support/DeskBuddy` if you want to remove that too.
