package metricexpr

import (
	"reflect"
	"testing"
)

func TestNonEmptyExprLines(t *testing.T) {
	t.Parallel()
	got := NonEmptyExprLines("a\n\nb  \n c ")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if NonEmptyExprLines("  \n\t  ") != nil {
		t.Fatal("expected nil")
	}
}

func TestCompositeEnvDimensionIDs(t *testing.T) {
	t.Cleanup(ClearProgramCache)
	t.Parallel()
	ids, err := CompositeEnvDimensionIDs([]string{"a + b", "b + c"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("got %#v want %#v", ids, want)
	}
}
