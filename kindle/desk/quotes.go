package main

// Short lines for the standby clock. Public-domain quotes, plus a few of our own.
// One is picked per hour, so the e-ink isn't redrawn every minute.

import "time"

var quotes = [][2]string{
	{"Well begun is half done.", "Aristotle"},
	{"It does not matter how slowly you go, as long as you do not stop.", "Confucius"},
	{"The journey of a thousand miles begins with one step.", "Lao Tzu"},
	{"We suffer more often in imagination than in reality.", "Seneca"},
	{"While we wait for life, life passes.", "Seneca"},
	{"Begin, be bold, and venture to be wise.", "Horace"},
	{"Our life is frittered away by detail. Simplify, simplify.", "Henry David Thoreau"},
	{"Do the thing and you will have the power.", "Ralph Waldo Emerson"},
	{"The best way out is always through.", "Robert Frost"},
	{"He who has a why to live can bear almost any how.", "Friedrich Nietzsche"},
	{"Rest is not idleness.", "John Lubbock"},
	{"Nothing is particularly hard if you divide it into small jobs.", "Henry Ford"},
	{"Be not afraid of going slowly; be afraid only of standing still.", "Chinese proverb"},
	{"The secret of getting ahead is getting started.", "Mark Twain"},
	{"Very little is needed to make a happy life.", "Marcus Aurelius"},
	{"One thing at a time.", ""},
	{"Finish one thing, then the next.", ""},
	{"A clear desk, a clear head.", ""},
	{"Small steps still move you forward.", ""},
	{"Close the loop before you open a new one.", ""},
	{"Breathe out. Then begin.", ""},
	{"Done today beats perfect someday.", ""},
	{"Make it work, then make it good.", ""},
	{"Slow is smooth, and smooth is fast.", ""},
}

// quoteFor picks the quote for this hour (different each hour, same all hour).
func quoteFor(t time.Time) (text, who string) {
	// int64: the Kindle is 32-bit, where this overflowed and crashed the app.
	h := int64(t.Year())*8784 + int64(t.YearDay())*24 + int64(t.Hour())
	q := quotes[(h*7919)%int64(len(quotes))]
	return q[0], q[1]
}
