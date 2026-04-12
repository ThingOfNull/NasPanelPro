package netdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchChartData_mock(t *testing.T) {
	const body = `{
  "labels": ["time", "user", "system"],
  "data": [[1700000000, 1.5, 8.25]]
}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/data" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("chart") != "system.cpu" {
			t.Fatalf("chart param: %q", r.URL.Query().Get("chart"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL}
	cols, err := c.FetchChartData(context.Background(), "system.cpu", DataOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if cols["user"] != 1.5 || cols["system"] != 8.25 {
		t.Fatalf("%v", cols)
	}
}

func TestFetchChartsData_partialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chart := r.URL.Query().Get("chart")
		w.Header().Set("Content-Type", "application/json")
		switch chart {
		case "a":
			_, _ = w.Write([]byte(`{"labels":["time","x"],"data":[[1,2]]}`))
		case "b":
			w.WriteHeader(404)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	snap, err := c.FetchChartsData(context.Background(), []string{"a", "b"}, DataOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if snap["a"]["x"] != 2 {
		t.Fatalf("%v", snap)
	}
	if _, ok := snap["b"]; ok {
		t.Fatal("b should be missing")
	}
}

func TestDataSnapshot_Lookup(t *testing.T) {
	s := DataSnapshot{"c": {"d": 3.14}}
	v, ok := s.Lookup("c", "d")
	if !ok || v != 3.14 {
		t.Fatal()
	}
}

func TestParseDataLabelsValues_fixtureFile(t *testing.T) {
	raw := `{"labels":["time","guest_nice","user"],"data":[[1775989683,0,1.679261]]}`
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	cols, err := ParseDataLabelsValues(m)
	if err != nil {
		t.Fatal(err)
	}
	if cols["user"] != 1.679261 {
		t.Fatalf("%v", cols)
	}
}
