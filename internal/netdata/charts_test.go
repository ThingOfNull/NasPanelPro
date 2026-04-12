package netdata

import (
	"strings"
	"testing"
)

func TestDecodeChartsResponse_wrappedCharts(t *testing.T) {
	const j = `{
  "hostname": "x",
  "charts": {
    "system.cpu": {
      "id": "system.cpu",
      "title": "CPU",
      "dimensions": {
        "user": {"name": "user"},
        "system": {"name": "system", "multiplier": 1}
      }
    }
  }
}`
	charts, err := DecodeChartsResponse(strings.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := charts["system.cpu"]
	if !ok {
		t.Fatal("missing chart")
	}
	ids := DimensionIDs(&c)
	if len(ids) != 2 {
		t.Fatalf("dims: %v", ids)
	}
	if DimensionStringField(c.Dimensions["user"], "name") != "user" {
		t.Fatal("user name")
	}
}

func TestDecodeChartsResponse_flatMap(t *testing.T) {
	const j = `{
  "hostname": "h",
  "system.cpu": {
    "id": "system.cpu",
    "dimensions": {"idle": {"name": "idle"}}
  }
}`
	charts, err := DecodeChartsResponse(strings.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := charts["system.cpu"]
	if !ok {
		t.Fatal("missing")
	}
	if len(c.Dimensions) != 1 {
		t.Fatal(c.Dimensions)
	}
}

func TestParseDataLabelsValues(t *testing.T) {
	payload := map[string]interface{}{
		"labels": []interface{}{"time", "user", "system"},
		"data": []interface{}{
			[]interface{}{1.0, 10.5, 20.0},
		},
	}
	cols, err := ParseDataLabelsValues(payload)
	if err != nil {
		t.Fatal(err)
	}
	if cols["user"] != 10.5 || cols["system"] != 20.0 {
		t.Fatalf("%v", cols)
	}
}
