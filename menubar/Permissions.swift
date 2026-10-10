// The permissions Desk Buddy uses, for the window's Permissions section: what each is for,
// whether macOS has allowed it, and a button to ask (first time) or open System Settings.

import AppKit
import CoreBluetooth
import EventKit

struct Permission {
    enum State { case allowed, denied, notAsked, unknown(String) }

    let title: String
    let detail: String
    let settingsPane: String // System Settings › Privacy & Security › …
    let state: () -> State
    let ask: (@escaping () -> Void) -> Void // asks macOS (first time); calls back when answered

    func openSettings() {
        NSWorkspace.shared.open(URL(string: "x-apple.systempreferences:com.apple.preference.security?" + settingsPane)!)
    }
}

enum Permissions {
    static let all: [Permission] = [
        Permission(
            title: "Calendar",
            detail: "Shows your Mac's calendar on the Kindle: today's events, the month, meeting heads-ups.",
            settingsPane: "Privacy_Calendars",
            state: {
                switch EKEventStore.authorizationStatus(for: .event) {
                case .fullAccess, .authorized: return .allowed
                case .notDetermined: return .notAsked
                case .writeOnly: return .unknown("Only adding events is allowed: it needs full access")
                default: return .denied
                }
            },
            ask: { done in
                let store = EKEventStore(), back: EKEventStoreRequestAccessCompletionHandler = { _, _ in DispatchQueue.main.async(execute: done) }
                if #available(macOS 14, *) { store.requestFullAccessToEvents(completion: back) } else { store.requestAccess(to: .event, completion: back) } // macOS 13
            }
        ),
        Permission(
            title: "Controlling Spotify",
            detail: "Shows what's playing on the Kindle, and lets its buttons play, pause and skip.",
            settingsPane: "Privacy_Automation",
            state: { automation("com.spotify.client", appName: "Spotify") },
            ask: { done in askAutomation("com.spotify.client", done) }
        ),
        Permission(
            title: "Controlling Apple Music",
            detail: "The same for Apple Music, if you use it instead of Spotify.",
            settingsPane: "Privacy_Automation",
            state: { automation("com.apple.Music", appName: "Music") },
            ask: { done in askAutomation("com.apple.Music", done) }
        ),
        Permission(
            title: "Bluetooth",
            detail: "Connects your headphones when you pick them on the Kindle's music screen.",
            settingsPane: "Privacy_Bluetooth",
            state: {
                switch CBManager.authorization {
                case .allowedAlways: return .allowed
                case .notDetermined: return .notAsked
                default: return .denied
                }
            },
            ask: { done in
                bluetoothAsker = BluetoothAsker(done) // creating a manager makes macOS ask
            }
        ),
    ]

    // Whether this app may send Apple events to an app (it has to be open to tell).
    static func automation(_ bundleID: String, appName: String) -> Permission.State {
        guard !NSRunningApplication.runningApplications(withBundleIdentifier: bundleID).isEmpty else {
            return .unknown("Open \(appName) to check")
        }
        // ask on a separate thread and give up after 3 s (it can hang if the app is busy); while
        // one is still waiting, don't start another (they'd pile up)
        lock.lock()
        if waiting.contains(bundleID) {
            lock.unlock()
            return .unknown("Couldn't tell right now")
        }
        waiting.insert(bundleID)
        lock.unlock()
        var result: OSStatus = 1
        let done = DispatchSemaphore(value: 0)
        DispatchQueue.global().async {
            var target = NSAppleEventDescriptor(bundleIdentifier: bundleID).aeDesc!.pointee
            result = AEDeterminePermissionToAutomateTarget(&target, typeWildCard, typeWildCard, false)
            lock.lock()
            waiting.remove(bundleID)
            lock.unlock()
            done.signal()
        }
        guard done.wait(timeout: .now() + 3) == .success else { return tryHarmlessEvent(bundleID) }
        switch result {
        case noErr: return .allowed
        case OSStatus(errAEEventWouldRequireUserConsent): return .notAsked
        case OSStatus(errAEEventNotPermitted): return .denied
        default: return tryHarmlessEvent(bundleID)
        }
    }

