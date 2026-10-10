// Putting Desk Buddy on a Kindle over USB, from the app: plug in a jailbroken Kindle, click
// Put Desk Buddy on it; the app copies the desk folder and the library item, already set up
// for this computer (server + key, so the Kindle connects without Find it / Allow), checks
// every file really landed (USB with Kindles can drop files on newer macOS), and ejects it.
// The manual way: Save the files… writes the same thing as a zip with the steps inside.
// The Kindle files come from the bundle: Resources/public/kindle/desk-<sha>.bin and run-<sha>.sh
// (the current release) and Resources/kindle-setup/launcher.sh (install.sh puts them there).

import AppKit
import CryptoKit

final class KindleSetup {
    let model: HelperModel
    let resources: URL
    let kindleConfig: () -> [String: Any] // {server, key}: where this Kindle should connect
    private var volume: URL?

    init(model: HelperModel, resources: URL, kindleConfig: @escaping () -> [String: Any]) {
        self.model = model
        self.resources = resources
        self.kindleConfig = kindleConfig
        let nc = NSWorkspace.shared.notificationCenter
        nc.addObserver(forName: NSWorkspace.didMountNotification, object: nil, queue: .main) { [weak self] _ in self?.scan() }
        nc.addObserver(forName: NSWorkspace.didUnmountNotification, object: nil, queue: .main) { [weak self] _ in self?.scan() }
        scan()
    }

    // A Kindle drive: a volume with the Kindle's own documents and system folders.
    static func isKindle(_ v: URL) -> Bool {
        let fm = FileManager.default
        var dir: ObjCBool = false
        return fm.fileExists(atPath: v.appendingPathComponent("documents").path, isDirectory: &dir) && dir.boolValue
            && fm.fileExists(atPath: v.appendingPathComponent("system").path)
    }

    func scan() {
        let vols = FileManager.default.mountedVolumeURLs(includingResourceValuesForKeys: nil, options: [.skipHiddenVolumes]) ?? []
        volume = vols.first(where: KindleSetup.isKindle)
        if ["copying", "checking"].contains(model.kSetupState) { return }
        if let v = volume {
            if model.kSetupState != "done" || model.kSetupDrive != v.lastPathComponent {
                model.kSetupDrive = v.lastPathComponent
                model.kSetupHas = FileManager.default.fileExists(atPath: v.appendingPathComponent("desk/desk").path)
                model.kSetupState = "found"
                model.kSetupNote = ""
            }
        } else if model.kSetupState != "done" {
            model.kSetupState = "waiting"
            model.kSetupNote = ""
        }
    }

    // the files that go onto the Kindle: [path on the Kindle: contents]
    func files() throws -> [(String, Data)] {
        let pub = resources.appendingPathComponent("public/kindle")
        let names = (try? FileManager.default.contentsOfDirectory(atPath: pub.path)) ?? []
        guard let bin = names.first(where: { $0.hasPrefix("desk-") && $0.hasSuffix(".bin") }),
              let run = names.first(where: { $0.hasPrefix("run-") && $0.hasSuffix(".sh") }) else {
            throw NSError(domain: "setup", code: 1, userInfo: [NSLocalizedDescriptionKey: "This copy of Desk Buddy doesn't have the Kindle files in it."])
        }
        var cfg: [String: Any] = ["server": "", "key": "", "rotation": 90, "full_refresh_every": 40]
        // a Kindle that had Desk Buddy before keeps which way it stands
        if let v = volume, let d = FileManager.default.contents(atPath: v.appendingPathComponent("desk/config.json").path),
           let old = (try? JSONSerialization.jsonObject(with: d)) as? [String: Any] {
            for k in ["rotation", "full_refresh_every"] { if let x = old[k] { cfg[k] = x } }
        }
        for (k, v) in kindleConfig() { cfg[k] = v }
        // the library item; on a Kindle that had Desk Buddy before, with its cover line (as the
        // Kindle app writes it, buddy.go ensureLibraryItem), so the book keeps its cover
        var launcher = try String(contentsOf: resources.appendingPathComponent("kindle-setup/launcher.sh"), encoding: .utf8)
        if let v = volume, FileManager.default.fileExists(atPath: v.appendingPathComponent("desk/cover.png").path) {
            launcher = launcher.replacingOccurrences(of: "# DontUseFBInk", with: "# Icon: /mnt/us/desk/cover.png\n# DontUseFBInk")
        }
        return [
            ("desk/desk", try Data(contentsOf: pub.appendingPathComponent(bin))),
            ("desk/run.sh", try Data(contentsOf: pub.appendingPathComponent(run))),
            ("desk/config.json", try JSONSerialization.data(withJSONObject: cfg, options: [.prettyPrinted, .sortedKeys])),
            ("documents/Desk Buddy.sh", Data(launcher.utf8)),
        ]
    }

