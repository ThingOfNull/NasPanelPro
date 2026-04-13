package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDefault(t *testing.T) {
	c := DefaultLayout()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestChartsUsed(t *testing.T) {
	c := DefaultLayout()
	u := c.ChartsUsed()
	if len(u) < 1 {
		t.Fatal(u)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "layout.json")
	c := DefaultLayout()
	if err := SaveFile(p, c); err != nil {
		t.Fatal(err)
	}
	c2, err := LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if c2.ScreenWidth != c.ScreenWidth || len(c2.Scenes) != len(c.Scenes) {
		t.Fatalf("%+v", c2)
	}
}

func TestLoadFile_missing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("%v", err)
	}
}

func TestValidate_sceneDurationNegative(t *testing.T) {
	c := DefaultLayout()
	c.Scenes[0].Duration = -1
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for negative scene duration")
	}
}

func TestValidate_valueExprOk(t *testing.T) {
	c := DefaultLayout()
	w := &c.Scenes[0].Widgets[1]
	w.Dimensions = nil
	w.CompositeDimsExpr = true
	w.ValueExpr = "user + system"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_valueExprInvalid(t *testing.T) {
	c := DefaultLayout()
	w := &c.Scenes[0].Widgets[1]
	w.Dimensions = nil
	w.CompositeDimsExpr = true
	w.ValueExpr = "1 +"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for invalid value_expr")
	}
}
