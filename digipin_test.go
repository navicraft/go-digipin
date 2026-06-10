package digipin

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

// knownVectors are coordinate->canonical-code pairs that lock the encoding to the
// DIGIPIN specification.
var knownVectors = []struct {
	name      string
	lat, lng  float64
	canonical string
	formatted string
}{
	{"Bangalore", 12.9716, 77.5946, "4P3JK852C9", "4P3 JK85 2C9"},
	{"NewDelhi", 28.6139, 77.2090, "39J438TJC7", "39J 438T JC7"},
	{"Mumbai", 19.0760, 72.8777, "4FK5958823", "4FK 5958 823"},
	{"Chennai", 13.0827, 80.2707, "4T384LL5L9", "4T3 84LL 5L9"},
	{"Kolkata", 22.5726, 88.3639, "2TFJ7F86MM", "2TF J7F8 6MM"},
	{"DelhiSample", 28.70486848, 77.10376044, "39JJ567L7P", "39J J567 L7P"},
	{"ReadmeEncode", 13.11179621, 80.20264269, "4T396F42L7", "4T3 96F4 2L7"},
	{"NECorner", 38.5, 99.5, "8888888888", "888 8888 888"},
	{"SWCorner", 2.5, 63.5, "LLLLLLLLLL", "LLL LLLL LLL"},
}

func TestEncodeKnownVectors(t *testing.T) {
	for _, s := range knownVectors {
		t.Run(s.name, func(t *testing.T) {
			c, err := Encode(s.lat, s.lng)
			if err != nil {
				t.Fatalf("Encode(%v,%v) error: %v", s.lat, s.lng, err)
			}
			if c.Canonical() != s.canonical {
				t.Errorf("canonical = %q, want %q", c.Canonical(), s.canonical)
			}
			if c.Formatted() != s.formatted {
				t.Errorf("formatted = %q, want %q", c.Formatted(), s.formatted)
			}
			if c.String() != s.formatted {
				t.Errorf("String = %q, want %q", c.String(), s.formatted)
			}
			if c.Level() != 10 {
				t.Errorf("Level = %d, want 10", c.Level())
			}
		})
	}
}

func TestEncodeInvalid(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng float64
	}{
		{"lat too low", -91, 77},
		{"lat too high", 91, 77},
		{"lng too low", 12, -181},
		{"lng too high", 12, 181},
		{"NaN lat", math.NaN(), 77},
		{"NaN lng", 12, math.NaN()},
		{"Inf lat", math.Inf(1), 77},
		{"Inf lng", 12, math.Inf(-1)},
		{"outside DIGIPIN lat", 50, 77},
		{"outside DIGIPIN lng", 12, 120},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Encode(tt.lat, tt.lng); !errors.Is(err, ErrInvalidCoordinate) {
				t.Errorf("err = %v, want ErrInvalidCoordinate", err)
			}
		})
	}
}

func TestEncodeLevel(t *testing.T) {
	for level := 1; level <= 10; level++ {
		c, err := EncodeLevel(12.9716, 77.5946, level)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		if c.Level() != level {
			t.Errorf("level %d: Level() = %d", level, c.Level())
		}
		if want := Code("4P3JK852C9"[:level]); c != want {
			t.Errorf("level %d: = %q, want %q", level, c, want)
		}
	}
}

func TestEncodeLevelInvalid(t *testing.T) {
	for _, level := range []int{0, -1, 11, 100} {
		if _, err := EncodeLevel(12.9716, 77.5946, level); !errors.Is(err, ErrInvalidLevel) {
			t.Errorf("level %d: err = %v, want ErrInvalidLevel", level, err)
		}
	}
	if _, err := EncodeLevel(200, 77, 5); !errors.Is(err, ErrInvalidCoordinate) {
		t.Errorf("err = %v, want ErrInvalidCoordinate", err)
	}
}

