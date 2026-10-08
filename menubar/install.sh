#!/bin/sh
# Builds "Desk Buddy.app" into ~/Applications and starts it at every login.
# Re-run after changing backend/helper.py or backend/config.json (optional: copy
# config.example.json and fill in a cloud server; without it, Desk Buddy runs on this Mac).
# Uninstall from its menu.
set -e
cd "$(dirname "$0")"
APP="$HOME/Applications/Desk Buddy.app"
LABEL=app.deskbuddy.mac
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"

# Stop a running copy first.
launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
# …and anything it started (older versions left helper.py running when stopped this way).
pkill -f "Desk Buddy.app/Contents/MacOS/DeskBuddy" 2>/dev/null || true # a copy opened by hand
# Wait for them to be gone (up to 5 s), then force: the new copy must not find an old one
# still running (it would hand over to it and quit, leaving an app whose files were replaced).
for _ in 1 2 3 4 5 6 7 8 9 10; do pgrep -f "Desk Buddy.app/Contents/MacOS/DeskBuddy" >/dev/null || break; sleep 0.5; done
pkill -9 -f "Desk Buddy.app/Contents/MacOS/DeskBuddy" 2>/dev/null || true
pkill -f "Desk Buddy.app/Contents/Resources/helper.py" 2>/dev/null || true
pkill -f "Desk Buddy.app/Contents/MacOS/desk-buddy-server" 2>/dev/null || true

rm -rf "$APP/Contents" # replace what's inside, keep the app itself (so it stays in the Dock)
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$HOME/Library/LaunchAgents"
swiftc -O -o "$APP/Contents/MacOS/DeskBuddy" main.swift Backend.swift CalendarSync.swift IconSync.swift Permissions.swift HelperUI.swift
swiftc -O -o "$APP/Contents/MacOS/audio-out" AudioOut.swift  # the Mac's sound outputs (Kindle music screen)
cp ../backend/helper.py AppIcon.icns "$APP/Contents/Resources/"
# your own settings if you made backend/config.json, else the example (starts on "This Mac")
if [ -f ../backend/config.json ]; then cp ../backend/config.json "$APP/Contents/Resources/"; else cp ../backend/config.example.json "$APP/Contents/Resources/config.json"; fi
cp ../brand/menubar.png ../brand/menubar@2x.png "$APP/Contents/Resources/"  # Desk Buddy in the menu bar
cp ../brand/spotify-mark-alpha.png "$APP/Contents/Resources/spotify-mark.png"                    # Permissions: Spotify's icon
cp ../brand/buddy-kindle.png "$APP/Contents/Resources/buddy.png"          # …and in the window
cp ../kindle/desk/fonts/AtkinsonHyperlegible-Regular.ttf ../kindle/desk/fonts/AtkinsonHyperlegible-Bold.ttf "$APP/Contents/Resources/"  # the window's font (as on the Kindle)
# The local server ("This Mac" backend): the same backend code, built into one program,
# plus the Kindle app builds it serves for updates.
(cd ../worker && npm run --silent build:local >/dev/null)
cp "../worker/local/dist/desk-buddy-server-$(uname -m | sed 's/x86_64/x64/')" "$APP/Contents/MacOS/desk-buddy-server"
cp -R ../worker/public "$APP/Contents/Resources/public"
# The cloud setup kit (Server › Your own cloud server): the same server code + setup.sh; the
# app copies it with Resources/public into Application Support and runs setup.sh there.
mkdir -p "$APP/Contents/Resources/cloud-kit"
cp -R ../worker/src ../worker/schema.sql ../worker/setup.sh "$APP/Contents/Resources/cloud-kit/"
node -e 'const p=require("../worker/package.json");delete p.devDependencies.bun;delete p.scripts;console.log(JSON.stringify(p,null,2))' > "$APP/Contents/Resources/cloud-kit/package.json"
cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleIdentifier</key><string>$LABEL</string>
  <key>CFBundleName</key><string>Desk Buddy</string>
  <key>CFBundleExecutable</key><string>DeskBuddy</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>1.0</string>
  <key>NSCalendarsFullAccessUsageDescription</key><string>Desk Buddy shows your calendar events on the Kindle: what's on now, what's next, and a month view.</string>
  <key>NSCalendarsUsageDescription</key><string>Desk Buddy shows your calendar events on the Kindle.</string>
  <key>NSAppleEventsUsageDescription</key><string>Desk Buddy shows what's playing in Spotify or Apple Music on the Kindle, and its buttons play, pause and skip.</string>
  <key>NSBluetoothAlwaysUsageDescription</key><string>Desk Buddy can connect your Bluetooth headphones when you pick them on the Kindle's music screen.</string>
  <key>LSUIElement</key><true/>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
</dict></plist>
EOF
codesign --force --sign - "$APP" >/dev/null 2>&1 || true

cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key><array><string>$APP/Contents/MacOS/DeskBuddy</string></array>
  <key>RunAtLoad</key><true/>
  <key>ProcessType</key><string>Interactive</string>
</dict></plist>
EOF
launchctl bootstrap "gui/$(id -u)" "$PLIST"
echo "Installed: $APP (starts at login)"
