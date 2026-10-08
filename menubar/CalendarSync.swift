// Sends every event from the Mac's Calendar app (all accounts) to the Desk Buddy
// backend: straight away when something changes, or on request. An hourly look keeps the
// date range moving; it only sends when the events differ from what was sent last.

import EventKit
import Foundation

final class CalendarSync {
    let store = EKEventStore()
    var onStatus: (String) -> Void = { _ in }
    var enabled = true { didSet { if enabled { syncSoon() } } }
    private var timer: Timer?
    private var pending: DispatchWorkItem?
    private var sent: Data? // the last body the server accepted (with its address)
    var server = "", key = "" // set by the app (Cloud or this Mac's local server)

    init(server: String, key: String) {
        self.server = server
        self.key = key
    }

    func start() {
        store.requestFullAccessToEvents { [weak self] granted, _ in
            DispatchQueue.main.async {
                guard let self else { return }
                guard granted else {
                    self.onStatus("Calendar: no access · System Settings → Privacy & Security → Calendars")
                    return
                }
                NotificationCenter.default.addObserver(forName: .EKEventStoreChanged, object: self.store, queue: .main) { [weak self] _ in
                    self?.syncSoon() // an event was added, moved or deleted
                }
                self.timer = Timer.scheduledTimer(withTimeInterval: 60 * 60, repeats: true) { [weak self] _ in self?.sync() }
                self.sync()
            }
        }
    }

    // Changes often come in bursts (a calendar account refreshing): wait a few seconds.
    func syncSoon() {
        pending?.cancel()
        let job = DispatchWorkItem { [weak self] in self?.sync() }
        pending = job
        DispatchQueue.main.asyncAfter(deadline: .now() + 3, execute: job)
    }

    func sync(force: Bool = false) {
        guard enabled else { return }
        guard EKEventStore.authorizationStatus(for: .event) == .fullAccess else {
            onStatus("Calendar: no access · System Settings → Privacy & Security → Calendars")
            return
        }
        guard !server.isEmpty, let url = URL(string: server + "/api/calendar/push") else {
            onStatus("Calendar: no backend address in config.json")
            return
        }
        let now = Date()
        let from = now.addingTimeInterval(-40 * 86400), to = now.addingTimeInterval(100 * 86400)
        let events = store.events(matching: store.predicateForEvents(withStart: from, end: to, calendars: nil))
        var cal = Calendar.current
        let utc = TimeZone(identifier: "UTC")!
        let list: [[String: Any]] = events.map { e in
            var start = e.startDate.timeIntervalSince1970, end = e.endDate.timeIntervalSince1970
            if e.isAllDay { // the Kindle expects all-day events as UTC midnights, end exclusive
                cal.timeZone = .current
                let first = cal.dateComponents([.year, .month, .day], from: e.startDate)
                let last = cal.dateComponents([.year, .month, .day], from: e.endDate.addingTimeInterval(-1))
                var u = Calendar(identifier: .gregorian)
                u.timeZone = utc
                start = u.date(from: first)!.timeIntervalSince1970
                end = u.date(byAdding: .day, value: 1, to: u.date(from: last)!)!.timeIntervalSince1970
            }
            return [
                "title": e.title ?? "", "start": start, "end": end, "all_day": e.isAllDay,
                "location": e.location ?? "", "description": e.notes ?? "", "cal": e.calendar?.title ?? "",
                "url": e.url?.absoluteString ?? "",
            ]
        }
        var req = URLRequest(url: url, timeoutInterval: 20)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue(key, forHTTPHeaderField: "X-Key")
        req.setValue("DeskBuddy-helper/1.0", forHTTPHeaderField: "User-Agent")
        let body = try? JSONSerialization.data(withJSONObject: ["source": "mac", "events": list], options: .sortedKeys)
        let stamp = (server.data(using: .utf8) ?? Data()) + (body ?? Data())
        if !force && stamp == sent { return } // nothing new for the server
        req.httpBody = body
        URLSession.shared.dataTask(with: req) { [weak self] _, resp, err in
            let ok = (resp as? HTTPURLResponse)?.statusCode == 200
            if ok { DispatchQueue.main.async { self?.sent = stamp } }
            let time = DateFormatter.localizedString(from: Date(), dateStyle: .none, timeStyle: .short)
            DispatchQueue.main.async {
                self?.onStatus(ok ? "Calendar: \(list.count) events synced at \(time)"
                                  : "Calendar: couldn't sync (\(err?.localizedDescription ?? "server error")) · will retry")
            }
        }.resume()
    }
}
