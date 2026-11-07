package geo

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const closeEpsilon = 1e-7

// ValidateLatLng checks a WGS84 coordinate.
func ValidateLatLng(lat, lng float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if math.IsNaN(lng) || math.IsInf(lng, 0) || lng < -180 || lng > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

// ValidateRing checks a closed [latitude, longitude] polygon and returns a
// copy whose final point exactly matches the first.
func ValidateRing(in [][]float64) ([][]float64, error) {
	if len(in) < 4 {
		return nil, errors.New("coordinates must contain at least 4 points, including the closing point")
	}
	if len(in) > 1000 {
		return nil, errors.New("coordinates exceed the 1000 point limit")
	}
	out := make([][]float64, len(in))
	for i, point := range in {
		if len(point) != 2 {
			return nil, fmt.Errorf("coordinate %d must be a [latitude, longitude] pair", i)
		}
		lat, lng := point[0], point[1]
		if err := ValidateLatLng(lat, lng); err != nil {
			return nil, fmt.Errorf("coordinate %d: %w", i, err)
		}
		out[i] = []float64{lat, lng}
	}
	if !nearly(out[0], out[len(out)-1]) {
		return nil, errors.New("first and last coordinates must be identical to close the polygon")
	}
	out[len(out)-1][0] = out[0][0]
	out[len(out)-1][1] = out[0][1]
	if ringArea(out) < 1e-12 {
		return nil, errors.New("polygon area must be greater than zero")
	}
	return out, nil
}

// WKT renders a closed [latitude, longitude] ring as a PostGIS polygon.
// PostGIS expects longitude first.
func WKT(coords [][]float64) string {
	var b strings.Builder
	b.WriteString("POLYGON((")
	for i, point := range coords {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(point[1], 'f', 8, 64))
		b.WriteByte(' ')
		b.WriteString(strconv.FormatFloat(point[0], 'f', 8, 64))
	}
	b.WriteString("))")
	return b.String()
}

// PointInPolygon reports whether a [latitude, longitude] point is inside the
// ring using the even-odd rule. The ring may be closed or open.
//
// Request handling uses PostGIS ST_Covers so the spatial index does the work.
// This function is the reference implementation covered by unit tests.
func PointInPolygon(lat, lng float64, ring [][]float64) bool {
	n := len(ring)
	if n < 3 {
		return false
	}
	if nearly(ring[0], ring[n-1]) {
		n--
	}
	if n < 3 {
		return false
	}
	inside := false
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		latI, lngI := ring[i][0], ring[i][1]
		latJ, lngJ := ring[j][0], ring[j][1]
		// The latitude test is false when the edge is horizontal, so the
		// division below is not evaluated for a zero denominator.
		if (latI > lat) != (latJ > lat) &&
			lng < (lngJ-lngI)*(lat-latI)/(latJ-latI)+lngI {
			inside = !inside
		}
	}
	return inside
}

func ringArea(coords [][]float64) float64 {
	n := len(coords)
	if n > 1 && nearly(coords[0], coords[n-1]) {
		n--
	}
	var sum float64
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		sum += coords[i][1]*coords[j][0] - coords[j][1]*coords[i][0]
	}
	return math.Abs(sum) / 2
}

func nearly(a, b []float64) bool {
	return math.Abs(a[0]-b[0]) <= closeEpsilon && math.Abs(a[1]-b[1]) <= closeEpsilon
}
