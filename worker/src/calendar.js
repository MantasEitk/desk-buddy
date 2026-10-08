// Calendars: private ICS links (Google, Outlook, iCloud...) fetched every 10 minutes and
// expanded into plain events, repeats included, for about a month back and three ahead.

import ICAL from "ical.js";

const REFRESH = 600; // seconds
const BACK = 40 * 86400, AHEAD = 100 * 86400;

// Fetch and expand every calendar. Returns the list of events, oldest first.
export async function loadCalendars(calendars, now) {
  const from = now - BACK, to = now + AHEAD;
  const events = [];
  const errors = [];
  for (const cal of calendars) {
    try {
      const res = await fetch(cal.url.replace(/^webcal:/, "https:"), { headers: { "user-agent": "DeskBuddy/1.0" } });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      for (const e of expand(await res.text(), from, to)) events.push({ ...e, cal: cal.name, profile: cal.profile });
    } catch (e) {
      errors.push(`${cal.name}: ${e.message}`);
    }
  }
  events.sort((a, b) => a.start - b.start);
  return { events, errors };
}

export const stale = (cache, now) => !cache || now - cache.at > REFRESH;

function expand(text, from, to) {
  const root = new ICAL.Component(ICAL.parse(text));
  for (const tz of root.getAllSubcomponents("vtimezone")) ICAL.TimezoneService.register(tz);

  // Group moved/changed single occurrences with the event they belong to.
  const byUid = new Map();
  for (const v of root.getAllSubcomponents("vevent")) {
    const ev = new ICAL.Event(v);
    const g = byUid.get(ev.uid) || { master: null, exceptions: [] };
    if (ev.isRecurrenceException()) g.exceptions.push(ev);
    else g.master = ev;
    byUid.set(ev.uid, g);
  }

  const out = [];
  const push = (ev, start, end) => {
    const s = start.toUnixTime(), e = end ? end.toUnixTime() : s;
    if (e < from || s > to) return;
    if (String(ev.component.getFirstPropertyValue("status") || "").toUpperCase() === "CANCELLED") return;
    out.push({
      title: ev.summary || "(no title)",
      start: s,
      end: Math.max(e, s),
      all_day: start.isDate,
      location: ev.location || "",
      description: tidy(ev.description || ""),
    });
  };

  for (const { master, exceptions } of byUid.values()) {
    if (!master) {
      exceptions.forEach((x) => push(x, x.startDate, x.endDate));
      continue;
    }
    exceptions.forEach((x) => master.relateException(x));
    if (!master.isRecurring()) {
      push(master, master.startDate, master.endDate);
      continue;
    }
    const it = master.iterator();
    for (let n = 0, t; (t = it.next()) && n < 1000; n++) {
      if (t.toUnixTime() > to) break;
      const d = master.getOccurrenceDetails(t);
      push(d.item, d.startDate, d.endDate);
    }
  }
  return out;
}

// Calendar descriptions are often HTML or full of meeting-link boilerplate; keep it readable.
function tidy(s) {
  return s
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, " ").replace(/&amp;/g, "&").replace(/&lt;/g, "<").replace(/&gt;/g, ">")
    .replace(/\n{3,}/g, "\n\n")
    .trim()
    .slice(0, 600);
}