    // install copies, checks (twice if needed) and ejects
    func install() {
        guard let v = volume, !["copying", "checking"].contains(model.kSetupState) else { return }
        model.kSetupState = "copying"
        model.kSetupNote = ""
        DispatchQueue.global(qos: .userInitiated).async {
            let result: String?
            do {
                let fs = try self.files()
                // Spotlight indexing the Kindle makes its USB connection drop more often
                FileManager.default.createFile(atPath: v.appendingPathComponent(".metadata_never_index").path, contents: Data())
                var bad = fs
                for attempt in 0..<2 where !bad.isEmpty {
                    for (path, data) in bad { try KindleSetup.write(data, to: v.appendingPathComponent(path)) }
                    DispatchQueue.main.async { self.model.kSetupState = "checking" }
                    KindleSetup.removeAppleDouble(v, fs.map { $0.0 })
                    sync()
                    bad = fs.filter { !KindleSetup.landed($0.1, at: v.appendingPathComponent($0.0)) }
                    if attempt == 0 && !bad.isEmpty { DispatchQueue.main.async { self.model.kSetupState = "copying" } }
                }
                result = bad.isEmpty ? nil : "Some files didn't stick on the Kindle (\(bad.map { $0.0 }.joined(separator: ", "))). Unplug it, plug it straight into this Mac (not a hub), wait 20 seconds, and try again."
            } catch {
                result = "Couldn't copy onto the Kindle: \(error.localizedDescription) Unplug it, plug it in again and try again."
            }
            DispatchQueue.main.async {
                if let result {
                    self.model.kSetupState = "failed"
                    self.model.kSetupNote = result
                    return
                }
                self.model.kSetupState = "ejecting"
                let p = Process()
                p.executableURL = URL(fileURLWithPath: "/usr/sbin/diskutil")
                p.arguments = ["eject", v.path]
                p.terminationHandler = { p in
                    DispatchQueue.main.async {
                        self.model.kSetupState = "done"
                        self.model.kSetupNote = p.terminationStatus == 0 ? ""
                            : "Eject the Kindle before unplugging it (right-click it in Finder › Eject)."
                    }
                }
                do { try p.run() } catch {
                    self.model.kSetupState = "done"
                    self.model.kSetupNote = "Eject the Kindle before unplugging it (right-click it in Finder › Eject)."
                }
            }
        }
    }

    static func write(_ data: Data, to url: URL) throws {
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try? FileManager.default.removeItem(at: url)
        guard FileManager.default.createFile(atPath: url.path, contents: nil), let h = FileHandle(forWritingAtPath: url.path) else {
            throw NSError(domain: "setup", code: 2, userInfo: [NSLocalizedDescriptionKey: "Couldn't create \(url.lastPathComponent)."])
        }
        defer { try? h.close() }
        try h.write(contentsOf: data)
        try h.synchronize() // onto the Kindle, not just the Mac's cache
    }

    // macOS leaves "._name" files beside what it writes on the Kindle's drive; the Kindle's
    // library would show "._Desk Buddy.sh" as a broken item, so they go
    static func removeAppleDouble(_ v: URL, _ paths: [String]) {
        var all = Set(paths)
        for p in paths { all.insert((p as NSString).deletingLastPathComponent) }
        all.insert(".metadata_never_index")
        for p in all where !p.isEmpty {
            let u = v.appendingPathComponent(p)
            try? FileManager.default.removeItem(at: u.deletingLastPathComponent().appendingPathComponent("._" + u.lastPathComponent))
        }
    }

    // landed: the file on the Kindle has the same bytes (read past the Mac's cache)
    static func landed(_ data: Data, at url: URL) -> Bool {
        let fd = open(url.path, O_RDONLY)
        guard fd >= 0 else { return false }
        defer { close(fd) }
        _ = fcntl(fd, F_NOCACHE, 1)
        var got = Data(), buf = [UInt8](repeating: 0, count: 1 << 16)
        while true {
            let n = read(fd, &buf, buf.count)
            if n <= 0 { break }
            got.append(buf, count: n)
        }
        return SHA256.hash(data: got) == SHA256.hash(data: data)
    }

    // The manual way: a zip with the same files (already set up for this computer) and the steps.
    func saveFiles() {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "Desk Buddy for Kindle.zip"
        panel.message = "The files for your Kindle, set up to connect to this computer. Keep them to yourself: they include your Desk Buddy key."
        NSApp.activate(ignoringOtherApps: true)
        guard panel.runModal() == .OK, let dest = panel.url else { return }
        do {
            let dir = FileManager.default.temporaryDirectory.appendingPathComponent("DeskBuddyKindle-" + UUID().uuidString)
            let top = dir.appendingPathComponent("Desk Buddy for Kindle")
            for (path, data) in try files() { try KindleSetup.write(data, to: top.appendingPathComponent(path)) }
            try KindleSetup.steps.write(to: top.appendingPathComponent("How to install.txt"), atomically: true, encoding: .utf8)
            try? FileManager.default.removeItem(at: dest)
            let p = Process()
            p.executableURL = URL(fileURLWithPath: "/usr/bin/ditto")
            p.arguments = ["-c", "-k", "--norsrc", "--noextattr", "--keepParent", top.path, dest.path] // no "._" files (the Kindle would list one)
            try p.run()
            p.waitUntilExit()
            try? FileManager.default.removeItem(at: dir)
            if p.terminationStatus == 0 { NSWorkspace.shared.activateFileViewerSelecting([dest]) }
        } catch {
            let a = NSAlert()
            a.messageText = "Couldn't save the Kindle files"
            a.informativeText = error.localizedDescription
            a.runModal()
        }
    }

    static let steps = """
    Putting Desk Buddy on your Kindle by hand

    Your Kindle needs to be jailbroken first, with the hotfix (kindlemodding.org).

    1. Connect the Kindle to your computer with USB. It shows up as a drive called Kindle.
    2. Copy the "desk" folder to the top level of the Kindle drive (next to "documents").
    3. Copy "Desk Buddy.sh" from the "documents" folder here into the Kindle's "documents" folder.
    4. Eject the Kindle before unplugging it.
    5. On the Kindle, open Desk Buddy from the library. It connects to your computer by itself.

    If files go missing after ejecting: plug the Kindle straight into the computer (not a hub),
    wait 20 seconds before copying, and check the files are there after plugging it back in.

    desk/config.json holds your Desk Buddy key: don't share these files.
    """
}
