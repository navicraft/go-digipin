package digipin

import (
	"math"
	"testing"
)

// FuzzEncodeDecodeRoundTrip checks that any in-bounds coordinate encodes to a
// code whose cell contains both the original point and the decoded center, for
// every precision level.
func FuzzEncodeDecodeRoundTrip(f *testing.F) {
	f.Add(12.9716, 77.5946, 10)
	f.Add(2.5, 63.5, 1)
	f.Add(38.5, 99.5, 7)
	f.Add(28.70486848, 77.10376044, 10)

	f.Fuzz(func(t *testing.T, lat, lng float64, level int) {
		if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
			t.Skip()
		}
		if lat < minLat || lat > maxLat || lng < minLng || lng > maxLng {
			t.Skip()
		}
		level = level%10 + 1
		if level < 1 {
			level += 10
		}

		c, err := EncodeLevel(lat, lng, level)
		if err != nil {
			t.Fatalf("EncodeLevel(%v,%v,%d): %v", lat, lng, level, err)
		}
		if c.Level() != level {
			t.Fatalf("level = %d, want %d", c.Level(), level)
		}

		b := c.Bounds()
		const eps = 1e-9
		if lat < b.MinLatitude-eps || lat > b.MaxLatitude+eps ||
			lng < b.MinLongitude-eps || lng > b.MaxLongitude+eps {
			t.Fatalf("point (%v,%v) not inside cell %v for code %q", lat, lng, b, c)
		}
		if !b.Contains(c.Coordinate()) {
			t.Fatalf("center %v not inside cell %v", c.Coordinate(), b)
		}
	})
}

// FuzzDecode checks that Decode never panics on arbitrary input and that any
// accepted code is valid and round-trips through Parse.
func FuzzDecode(f *testing.F) {
	f.Add("39JJ567L7P")
	f.Add("39J J567 L7P")
	f.Add("39J-J56-7L7P")
	f.Add("")
	f.Add("not a code")

	f.Fuzz(func(t *testing.T, s string) {
		coord, err := Decode(s)
		if err != nil {
			return
		}
		if math.IsNaN(coord.Latitude) || math.IsNaN(coord.Longitude) {
			t.Fatalf("decoded NaN from %q", s)
		}
		c, err := Parse(s)
		if err != nil {
			t.Fatalf("Decode accepted %q but Parse rejected it: %v", s, err)
		}
		if !c.Valid() {
			t.Fatalf("Parse(%q) produced invalid code %q", s, c)
		}
	})
}
