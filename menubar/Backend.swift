// Where the Desk Buddy backend runs: Cloud (Cloudflare) or This Mac (a local server).
//
// The choice lives in ~/Library/Application Support/DeskBuddy/backend.json. In "local" mode
// this app runs the bundled desk-buddy-server (the same backend code as on Cloudflare, with
// a SQLite file in that folder) and keeps it running while the app is open. The Kindle and
// the phone page then talk to this Mac over Wi-Fi, so they only update while it's awake.

import AppKit

final class Backend {
    static let folder = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent("Library/Application Support/DeskBuddy")
    static let file = folder.appendingPathComponent("backend.json")
    static let port = 8787

    struct Settings: Codable {
        var mode: String   // "cloud" or "local"
        var cloud: String  // the Cloudflare address
        var key: String
        var oldKey: String? // after a key change: still accepted by this Mac's server, which hands out the new one
    }

    private(set) var settings: Settings
    private var server: Process?
    var serverPID: Int32? { server?.processIdentifier }
    private var restart: Timer?
    var onStatus: ((String) -> Void)?

    // First run: start from the bundled config.json (the cloud address and key). Without a
    // cloud address there (a fresh install for someone new), start on This Mac with a new key:
    // the Kindle pairs with it over the Wi-Fi (see watchPairing), so nothing needs typing.
    init(bundledConfig: String) {
        if let d = try? Data(contentsOf: Backend.file), let s = try? JSONDecoder().decode(Settings.self, from: d) {
            settings = s
        } else {
            var cloud = "", key = ""
            if let d = FileManager.default.contents(atPath: bundledConfig),
               let c = try? JSONSerialization.jsonObject(with: d) as? [String: Any] {
                cloud = c["server"] as? String ?? ""
                key = c["key"] as? String ?? ""
            }
            if cloud.isEmpty || key.isEmpty {
                settings = Settings(mode: "local", cloud: cloud, key: Backend.newKey())
            } else {
                settings = Settings(mode: "cloud", cloud: cloud, key: key)
            }
            save()
        }
    }

    var local: Bool { settings.mode == "local" }

    // The address this Mac's own parts (helper.py, calendar, icons) use.
    var address: String { local ? "http://127.0.0.1:\(Backend.port)" : settings.cloud }

    // The address the Kindle and phone use for this Mac on the Wi-Fi.
    var lanAddress: String {
        "http://\(Backend.wifiIP() ?? (Host.current().localizedName ?? "this-mac") + ".local"):\(Backend.port)"
    }

    private func save() {
        try? FileManager.default.createDirectory(at: Backend.folder, withIntermediateDirectories: true)
        try? JSONEncoder().encode(settings).write(to: Backend.file)
    }

    // MARK: your own cloud server (Server › Your own cloud server)

    static let kitFolder = folder.appendingPathComponent("cloud-setup")

    // Copies the setup kit (server code + setup.sh, and the Kindle builds) to Application Support,
    // with the key in a private file, so setup.sh (or an AI helping) never needs to see it typed.
    func prepareCloudKit() throws -> URL {
        let fm = FileManager.default, res = Bundle.main.resourceURL!, kit = Backend.kitFolder
        try fm.createDirectory(at: kit, withIntermediateDirectories: true)
        for name in ["src", "schema.sql", "setup.sh", "package.json", "public"] { // node_modules and its config stay for re-runs
            let to = kit.appendingPathComponent(name)
            try? fm.removeItem(at: to)
            try fm.copyItem(at: res.appendingPathComponent(name == "public" ? "public" : "cloud-kit/" + name), to: to)
        }
        let key = kit.appendingPathComponent("key")
        try settings.key.write(to: key, atomically: true, encoding: .utf8)
        try fm.setAttributes([.posixPermissions: 0o600], ofItemAtPath: key.path)
        return kit
    }

    // Opens Terminal running setup.sh in the kit (a .command file: no extra permission needed).
    func runCloudSetup() throws {
        let kit = try prepareCloudKit()
        let cmd = kit.appendingPathComponent("Put Desk Buddy online.command")
        try """
        #!/bin/sh
        cd "$(dirname "$0")"
        sh setup.sh
        echo "  You can close this window."
        """.write(to: cmd, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: cmd.path)
        NSWorkspace.shared.open(cmd)
    }

    // Checks an address answers as a Desk Buddy server with this key: nil, or what's wrong.
    func checkCloud(_ url: String, done: @escaping (String?) -> Void) {
        guard let u = URL(string: url.trimmingCharacters(in: .whitespaces).trimmingCharacters(in: CharacterSet(charactersIn: "/")) + "/api/where"),
              u.scheme == "https" else { return done("Paste the https://… address the setup printed.") }
        var r = URLRequest(url: u, timeoutInterval: 20)
        r.setValue(settings.key, forHTTPHeaderField: "X-Key")
        r.setValue("DeskBuddy-helper/1.0", forHTTPHeaderField: "User-Agent")
        URLSession.shared.dataTask(with: r) { _, resp, _ in
            let code = (resp as? HTTPURLResponse)?.statusCode ?? 0
            DispatchQueue.main.async {
                done(code == 200 ? nil : code == 401 ? "That server has a different key. Run the setup from this app again." : "Couldn't reach it. Check the address, or wait a minute and try again.")
            }
        }.resume()
    }

