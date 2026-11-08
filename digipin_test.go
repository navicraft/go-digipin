package digipin

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
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
	// Worked example from the specification, section 6 (Dak Bhawan). Section 6
	// prints it grouped 3-3-4 ("39J 49L L8T4"), but the normative format rule
	// (section 2c) is 3-4-3, which Formatted follows.
	{"DakBhawan", 28.622788, 77.213033, "39J49LL8T4", "39J 49LL 8T4"},
	// Example from the reference implementation's getDigiPin doc comment.
	{"ReferenceEncode", 13.11179621, 80.20264269, "4T396F42L7", "4T3 96F4 2L7"},
	{"Bangalore", 12.9716, 77.5946, "4P3JK852C9", "4P3 JK85 2C9"},
	{"NewDelhi", 28.6139, 77.2090, "39J438TJC7", "39J 438T JC7"},
	{"Mumbai", 19.0760, 72.8777, "4FK5958823", "4FK 5958 823"},
	{"Chennai", 13.0827, 80.2707, "4T384LL5L9", "4T3 84LL 5L9"},
	{"Kolkata", 22.5726, 88.3639, "2TFJ7F86MM", "2TF J7F8 6MM"},
	{"DelhiSample", 28.70486848, 77.10376044, "39JJ567L7P", "39J J567 L7P"},
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