    // macOS didn't say (Spotify is often slow to answer it): ask the app something harmless
    // ("player state") from this app itself, so macOS answers for Desk Buddy (asking
    // through osascript would check osascript's permission instead). An answer means we're
    // allowed; error -1743 means we're not. Gives up after 4 s.
    static let askQueue = DispatchQueue(label: "permission-ask") // NSAppleScript: one at a time

    static func tryHarmlessEvent(_ bundleID: String) -> Permission.State {
        var state: Permission.State = .unknown("Couldn't tell right now")
        let done = DispatchSemaphore(value: 0)
        askQueue.async {
            var err: NSDictionary?
            let script = NSAppleScript(source: "tell application id \"\(bundleID)\" to get player state")
            if script?.executeAndReturnError(&err) != nil {
                state = .allowed
            } else if (err?[NSAppleScript.errorNumber] as? Int) == -1743 {
                state = .denied
            }
            done.signal()
        }
        _ = done.wait(timeout: .now() + 4)
        return state
    }

    static func askAutomation(_ bundleID: String, _ done: @escaping () -> Void) {
        DispatchQueue.global().async { // this waits for the answer, so not on the main thread
            if var target = NSAppleEventDescriptor(bundleIdentifier: bundleID).aeDesc?.pointee {
                _ = AEDeterminePermissionToAutomateTarget(&target, typeWildCard, typeWildCard, true)
            }
            DispatchQueue.main.async(execute: done)
        }
    }

    static var bluetoothAsker: BluetoothAsker?
    private static let lock = NSLock()
    private static var waiting = Set<String>() // automation checks still waiting on macOS
}

final class BluetoothAsker: NSObject, CBCentralManagerDelegate {
    var manager: CBCentralManager?
    let done: () -> Void
    init(_ done: @escaping () -> Void) {
        self.done = done
        super.init()
        manager = CBCentralManager(delegate: self, queue: .main)
    }
    func centralManagerDidUpdateState(_ central: CBCentralManager) { done() }
}

// Keeps the window's Permissions rows (HelperModel.perms) up to date while it's showing.
final class PermissionChecker {
    private let model: HelperModel
    private var known: [Int: Permission.State] = [:]
    // The checks can block (the Automation one waits on macOS and the other app, and can hang),
    // so they run in the background; each row fills in when its answer comes. Never on the
    // main thread: a hang there froze the whole app once.
    private var checking = Set<Int>()

    init(model: HelperModel) {
        self.model = model
        model.perms = Permissions.all.enumerated().map { HelperModel.PermRow(id: $0.offset, title: $0.element.title, detail: $0.element.detail) }
        refresh()
    }

    func refresh() {
        for (i, p) in Permissions.all.enumerated() where !checking.contains(i) {
            checking.insert(i)
            DispatchQueue.global(qos: .utility).async {
                let st = p.state()
                DispatchQueue.main.async { [weak self] in
                    self?.checking.remove(i)
                    self?.show(i, st)
                }
            }
        }
    }

    private func show(_ i: Int, _ state: Permission.State) {
        known[i] = state
        var r = model.perms[i]
        switch state {
        case .allowed: (r.status, r.ok, r.button) = ("Allowed", true, "Open Settings")
        case .denied: (r.status, r.ok, r.button) = ("Not allowed: turn it on in System Settings", false, "Open Settings")
        case .notAsked: (r.status, r.ok, r.button) = ("Not asked yet", false, "Allow…")
        case .unknown(let why): (r.status, r.ok, r.button) = (why, false, "Open Settings")
        }
        model.perms[i] = r
    }

    // The row's button: ask macOS the first time, otherwise open the right System Settings pane.
    func tapped(_ i: Int) {
        let p = Permissions.all[i]
        if case .notAsked? = known[i] {
            p.ask { [weak self] in self?.refresh() }
        } else {
            p.openSettings()
        }
    }
}
