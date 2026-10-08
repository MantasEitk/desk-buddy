package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Changes made offline go out in order once the server is back; a task added offline gets
// its real id, and a later tick of it uses that id.
func TestOutboxFlush(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/api/tasks" {
			json.NewEncoder(w).Encode(map[string]int{"id": 42})
			return
		}
		if strings.Contains(r.URL.Path, "/7/") { // a task deleted elsewhere: refused, dropped
			w.WriteHeader(404)
			return
		}
		w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k")
	o := &Outbox{}
	o.add(Target{Act: "toggle", ID: 7})
	add := o.add(Target{Act: "addtask", Arg: "work|Buy milk"})
	o.add(Target{Act: "toggle", ID: add.Temp})
	o.add(Target{Act: "addtask", ID: add.Temp, Arg: "work|Oat milk"}) // a subtask of it

	d := &Desk{Profile: "work"}
	o.applyAll(d)
	if len(d.Tasks) != 2 || !d.Tasks[0].Done || d.Tasks[1].Parent == nil || *d.Tasks[1].Parent != add.Temp {
		t.Fatalf("offline view: %+v", d.Tasks)
	}

	n, ids, err := o.flush(c, o.Items)
	if err != nil || n != 4 || ids[add.Temp] != 42 {
		t.Fatalf("flush: %d %v %v", n, ids, err)
	}
	want := "POST /api/tasks/7/toggle,POST /api/tasks,POST /api/tasks/42/toggle,POST /api/tasks"
	if strings.Join(got, ",") != want {
		t.Fatalf("sent %v", got)
	}

	// offline: nothing is dealt with, and the error says so
	srv.Close()
	n, _, err = o.flush(c, o.Items)
	if n != 0 || err == nil || !netError(err) {
		t.Fatalf("offline flush: %d %v", n, err)
	}
}