func TestDecodeKnownVector(t *testing.T) {
	coord, err := Decode("4P3JK852C9")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(coord.Latitude-12.971601) > 1e-5 || math.Abs(coord.Longitude-77.594584) > 1e-5 {
		t.Errorf("decode = %v, want ~(12.971601, 77.594584)", coord)
	}
}

func TestDecodeLenientForms(t *testing.T) {
	want, err := Decode("39JJ567L7P")
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range []string{"39J J567 L7P", "39J-J56-7L7P", "39jj567l7p", " 39J J567 L7P "} {
		got, err := Decode(form)
		if err != nil {
			t.Fatalf("Decode(%q): %v", form, err)
		}
		if !got.Equal(want) {
			t.Errorf("Decode(%q) = %v, want %v", form, got, want)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	for _, code := range []string{"", "---", "FFFFFFFFFFF", "0", "39JJ567L7I", "é", "39J@567L7P"} {
		if _, err := Decode(code); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("Decode(%q) err = %v, want ErrInvalidCode", code, err)
		}
	}
}

func TestParse(t *testing.T) {
	for _, form := range []string{"39JJ567L7P", "39J J567 L7P", "39J-J56-7L7P", "39j j567 l7p"} {
		c, err := Parse(form)
		if err != nil {
			t.Fatalf("Parse(%q): %v", form, err)
		}
		if c.Canonical() != "39JJ567L7P" {
			t.Errorf("Parse(%q) = %q", form, c.Canonical())
		}
	}
	if c, err := Parse("39J"); err != nil || c != "39J" {
		t.Errorf("Parse partial = %q, %v", c, err)
	}
	if _, err := Parse("bad!"); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("Parse invalid err = %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, s := range knownVectors {
		t.Run(s.name, func(t *testing.T) {
			c, _ := Encode(s.lat, s.lng)
			coord, err := Decode(c.Canonical())
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(coord.Latitude-s.lat) > 0.01 || math.Abs(coord.Longitude-s.lng) > 0.01 {
				t.Errorf("round trip drifted: got %v, want ~(%v,%v)", coord, s.lat, s.lng)
			}
		})
	}
}

func TestClampIndex(t *testing.T) {
	for in, want := range map[int]int{-5: 0, -1: 0, 0: 0, 2: 2, 3: 3, 4: 3, 9: 3} {
		if got := clampIndex(in); got != want {
			t.Errorf("clampIndex(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestValidateCoordinates(t *testing.T) {
	if err := ValidateCoordinates(12.9716, 77.5946); err != nil {
		t.Errorf("valid coords rejected: %v", err)
	}
	for _, c := range [][2]float64{{-91, 0}, {0, 181}, {math.NaN(), 0}, {math.Inf(1), 0}} {
		if err := ValidateCoordinates(c[0], c[1]); !errors.Is(err, ErrInvalidCoordinate) {
			t.Errorf("ValidateCoordinates(%v) = %v, want ErrInvalidCoordinate", c, err)
		}
	}
}

func TestValidateCode(t *testing.T) {
	if err := ValidateCode("39J-J56-7L7P"); err != nil {
		t.Errorf("valid code rejected: %v", err)
	}
	for _, code := range []string{"", "39J", "FFFFFFFFFFF", "39J@567L7P"} {
		if err := ValidateCode(code); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("ValidateCode(%q) = %v, want ErrInvalidCode", code, err)
		}
	}
}

func TestConstants(t *testing.T) {
	if len(charset) != 16 {
		t.Errorf("charset length = %d, want 16", len(charset))
	}
	for _, bad := range []rune{'0', '1', 'I', 'O'} {
		if strings.ContainsRune(charset, bad) {
			t.Errorf("charset must not contain ambiguous %q", bad)
		}
	}
	for _, ch := range []byte(charset) {
		if _, ok := gridPos[ch]; !ok {
			t.Errorf("charset char %q missing from gridPos", ch)
		}
	}
}

func TestCodeValid(t *testing.T) {
	if !Code("39JJ567L7P").Valid() {
		t.Error("valid code reported invalid")
	}
	for _, bad := range []Code{"", "FFFFFFFFFFF", "39J0", "39j"} {
		if bad.Valid() {
			t.Errorf("Code(%q).Valid() = true, want false", bad)
		}
	}
}

func TestCodeFormattedPartial(t *testing.T) {
	c, _ := EncodeLevel(12.9716, 77.5946, 5)
	if got := c.Formatted(); got != c.Canonical() {
		t.Errorf("partial Formatted = %q, want unchanged %q", got, c.Canonical())
	}
}

func TestCodeParent(t *testing.T) {
	c, _ := Encode(12.9716, 77.5946) // 4P3JK852C9
	p, ok := c.Parent()
	if !ok || p != "4P3JK852C" {
		t.Errorf("Parent = %q, %v; want 4P3JK852C, true", p, ok)
	}
	for _, code := range []Code{"", "4"} {
		if _, ok := code.Parent(); ok {
			t.Errorf("Parent(%q) ok = true, want false", code)
		}
	}
}

func TestCodeCoordinateAndBounds(t *testing.T) {
	c, _ := Encode(12.9716, 77.5946)
	coord := c.Coordinate()
	b := c.Bounds()
	if !b.Contains(coord) {
		t.Errorf("bounds %v do not contain center %v", b, coord)
	}
	want, _ := Decode(c.Canonical())
	if !coord.Equal(want) {
		t.Errorf("Coordinate = %v, want %v", coord, want)
	}
}

func TestCodeText(t *testing.T) {
	c, _ := Encode(28.70486848, 77.10376044) // 39JJ567L7P
	data, err := json.Marshal(c)
	if err != nil || string(data) != `"39JJ567L7P"` {
		t.Errorf("Marshal = %s, %v", data, err)
	}

	var out Code
	if err := json.Unmarshal([]byte(`"39J J567 L7P"`), &out); err != nil {
		t.Fatal(err)
	}
	if out != "39JJ567L7P" {
		t.Errorf("Unmarshal = %q", out)
	}

	type wrap struct {
		C Code `json:"c"`
	}
	b, _ := json.Marshal(wrap{C: c})
	if string(b) != `{"c":"39JJ567L7P"}` {
		t.Errorf("struct marshal = %s", b)
	}
	if err := out.UnmarshalText([]byte("bad!")); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("UnmarshalText invalid err = %v", err)
	}
}

func TestCodeNeighbors(t *testing.T) {
	c, _ := Encode(12.9716, 77.5946)
	ns := c.Neighbors()
	if len(ns) != 8 {
		t.Fatalf("interior neighbors = %d, want 8", len(ns))
	}
	seen := map[Code]bool{c: true}
	for _, n := range ns {
		if n.Level() != c.Level() {
			t.Errorf("neighbor level = %d, want %d", n.Level(), c.Level())
		}
		if seen[n] {
			t.Errorf("duplicate or self neighbor %q", n)
		}
		seen[n] = true
	}

	corner, _ := Encode(2.5, 63.5)
	if cn := corner.Neighbors(); len(cn) >= 8 {
		t.Errorf("corner neighbors = %d, want < 8", len(cn))
	}
}

func TestCodeDistanceAndMaps(t *testing.T) {
	a, _ := Encode(12.9716, 77.5946)
	z, _ := Encode(28.6139, 77.2090)
	if d := a.DistanceTo(z); math.Abs(d-1740000) > 30000 {
		t.Errorf("Code.DistanceTo = %v, want ~1740000", d)
	}
	if d := Distance(a, z); math.Abs(d-1740000) > 30000 {
		t.Errorf("Distance = %v, want ~1740000", d)
	}
	if !strings.HasPrefix(a.MapsLink(), "https://www.google.com/maps/search/?api=1&query=") {
		t.Errorf("Code.MapsLink = %q", a.MapsLink())
	}
}

func TestCodeGeoJSON(t *testing.T) {
	c, _ := EncodeLevel(12.9716, 77.5946, 1)
	var f struct {
		Type     string `json:"type"`
		Geometry struct {
			Type        string        `json:"type"`
			Coordinates [][][]float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(c.GeoJSON(), &f); err != nil {
		t.Fatalf("invalid GeoJSON: %v", err)
	}
	if f.Type != "Feature" || f.Geometry.Type != "Polygon" {
		t.Errorf("types: %s / %s", f.Type, f.Geometry.Type)
	}
	if len(f.Geometry.Coordinates) != 1 || len(f.Geometry.Coordinates[0]) != 5 {
		t.Errorf("ring should have 5 points, got %v", f.Geometry.Coordinates)
	}
	if f.Properties["digipin"] != c.Canonical() {
		t.Errorf("properties.digipin = %q", f.Properties["digipin"])
	}
	ring := f.Geometry.Coordinates[0]
	if ring[0][0] != ring[4][0] || ring[0][1] != ring[4][1] {
		t.Errorf("ring not closed: %v vs %v", ring[0], ring[4])
	}
}

func TestCoordinate(t *testing.T) {
	c := Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	if got := c.String(); got != "(12.971600, 77.594600)" {
		t.Errorf("String = %q", got)
	}
	if !c.Equal(Coordinate{Latitude: 12.9716, Longitude: 77.5946}) {
		t.Error("identical coords should be equal")
	}
	if c.Equal(Coordinate{Latitude: 13, Longitude: 77.5946}) {
		t.Error("different coords should not be equal")
	}
	near := Coordinate{Latitude: 12.97165, Longitude: 77.59465}
	if !c.WithinTolerance(near, 0) { // 0 -> DefaultTolerance
		t.Error("should be within default tolerance")
	}
	if c.WithinTolerance(near, 1e-9) {
		t.Error("should not be within tiny tolerance")
	}
	if got := c.MapsLink(); got != "https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946" {
		t.Errorf("MapsLink = %q", got)
	}
}

func TestBounds(t *testing.T) {
	b := Bounds{MinLatitude: 12.0, MaxLatitude: 13.0, MinLongitude: 77.0, MaxLongitude: 79.0}
	if got := b.String(); got != "(12.000000,77.000000) to (13.000000,79.000000)" {
		t.Errorf("String = %q", got)
	}
	if b.Area() != 2.0 || b.Height() != 1.0 || b.Width() != 2.0 {
		t.Errorf("Area/Height/Width = %v/%v/%v", b.Area(), b.Height(), b.Width())
	}
	if c := b.Center(); c.Latitude != 12.5 || c.Longitude != 78.0 {
		t.Errorf("Center = %v", c)
	}
	corners := b.Corners()
	if corners[0] != (Coordinate{12.0, 77.0}) || corners[2] != (Coordinate{13.0, 79.0}) {
		t.Errorf("Corners = %v", corners)
	}
	if !b.Contains(Coordinate{Latitude: 12.5, Longitude: 78.0}) {
		t.Error("should contain interior point")
	}
	if b.Contains(Coordinate{Latitude: 50, Longitude: 78}) {
		t.Error("should not contain exterior point")
	}

	var bf struct {
		Properties map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(b.GeoJSON(), &bf); err != nil {
		t.Fatalf("invalid Bounds GeoJSON: %v", err)
	}
	if len(bf.Properties) != 0 {
		t.Errorf("bounds properties should be empty, got %v", bf.Properties)
	}
}

func TestMapsLink(t *testing.T) {
	const want = "https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946"
	if got := MapsLink(12.9716, 77.5946); got != want {
		t.Errorf("MapsLink = %q, want %q", got, want)
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Encode(28.70486848, 77.10376044)
	}
}

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Decode("39JJ567L7P")
	}
}
