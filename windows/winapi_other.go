//go:build !windows

package main

// Stand-ins so the rest can be built and tested on a Mac (the real ones are in winapi_windows.go).

func idleSeconds() float64             { return 5 }
func darkMode() (bool, bool)           { return false, true }
func chimeSound()                      {}
func shellOpen(string) error           { return nil }
func onlyOne() bool                    { return true }
func startsWithWindows() bool          { return false }
func setStartWithWindows(bool) error   { return nil }