    func setCloud(_ url: String) {
        settings.cloud = url.trimmingCharacters(in: .whitespaces).trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        save()
        try? FileManager.default.removeItem(at: Backend.kitFolder.appendingPathComponent("key")) // done with it
    }

    // MARK: the local server

    private var stopLater: DispatchWorkItem?

    func startServer() {
        stopLater?.cancel() // switched back within the minute: keep it
        guard local, server == nil else { return }
        if !Backend.portFree(Backend.port) { // e.g. another server left running: say so, look again soon
            onStatus?("Port \(Backend.port) is used by another program · retrying")
            restart = Timer.scheduledTimer(withTimeInterval: 15, repeats: false) { [weak self] _ in self?.startServer() }
            return
        }
        let bin = Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/desk-buddy-server")
        let p = Process()
        p.executableURL = bin
        p.arguments = ["--data", Backend.folder.appendingPathComponent("server").path, "--key", settings.key]
            + (settings.oldKey.map { ["--key-old", $0] } ?? []) + (settings.cloud.isEmpty ? [] : ["--cloud", settings.cloud]) + [
                       "--port", String(Backend.port),
                       "--assets", Bundle.main.resourceURL!.appendingPathComponent("public").path]
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        p.terminationHandler = { [weak self] _ in
            DispatchQueue.main.async {
                guard let self, self.server != nil else { return }
                self.server = nil
                self.onStatus?("Local server stopped · restarting…")
                self.restart = Timer.scheduledTimer(withTimeInterval: 5, repeats: false) { [weak self] _ in self?.startServer() }
            }
        }
        do {
            try p.run()
            server = p
            onStatus?("Local server running")
        } catch {
            onStatus?("Couldn't start the local server: \(error.localizedDescription)")
        }
    }

