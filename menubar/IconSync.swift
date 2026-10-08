// Sends the real app icons for this Mac's Deck tiles to the backend, as small
// high-contrast greyscale PNGs, so the Kindle can draw them instead of letters.

import AppKit

final class IconSync {
    var server = "", key = "" // set by the app (Cloud or this Mac's local server)
    private var timer: Timer?
    var enabled = true
    private let queue = DispatchQueue(label: "icons", qos: .utility) // never on the main thread: it froze the window
    private var sent = "" // what was uploaded last (tiles + every app and its date): unchanged = nothing to do

    init(server: String, key: String) {
        self.server = server
        self.key = key
    }

    private var watchers: [DispatchSourceFileSystemObject] = []
    private var soon: DispatchWorkItem?

    // Looks again when an app is added, removed or updated in the app folders (plus a
    // 6-hourly look for anything missed); unchanged apps cost nothing (see `sent`).
    func start() {
        sync()
        timer = Timer.scheduledTimer(withTimeInterval: 6 * 3600, repeats: true) { [weak self] _ in self?.sync() }
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        for dir in ["/Applications", "/Applications/Utilities", "\(home)/Applications"] {
            let fd = open(dir, O_EVTONLY)
            guard fd >= 0 else { continue }
            let w = DispatchSource.makeFileSystemObjectSource(fileDescriptor: fd, eventMask: [.write, .rename, .delete], queue: .main)
            w.setEventHandler { [weak self] in self?.syncSoon() }
            w.setCancelHandler { close(fd) }
            w.resume()
            watchers.append(w)
        }
    }

    // installs come in bursts (copying, then unpacking): wait for them to settle
    private func syncSoon() {
        soon?.cancel()
        let job = DispatchWorkItem { [weak self] in self?.sync() }
        soon = job
        DispatchQueue.main.asyncAfter(deadline: .now() + 20, execute: job)
    }

    private func request(_ path: String, body: Any? = nil) -> URLRequest? {
        guard let url = URL(string: server + path) else { return nil }
        var r = URLRequest(url: url, timeoutInterval: 20)
        r.setValue(key, forHTTPHeaderField: "X-Key")
        r.setValue("DeskBuddy-helper/1.0", forHTTPHeaderField: "User-Agent")
        if let body {
            r.httpMethod = "POST"
            r.setValue("application/json", forHTTPHeaderField: "Content-Type")
            r.httpBody = try? JSONSerialization.data(withJSONObject: body)
        }
        return r
    }

    // Reads the Work tiles (they run on the Mac), finds each app, uploads its icon.
    func sync() {
        guard enabled, let get = request("/api/deck") else { return }
        URLSession.shared.dataTask(with: get) { [weak self] data, _, _ in
            guard let self, let data,
                  let deck = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let tiles = deck["work"] as? [[String: Any]] else { return }
            self.queue.async {
                let all = Self.allApps()
                let modified = { (path: String) in
                    ((try? FileManager.default.attributesOfItem(atPath: path)[.modificationDate]) as? Date)?.timeIntervalSince1970 ?? 0
                }
                let sig = self.server + "|" + tiles.map { "\($0["label"] ?? "")=\($0["app"] ?? "")" }.joined(separator: ",")
                    + "|" + all.map { "\($0.1)@\(modified($0.1))" }.joined(separator: ",")
                guard sig != self.sent else { return }
                var icons: [String: String] = [:]
                // Favourite tiles, keyed by their label ("Chrome" for Google Chrome)…
                for t in tiles {
                    guard let label = t["label"] as? String, let app = t["app"] as? String,
                          let path = Self.find(app), let png = Self.png(NSWorkspace.shared.icon(forFile: path)) else { continue }
                    icons[label] = png.base64EncodedString()
                }
                // …and every app on the Mac, for the Kindle's "All apps" view.
                for (name, path) in all where icons[name] == nil {
                    if let png = Self.png(NSWorkspace.shared.icon(forFile: path)) { icons[name] = png.base64EncodedString() }
                }
                let body: [String: Any] = ["machine": "mac", "icons": icons, "apps": all.map { $0.0 }]
                if let post = self.request("/api/deck/icons", body: body) {
                    URLSession.shared.dataTask(with: post) { _, resp, _ in
                        if (resp as? HTTPURLResponse)?.statusCode == 200 { self.queue.async { self.sent = sig } }
                    }.resume()
                }
            }
        }.resume()
    }

    // Every app the user would open: top level of the usual app folders, by name.
    static func allApps() -> [(String, String)] {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        var seen = Set<String>(), out: [(String, String)] = []
        for dir in ["/Applications", "/Applications/Utilities", "/System/Applications", "/System/Applications/Utilities", "\(home)/Applications"] {
            for f in (try? FileManager.default.contentsOfDirectory(atPath: dir)) ?? [] where f.hasSuffix(".app") {
                let name = String(f.dropLast(4))
                if seen.insert(name).inserted { out.append((name, "\(dir)/\(f)")) }
            }
        }
        out.append(("Finder", "/System/Library/CoreServices/Finder.app"))
        return out.sorted { $0.0.localizedCaseInsensitiveCompare($1.0) == .orderedAscending }
    }

    static func find(_ name: String) -> String? {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        for dir in ["/Applications", "/System/Applications", "/System/Applications/Utilities",
                    "\(home)/Applications", "/System/Library/CoreServices", "/Applications/Utilities"] {
            let p = "\(dir)/\(name).app"
            if FileManager.default.fileExists(atPath: p) { return p }
        }
        return nil
    }

    // 96×96 greyscale on white, darkest tone stretched to black so it reads on e-ink.
    static func png(_ icon: NSImage) -> Data? {
        let n = 96
        guard let rgb = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: n, pixelsHigh: n, bitsPerSample: 8,
                                         samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
                                         colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0) else { return nil }
        NSGraphicsContext.saveGraphicsState()
        NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rgb)
        NSColor.white.setFill()
        NSRect(x: 0, y: 0, width: n, height: n).fill()
        icon.draw(in: NSRect(x: 0, y: 0, width: n, height: n), from: .zero, operation: .sourceOver, fraction: 1)
        NSGraphicsContext.restoreGraphicsState()

        guard let src = rgb.bitmapData else { return nil }
        let row = rgb.bytesPerRow
        var lum = [Double](repeating: 255, count: n * n)
        var darkest = 255.0
        for y in 0..<n {
            for x in 0..<n { // drawn over white, so the colour can be read as is
                let o = y * row + x * 4
                let l = 0.299 * Double(src[o]) + 0.587 * Double(src[o + 1]) + 0.114 * Double(src[o + 2])
                lum[y * n + x] = l
                darkest = min(darkest, l)
            }
        }
        guard let gray = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: n, pixelsHigh: n, bitsPerSample: 8,
                                          samplesPerPixel: 1, hasAlpha: false, isPlanar: false,
                                          colorSpaceName: .deviceWhite, bytesPerRow: n, bitsPerPixel: 8),
              let px = gray.bitmapData else { return nil }
        let span = max(1, 255 - darkest)
        for i in 0..<(n * n) {
            px[i] = UInt8(max(0, min(255, (lum[i] - darkest) / span * 255)))
        }
        return gray.representation(using: .png, properties: [:])
    }
}
