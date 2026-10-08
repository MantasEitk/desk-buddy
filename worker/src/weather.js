// Weather from Open-Meteo (free, no account), cached for 20 minutes. Location: set with
// PUT /api/weather/location, otherwise Cloudflare's estimate of where the Kindle is.

const REFRESH = 20 * 60;

export async function weather(store, request) {
  let loc = await store.get("weather_loc");
  if (!loc && request.cf?.latitude) {
    loc = { lat: +request.cf.latitude, lon: +request.cf.longitude, name: request.cf.city || "Here", auto: true };
    await store.set("weather_loc", loc);
  }
  if (!loc) return { error: "No location yet." };
  const cache = await store.get("weather_cache");
  const now = Date.now() / 1000;
  if (cache && now - cache.at < REFRESH && cache.lat === loc.lat && cache.lon === loc.lon) return cache;

  const q = new URLSearchParams({
    latitude: loc.lat, longitude: loc.lon, timezone: store.tz, forecast_days: "7", wind_speed_unit: "mph", timeformat: "unixtime",
    current: "temperature_2m,apparent_temperature,weather_code,wind_speed_10m,relative_humidity_2m,is_day",
    hourly: "temperature_2m,precipitation_probability,weather_code,is_day",
    daily: "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunrise,sunset",
  });
  const res = await fetch("https://api.open-meteo.com/v1/forecast?" + q, { headers: { "user-agent": "DeskBuddy/1.0" } });
  if (!res.ok) return cache || { error: `Weather service said ${res.status}` };
  const w = await res.json();
  const c = w.current, h = w.hourly, d = w.daily;
  const r = Math.round;
  const start = h.time.findIndex((t) => t + 3600 > now);
  const out = {
    at: now, lat: loc.lat, lon: loc.lon, place: loc.name,
    current: { temp: r(c.temperature_2m), feels: r(c.apparent_temperature), code: c.weather_code, wind: r(c.wind_speed_10m), humidity: r(c.relative_humidity_2m), day: !!c.is_day },
    // every hour from now to the end of the week (the Kindle shows the next 12, or a chosen day)
    hours: h.time.slice(start).map((t, i) => ({
      t, temp: r(h.temperature_2m[start + i]), rain: h.precipitation_probability[start + i] ?? 0, code: h.weather_code[start + i], day: !!h.is_day[start + i],
    })),
    days: d.time.map((t, i) => ({
      t, code: d.weather_code[i], max: r(d.temperature_2m_max[i]), min: r(d.temperature_2m_min[i]), rain: d.precipitation_probability_max[i] ?? 0,
      sunrise: d.sunrise[i], sunset: d.sunset[i],
    })),
  };
  await store.set("weather_cache", out);
  return out;
}

// Turns a place name into coordinates (Open-Meteo's geocoder), e.g. "Manchester".
export async function findPlace(name) {
  const res = await fetch("https://geocoding-api.open-meteo.com/v1/search?count=1&language=en&name=" + encodeURIComponent(name));
  const r = (await res.json()).results?.[0];
  return r ? { lat: r.latitude, lon: r.longitude, name: r.name } : null;
}

// Up to 5 places for a name, with region and country so similar names can be told apart.
export async function findPlaces(name) {
  if (!name.trim()) return [];
  const res = await fetch("https://geocoding-api.open-meteo.com/v1/search?count=5&language=en&name=" + encodeURIComponent(name.trim()));
  const rs = (await res.json()).results || [];
  return rs.map((r) => ({ lat: r.latitude, lon: r.longitude, name: r.name, area: [r.admin1, r.country].filter(Boolean).join(", ") }));
}