    // True when nothing is listening on port (a test bind on all interfaces, as the server does).
    // SO_REUSEADDR like the server: connections still closing after the last server stopped
    // (TIME_WAIT) don't stop it starting, so they mustn't count as "in use" here either.
    static func portFree(_ port: Int) -> Bool {
        let fd = socket(AF_INET, SOCK_STREAM, 0)
        guard fd >= 0 else { return true }
        defer { close(fd) }
        var yes: Int32 = 1
        setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &yes, socklen_t(MemoryLayout<Int32>.size))
        var addr = sockaddr_in()
        addr.sin_family = sa_family_t(AF_INET)
        addr.sin_port = in_port_t(UInt16(port).bigEndian)
        addr.sin_addr = in_addr(s_addr: INADDR_ANY)
        return withUnsafePointer(to: &addr) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                bind(fd, $0, socklen_t(MemoryLayout<sockaddr_in>.size)) == 0
            }
        }
    }

    func stopServer() {
        restart?.invalidate()
        let p = server
        server = nil
        p?.terminate()
    }

    // MARK: switching

    // Moves to mode; copy brings everything (tasks, settings, calendar, deck) from the old
    // backend to the new one first. done(nil) when finished, or done(error message).
    func switchTo(_ mode: String, copy: Bool, done: @escaping (String?) -> Void) {
        let from = address
        let wasLocal = local
        if mode == "local" {
            if server == nil && !Backend.portFree(Backend.port) {
                return done("Another program on this Mac is already using port \(Backend.port), so Desk Buddy's server can't start. Quit that program (or restart the Mac) and try again.")
            }
            settings.mode = "local"
            startServer()
        }
        let to = mode == "local" ? "http://127.0.0.1:\(Backend.port)" : settings.cloud
        let finish: (String?) -> Void = { [weak self] err in
            DispatchQueue.main.async {
                guard let self else { return }
                if err != nil && mode == "local" && !wasLocal { // couldn't move: stay where we were
                    self.settings.mode = "cloud"
                    self.stopServer()
                } else if err == nil {
                    self.settings.mode = mode
                    // tell the Kindle: a note on the old server says where to go, the new one's is cleared
                    let kindleTo = mode == "local" ? self.lanAddress : self.settings.cloud
                    self.request(to + "/api/kindle/move", body: try? JSONSerialization.data(withJSONObject: ["server": ""])) { _ in }
                    self.request(from + "/api/kindle/move", body: try? JSONSerialization.data(withJSONObject:
                        ["server": kindleTo, "name": Host.current().localizedName ?? "This Mac"])) { _ in }
                    if mode == "cloud" { // keep this Mac's server a minute, so the Kindle can read the note
                        let job = DispatchWorkItem { [weak self] in if self?.local == false { self?.stopServer() } }
                        self.stopLater = job
                        DispatchQueue.main.asyncAfter(deadline: .now() + 60, execute: job)
                    }
                }
                self.save()
                done(err)
            }
        }
        guard copy else { return waitUntilUp(to) { ok in finish(ok ? nil : "The local server didn't start.") } }
        waitUntilUp(to) { [weak self] ok in
            guard let self, ok else { return finish("The local server didn't start.") }
            self.request(from + "/api/export") { data in
                guard let data, (try? JSONSerialization.jsonObject(with: data)) != nil else {
                    return finish("Couldn't read your data from \(wasLocal ? "this Mac" : "the cloud").")
                }
                self.request(to + "/api/import", body: data) { reply in
                    finish(reply == nil ? "Couldn't copy your data to \(mode == "local" ? "this Mac" : "the cloud")." : nil)
                }
            }
        }
    }

    private func waitUntilUp(_ base: String, tries: Int = 20, _ done: @escaping (Bool) -> Void) {
        request(base + "/api/where") { [weak self] data in
            if data != nil { return done(true) }
            guard tries > 0 else { return done(false) }
            DispatchQueue.global().asyncAfter(deadline: .now() + 0.5) { self?.waitUntilUp(base, tries: tries - 1, done) }
        }
    }

    // GET a backend path (e.g. "/api/state") from wherever it runs now.
    func get(_ path: String, _ done: @escaping (Data?) -> Void) { request(address + path, done) }

    // POST JSON to a backend path.
    func post(_ path: String, _ body: [String: Any], _ done: @escaping (Data?) -> Void = { _ in }) {
        request(address + path, body: try? JSONSerialization.data(withJSONObject: body), done)
    }

    // Any method (PATCH, DELETE…) on a backend path, with an optional JSON body.
    func send(_ method: String, _ path: String, _ body: [String: Any]? = nil, _ done: @escaping (Data?) -> Void = { _ in }) {
        request(address + path, body: body.flatMap { try? JSONSerialization.data(withJSONObject: $0) }, method: method, done)
    }

    private func request(_ url: String, body: Data? = nil, method: String? = nil, _ done: @escaping (Data?) -> Void) {
        guard let u = URL(string: url) else { return done(nil) }
        var r = URLRequest(url: u, timeoutInterval: 60)
        r.setValue(settings.key, forHTTPHeaderField: "X-Key")
        r.setValue("DeskBuddy-helper/1.0", forHTTPHeaderField: "User-Agent")
        if let body {
            r.httpMethod = "POST"
            r.httpBody = body
            r.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if let method { r.httpMethod = method }
        URLSession.shared.dataTask(with: r) { data, resp, _ in
            done((resp as? HTTPURLResponse)?.statusCode == 200 ? data : nil)
        }.resume()
    }

    static func newKey() -> String {
        let chars = Array("abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")
        return String((0..<24).map { _ in chars.randomElement()! })
    }

    // MARK: pairing (local mode)

    // A Kindle on the Wi-Fi asking to connect: onPairRequest(id, code, name) asks the user,
    // answerPair sends the answer (the server then hands the Kindle the key).
    var onPairRequest: ((String, String, String) -> Void)?
    private var pairTimer: Timer?
    private var asked = Set<String>()

    func watchPairing() {
        pairTimer?.invalidate()
        pairTimer = Timer.scheduledTimer(withTimeInterval: 2, repeats: true) { [weak self] _ in
            guard let self, self.local, self.server != nil else { return }
            self.request(self.address + "/api/pair/pending") { data in
                guard let data, let list = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]] else { return }
                DispatchQueue.main.async {
                    for p in list {
                        guard let id = p["id"] as? String, let code = p["code"] as? String, !self.asked.contains(id) else { continue }
                        self.asked.insert(id)
                        self.onPairRequest?(id, code, p["name"] as? String ?? "Kindle")
                    }
                }
            }
        }
    }

    func answerPair(_ id: String, allow: Bool) {
        let body = try? JSONSerialization.data(withJSONObject: ["allow": allow])
        request(address + "/api/pair/" + id, body: body) { _ in }
    }

    // This Mac's address on the Wi-Fi (en0), e.g. 192.168.1.23.
    static func wifiIP() -> String? {
        var ifs: UnsafeMutablePointer<ifaddrs>?
        guard getifaddrs(&ifs) == 0, let first = ifs else { return nil }
        defer { freeifaddrs(ifs) }
        var best: String?
        for p in sequence(first: first, next: { $0.pointee.ifa_next }) {
            let a = p.pointee
            guard a.ifa_addr.pointee.sa_family == UInt8(AF_INET) else { continue }
            let name = String(cString: a.ifa_name)
            guard name.hasPrefix("en") else { continue }
            var host = [CChar](repeating: 0, count: Int(NI_MAXHOST))
            getnameinfo(a.ifa_addr, socklen_t(a.ifa_addr.pointee.sa_len), &host, socklen_t(host.count), nil, 0, NI_NUMERICHOST)
            let ip = String(cString: host)
            if name == "en0" { return ip }
            best = best ?? ip
        }
        return best
    }
}
