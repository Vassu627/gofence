package httpapi

import "testing"

func TestParseLimit(t *testing.T) {
	n, err := parseLimit("")
	if err != nil || n != 50 {
		t.Fatal(n, err)
	}
	n, err = parseLimit("10")
	if err != nil || n != 10 {
		t.Fatal(n, err)
	}
	n, err = parseLimit("900")
	if err != nil || n != 500 {
		t.Fatal(n, err)
	}
	if _, err := parseLimit("0"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := parseLimit("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseOffset(t *testing.T) {
	n, err := parseOffset("")
	if err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err := parseOffset("-1"); err == nil {
		t.Fatal("expected error")
	}
}