// TestGridLineRule checks section 4.4 of the specification: a point on an
// interior grid line takes the cell to its north and/or east (so it is that
// cell's south-west corner), while points on the outer 38.5N and 99.5E lines
// take the cell to their south and/or west.
func TestGridLineRule(t *testing.T) {
	for level := 1; level <= 10; level++ {
		step := 36 / math.Pow(4, float64(level))
		lat, lng := minLat+step, minLng+2*step // an interior grid intersection
		for _, p := range []struct {
			name             string
			lat, lng         float64
			wantSouth, wantW float64 // expected MinLatitude, MinLongitude
		}{
			{"intersection", lat, lng, lat, lng},
			{"vertical line", lat + step/2, lng, lat, lng},
			{"horizontal line", lat, lng + step/2, lat, lng},
			{"top edge", maxLat, lng, maxLat - step, lng},
			{"right edge", lat, maxLng, lat, maxLng - step},
			{"top-right corner", maxLat, maxLng, maxLat - step, maxLng - step},
		} {
			c, err := EncodeLevel(p.lat, p.lng, level)
			if err != nil {
				t.Fatal(err)
			}
			b := c.Bounds()
			if p.name == "vertical line" {
				p.wantSouth = b.MinLatitude // only the column is under test
			}
			if p.name == "horizontal line" {
				p.wantW = b.MinLongitude // only the row is under test
			}
			if math.Abs(b.MinLatitude-p.wantSouth) > 1e-12 || math.Abs(b.MinLongitude-p.wantW) > 1e-12 {
				t.Errorf("level %d %s (%v, %v): cell %v, want SW corner (%v, %v)",
					level, p.name, p.lat, p.lng, b, p.wantSouth, p.wantW)
			}
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
	for _, form := range []string{"39J J567 L7P", "39J-J56-7L7P", "39jj567l7p", " 39J J567 L7P ", "\t39JJ567L7P\r\n"} {
		got, err := Decode(form)
		if err != nil {
			t.Fatalf("Decode(%q): %v", form, err)
		}
		if got != want {
			t.Errorf("Decode(%q) = %v, want %v", form, got, want)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	for _, code := range []string{"", "---", "FFFFFFFFFFF", "0", "39JJ567L7I", "é", "39J@567L7P", strings.Repeat("F", 1000)} {
		if _, err := Decode(code); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("Decode(%q) err = %v, want ErrInvalidCode", code, err)
		}
	}
	// The error names the character as the caller wrote it, not upper-cased.
	if _, err := Parse("39j-i"); err == nil || !strings.Contains(err.Error(), "'i'") {
		t.Errorf("Parse lower-case bad char err = %v, want it to mention 'i'", err)
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
	for _, c := range [][2]float64{{-91, 0}, {0, 181}, {math.NaN(), 0}, {math.Inf(1), 0}, {50, 77}, {12, 120}} {
		if err := ValidateCoordinates(c[0], c[1]); !errors.Is(err, ErrInvalidCoordinate) {
			t.Errorf("ValidateCoordinates(%v) = %v, want ErrInvalidCoordinate", c, err)
		}
	}
}

func TestValidateCode(t *testing.T) {
	for _, code := range []string{"39JJ567L7P", "39J J567 L7P", "39jj567l7p", " 39J J567 L7P\n", "\t39JJ567L7P\r\n"} {
		if err := ValidateCode(code); err != nil {
			t.Errorf("ValidateCode(%q) = %v, want nil", code, err)
		}
	}
	// The specification forbids hyphens and allows spaces only in a 3-4-3 grouping.
	for _, code := range []string{"", "39J", "FFFFFFFFFFF", "39J@567L7P", "39JJ567L7I",
		"39J-J56-7L7P", "39J-J567-L7P", "39JJ 567 L7P", "39J  J567L7P", "39J\tJ567\tL7P"} {
		if err := ValidateCode(code); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("ValidateCode(%q) = %v, want ErrInvalidCode", code, err)
		}
	}
}

func TestGrid(t *testing.T) {
	// The grid holds exactly the 16 approved characters, and gridPos inverts it.
	const approved = "23456789CFJKLMPT"
	valid := 0
	for b := range 256 {
		p := gridPos[b]
		if p < 0 {
			continue
		}
		valid++
		if digipinGrid[p/4][p%4] != byte(b) {
			t.Errorf("gridPos[%q] = %d does not point back to it", rune(b), p)
		}
		if !strings.ContainsRune(approved, rune(b)) {
			t.Errorf("unapproved character %q in grid", rune(b))
		}
	}
	if valid != len(approved) {
		t.Errorf("grid has %d characters, want %d", valid, len(approved))
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
	if coord != want {
		t.Errorf("Coordinate = %v, want %v", coord, want)
	}
	// Invalid codes yield zero values instead of a misleading location.
	for _, bad := range []Code{"", "zz", "39J0"} {
		if bad.Coordinate() != (Coordinate{}) || bad.Bounds() != (Bounds{}) || len(bad.Neighbors()) != 0 {
			t.Errorf("Code(%q): got %v / %v / %v, want zero values", bad, bad.Coordinate(), bad.Bounds(), bad.Neighbors())
		}
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
	b, err := json.Marshal(wrap{C: c})
	if err != nil || string(b) != `{"c":"39JJ567L7P"}` {
		t.Errorf("struct marshal = %s, %v", b, err)
	}
	if err := out.UnmarshalText([]byte("bad!")); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("UnmarshalText invalid err = %v", err)
	}

	// The zero Code round-trips, so optional Code fields work.
	var zero wrap
	b, err = json.Marshal(zero)
	if err != nil || string(b) != `{"c":""}` {
		t.Fatalf("zero marshal = %s, %v", b, err)
	}
	got := wrap{C: "39J"}
	if err := json.Unmarshal(b, &got); err != nil || got.C != "" {
		t.Errorf("zero unmarshal = %q, %v", got.C, err)
	}

	// An invalid Code refuses to marshal rather than writing unreadable data.
	if _, err := json.Marshal(wrap{C: "bad!"}); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("invalid marshal err = %v, want ErrInvalidCode", err)
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
	// Invalid codes give NaN and "" rather than distances or links from (0,0).
	if d := a.DistanceTo("zz"); !math.IsNaN(d) {
		t.Errorf("DistanceTo(invalid) = %v, want NaN", d)
	}
	if d := Code("zz").DistanceTo(a); !math.IsNaN(d) {
		t.Errorf("invalid.DistanceTo = %v, want NaN", d)
	}
	if l := Code("4p3jk852c9").MapsLink(); l != "" {
		t.Errorf("invalid MapsLink = %q, want empty", l)
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

	// Invalid codes, even with bytes that are not valid UTF-8, give valid JSON
	// with a null geometry.
	for _, bad := range []Code{"", "bad\x00\xff\U0001F600"} {
		var g map[string]any
		if err := json.Unmarshal(bad.GeoJSON(), &g); err != nil {
			t.Fatalf("Code(%q).GeoJSON invalid JSON: %v: %s", bad, err, bad.GeoJSON())
		}
		if g["geometry"] != nil || len(g["properties"].(map[string]any)) != 0 {
			t.Errorf("Code(%q).GeoJSON = %s, want null geometry and empty properties", bad, bad.GeoJSON())
		}
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
	if c.Equal(Coordinate{Latitude: 12.9716 + 1e-9, Longitude: 77.5946}) {
		t.Error("Equal must be exact")
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
	if !strings.Contains(string(b.GeoJSON()), `"properties":{}`) {
		t.Errorf("bounds properties should be {}, got %s", b.GeoJSON())
	}

	// Degenerate or non-finite bounds still produce valid JSON, with null geometry.
	nan, inf := math.NaN(), math.Inf(1)
	for _, bad := range []Bounds{{}, {MinLatitude: nan, MaxLatitude: 1, MaxLongitude: 1}, {MaxLatitude: 1, MaxLongitude: inf}, {MinLatitude: 2, MaxLatitude: 1, MaxLongitude: 1}} {
		var g map[string]any
		if err := json.Unmarshal(bad.GeoJSON(), &g); err != nil || g["geometry"] != nil {
			t.Errorf("%v.GeoJSON = %s, %v; want valid JSON with null geometry", bad, bad.GeoJSON(), err)
		}
	}
}

func TestMapsLink(t *testing.T) {
	const want = "https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946"
	if got := MapsLink(12.9716, 77.5946); got != want {
		t.Errorf("MapsLink = %q, want %q", got, want)
	}
}

// officialEncode is a line-for-line port of getDigiPin from India Post's
// reference implementation (github.com/INDIAPOST-gov/digipin, src/digipin.js).
// Its bounds update differs in form from narrow, so agreement proves the two
// recurrences produce identical codes.
func officialEncode(lat, lon float64) string {
	minLat, maxLat, minLon, maxLon := 2.5, 38.5, 63.5, 99.5
	digiPin := ""
	for range 10 {
		latDiv := (maxLat - minLat) / 4
		lonDiv := (maxLon - minLon) / 4
		row := 3 - int(math.Floor((lat-minLat)/latDiv))
		col := int(math.Floor((lon - minLon) / lonDiv))
		row = max(0, min(row, 3))
		col = max(0, min(col, 3))
		digiPin += string(digipinGrid[row][col])
		maxLat = minLat + latDiv*float64(4-row)
		minLat = minLat + latDiv*float64(3-row)
		minLon = minLon + lonDiv*float64(col)
		maxLon = minLon + lonDiv
	}
	return digiPin
}

func TestOfficialConformance(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	cell := 36.0 / math.Pow(4, 10) // level-10 cell size in degrees
	check := func(lat, lng float64) {
		c, err := Encode(lat, lng)
		if err != nil {
			t.Fatalf("Encode(%v, %v): %v", lat, lng, err)
		}
		if want := officialEncode(lat, lng); string(c) != want {
			t.Fatalf("Encode(%v, %v) = %s, official = %s", lat, lng, c, want)
		}
	}
	for range 100_000 {
		check(minLat+r.Float64()*36, minLng+r.Float64()*36)
		// Points exactly on, and one ULP either side of, level-10 cell edges.
		lat := minLat + float64(r.IntN(1<<20))*cell
		lng := minLng + float64(r.IntN(1<<20))*cell
		check(lat, lng)
		check(min(math.Nextafter(lat, 90), maxLat), min(math.Nextafter(lng, 180), maxLng))
		check(max(math.Nextafter(lat, -90), minLat), max(math.Nextafter(lng, -180), minLng))
	}
}

func BenchmarkEncode(b *testing.B) {
	for b.Loop() {
		_, _ = Encode(28.70486848, 77.10376044)
	}
}

func BenchmarkDecode(b *testing.B) {
	for b.Loop() {
		_, _ = Decode("39JJ567L7P")
	}
}
