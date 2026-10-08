package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// Joining without a code (DB_TEST_SERVER / DB_TEST_KEY, a throwaway local server): the PC asks,
// "the Mac" clicks Allow (through the API here), and the PC ends up connected with the key.
func TestJoinAllow(t *testing.T) {
	server, key := os.Getenv("DB_TEST_SERVER"), os.Getenv("DB_TEST_KEY")
	if server == "" {
		t.Skip("set DB_TEST_SERVER and DB_TEST_KEY")
	}
	os.Setenv("APPDATA", t.TempDir())
	saveConfig(Config{})
	go join(found{Name: "Test Mac", Addr: server})
	var j *joining
	for i := 0; i < 50; i++ {
		if j, _, _ = joinState(); j != nil && j.State == "waiting" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if j == nil || j.Code == "" {
		t.Fatal("didn't get to waiting for Allow", j)
	}
	// the Mac sees the request with the same code and the PC's name, and clicks Allow
	req, _ := http.NewRequest("GET", server+"/api/pair/pending", nil)
	req.Header.Set("X-Key", key)
	resp, _ := http.DefaultClient.Do(req)
	var pend []struct{ ID, Code, Name string }
	json.NewDecoder(resp.Body).Decode(&pend)
	if len(pend) != 1 || pend[0].Code != j.Code {
		t.Fatal("the Mac doesn't see the request", pend)
	}
	allow, _ := http.NewRequest("POST", server+"/api/pair/"+pend[0].ID, bytes.NewReader([]byte(`{"allow":true}`)))
	allow.Header.Set("X-Key", key)
	allow.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(allow)
	for i := 0; i < 50 && !connected(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if !connected() || current().Key != key || current().Mode != "remote" {
		t.Fatal("not connected after Allow")
	}
	t.Log("asked as", pend[0].Name, "with code", j.Code, "→ Allow → connected")
}
