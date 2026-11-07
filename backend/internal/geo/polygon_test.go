package geo

import (
	"strings"
	"testing"
)

func TestValidateRing(t *testing.T) {
	ring, err := ValidateRing([][]float64{{0, 0}, {0, 1}, {1, 1}, {0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ring) != 4 || ring[0][0] != ring[3][0] || ring[0][1] != ring[3][1] {
		t.Fatalf("ring not closed: %#v", ring)
	}

	cases := [][][]float64{
		{{0, 0}, {1, 1}, {0, 0}},
		{{0, 0}, {0, 1}, {1, 1}, {1, 0}},
		{{0, 0}, {0, 0}, {0, 0}, {0, 0}},
		{{91, 0}, {0, 1}, {1, 1}, {91, 0}},
		{{0, 0}, {0}},
	}
	for _, coords := range cases {
		if _, err := ValidateRing(coords); err == nil {
			t.Fatalf("expected error for %#v", coords)
		}
	}
}

func TestWKTLonLat(t *testing.T) {
	w := WKT([][]float64{{37.25, -122.5}, {37.5, -122.5}, {37.5, -122.25}, {37.25, -122.5}})
	if !strings.HasPrefix(w, "POLYGON((-122.50000000 37.25000000,") {
		t.Fatal(w)
	}
}

func TestPointInPolygon(t *testing.T) {
	square := [][]float64{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	if !PointInPolygon(0.5, 0.5, square) {
		t.Fatal("expected center inside")
	}
	if PointInPolygon(2, 2, square) {
		t.Fatal("expected outside")
	}

	delivery := [][]float64{
		{37.7749, -122.4194},
		{37.7849, -122.4194},
		{37.7849, -122.4094},
		{37.7749, -122.4094},
		{37.7749, -122.4194},
	}
	if !PointInPolygon(37.778, -122.418, delivery) {
		t.Fatal("expected delivery point inside")
	}
	if PointInPolygon(37.77, -122.4144, delivery) {
		t.Fatal("expected southern point outside")
	}

	restricted := [][]float64{
		{37.7810, -122.4160},
		{37.7840, -122.4160},
		{37.7840, -122.4110},
		{37.7810, -122.4110},
		{37.7810, -122.4160},
	}
	if !PointInPolygon(37.7825, -122.4135, restricted) {
		t.Fatal("expected restricted point inside")
	}
	if PointInPolygon(37.778, -122.418, restricted) {
		t.Fatal("delivery point should be outside the restricted yard")
	}
}
