package main

import (
	"testing"
	"time"
)

// Every hour of the coming years gives a quote (this used to overflow on the 32-bit Kindle).
func TestQuoteForEveryHour(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for h := 0; h < 24*366*5; h += 7 {
		if text, _ := quoteFor(start.Add(time.Duration(h) * time.Hour)); text == "" {
			t.Fatal("no quote")
		}
	}
}
