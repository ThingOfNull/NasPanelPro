package metricexpr

import (
	"math"
	"testing"
)

func TestEvalScalar_basic(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	v, err := EvalScalar(`used / (used + free) * 100`, map[string]float64{
		"used": 30,
		"free": 70,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v-30) > 1e-9 {
		t.Fatalf("got %v want 30", v)
	}
}

func TestEvalScalar_divByZero(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	v, err := EvalScalar(`used / free`, map[string]float64{
		"used": 1,
		"free": 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(v) {
		t.Fatalf("got %v want NaN", v)
	}
}

func TestEvalSeries_aligned(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	pts, err := EvalSeries(`a + b`, map[string][]float64{
		"a": {1, 2, 3},
		"b": {10, 20, 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 3 || pts[0] != 11 || pts[1] != 22 || pts[2] != 33 {
		t.Fatalf("got %v", pts)
	}
}

func TestEvalSeries_minLength(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	pts, err := EvalSeries(`a + b`, map[string][]float64{
		"a": {1, 2, 3},
		"b": {10, 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 2 {
		t.Fatalf("len got %d want 2", len(pts))
	}
}

func TestValidate_empty(t *testing.T) {
	if err := Validate(""); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_invalid(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	if err := Validate("1 +"); err == nil {
		t.Fatal("expected error")
	}
}
