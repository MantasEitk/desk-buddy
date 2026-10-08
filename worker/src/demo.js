// Demo mode, for photos and videos: placeholder tasks (stored as profiles "work_demo" /
// "home_demo", so ticking, pinning and focus all work on them), placeholder calendar
// events and Claude status. Real data is left alone. POST /api/demo {on: true|false}.

export const DEMO_TASKS = {
  work: [
    { text: "Finish homepage hero", pinned: 1, kids: [["Headline options", 1], ["Hero illustration", 0], ["Mobile layout", 0]] },
    { text: "Reply to Sam about the brief", tag: "Client" },
    { text: "Prepare invoice for Acme" },
    { text: "Review Figma comments", done: 1 },
    { text: "Write case study intro", notes: "Two paragraphs: the problem, then what changed." },
    { text: "Plan next week" },
  ],
  home: [
    { text: "Meal prep for the week", kids: [["Shopping list", 1], ["Cook rice and veg", 0]] },
    { text: "Gym: leg day" },
    { text: "Call Mum", done: 1 },
    { text: "Book dentist" },
    { text: "Read 20 pages" },
  ],
};

// Placeholder events around today (UK time), so the month grid and "next up" look full.
export function demoEvents(nowSec, offset) {
  const day = 86400;
  const midnight = Math.floor((nowSec + offset) / day) * day - offset; // today 00:00 local
  const at = (d, h, m) => midnight + d * day + h * 3600 + m * 60;
  const out = [];
  const ev = (d, h, m, mins, title, location = "", description = "") =>
    out.push({ title, start: at(d, h, m), end: at(d, h, m) + mins * 60, all_day: false, location, description, cal: "Work", profile: "both" });
  for (let d = -35; d <= 95; d++) {
    const wd = new Date((at(d, 12, 0)) * 1000).getUTCDay(); // 0 Sun … 6 Sat
    if (wd === 1 || wd === 3 || wd === 5) ev(d, 9, 30, 15, "Stand-up", "Google Meet");
    if (wd === 2) ev(d, 14, 0, 45, "Client call · Acme", "Zoom", "Review the homepage draft and agree next steps.");
    if (wd === 4) ev(d, 11, 0, 60, "Design review", "Studio", "Walk through the case study and the new illustrations.");
    if (wd === 1) ev(d, 16, 0, 30, "Plan the week");
    if (wd === 3) ev(d, 18, 30, 60, "Gym class", "Pure Gym");
  }
  // today: something on now-ish and something next, whatever the time
  const hourNow = Math.floor(((nowSec + offset) % day) / 3600);
  ev(0, Math.min(hourNow + 1, 22), 0, 30, "Coffee with Sam", "The corner café");
  ev(0, Math.min(hourNow + 3, 23), 0, 45, "Portfolio review", "Google Meet", "Bring the new illustrations.");
  out.push({ title: "Mum's birthday", start: midnight + offset + 4 * day, end: midnight + offset + 5 * day, all_day: true, location: "", description: "", cal: "Home", profile: "both" });
  out.push({ title: "Launch day", start: midnight + offset + 11 * day, end: midnight + offset + 12 * day, all_day: true, location: "", description: "", cal: "Work", profile: "both" });
  return out.sort((a, b) => a.start - b.start);
}
