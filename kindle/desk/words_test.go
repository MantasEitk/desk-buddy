package main

import (
	"testing"
	"time"
)

func TestTimeInWords(t *testing.T) {
	for in, want := range map[string]string{
		"21:49": "It's ten to ten", "09:20": "It's twenty past nine", "12:00": "It's noon", "00:01": "It's midnight",
		"23:58": "It's midnight", "15:15": "It's quarter past three", "07:30": "It's half past seven", "10:44": "It's quarter to eleven",
		"11:58": "It's noon", "08:03": "It's five past eight", "06:00": "It's six o'clock",
	} {
		tm, _ := time.Parse("15:04", in)
		if got := timeInWords(tm); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}
