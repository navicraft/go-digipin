package digipin

import "strconv"

// Code is a DIGIPIN: the canonical, continuous (separator-free) encoding of a
// location, e.g. "39JJ567L7P". Because it is a string value, two codes compare
// directly with ==, and it can be used as a map key or struct field.
//
// A Code is meant to be produced by Encode, EncodeLevel, or Parse, which
// guarantee it is valid. Methods assume that invariant; call Valid to check a
// Code obtained from an untrusted source.
type Code string

// Canonical returns the continuous, separator-free form, e.g. "39JJ567L7P".
func (c Code) Canonical() string {
	return string(c)
}

// Formatted returns a human-readable rendering. Full 10-character codes are
// grouped 3-4-3 with spaces, e.g. "39J J567 L7P"; partial codes are returned
// unchanged.
func (c Code) Formatted() string {
	if len(c) != codeLength {
		return string(c)
	}
	return string(c[0:3]) + " " + string(c[3:7]) + " " + string(c[7:10])
}

// String implements fmt.Stringer and returns the Formatted form.
func (c Code) String() string {
	return c.Formatted()
}

// Level returns the precision level: the number of characters, 1 (coarsest) to
// 10 (finest).
func (c Code) Level() int {
	return len(c)
}

// Valid reports whether the Code is a well-formed canonical DIGIPIN: 1..10
// characters, all from the approved set, with no separators or lower case.
func (c Code) Valid() bool {
	if len(c) < 1 || len(c) > codeLength {
		return false
	}
	for i := 0; i < len(c); i++ {
		if _, ok := gridPos[c[i]]; !ok {
			return false
		}
	}
	return true
}

// Coordinate returns the point at the center of the code's cell.
func (c Code) Coordinate() Coordinate {
	lat0, lat1, lng0, lng1 := walk(string(c))
	return Coordinate{
		Latitude:  (lat0 + lat1) / 2.0,
		Longitude: (lng0 + lng1) / 2.0,
	}
}

// Bounds returns the geographic bounding box covered by the code's cell.
func (c Code) Bounds() Bounds {
	lat0, lat1, lng0, lng1 := walk(string(c))
	return Bounds{
		MinLatitude:  lat0,
		MaxLatitude:  lat1,
		MinLongitude: lng0,
		MaxLongitude: lng1,
	}
}

// Parent returns the code one level coarser (the ancestor cell that contains
// this one) by dropping the final character. ok is false for a code with fewer
// than two characters, which has no parent.
func (c Code) Parent() (parent Code, ok bool) {
	if len(c) < 2 {
		return "", false
	}
	return c[:len(c)-1], true
}

// Neighbors returns the cells immediately adjacent to this one at the same
// precision level, in reading order (NW, N, NE, W, E, SW, S, SE). Cells that
// fall outside the DIGIPIN coverage area (for a code on the edge of the bounding
// box) are omitted, so an interior cell yields 8 neighbors and an edge or corner
// cell yields fewer.
func (c Code) Neighbors() []Code {
	b := c.Bounds()
	level := len(c)
	center := b.Center()
	dLat := b.Height()
	dLng := b.Width()

	neighbors := make([]Code, 0, 8)
	for dRow := -1; dRow <= 1; dRow++ {
		for dCol := -1; dCol <= 1; dCol++ {
			if dRow == 0 && dCol == 0 {
				continue
			}
			n, err := EncodeLevel(center.Latitude-float64(dRow)*dLat, center.Longitude+float64(dCol)*dLng, level)
			if err != nil {
				continue
			}
			neighbors = append(neighbors, n)
		}
	}
	return neighbors
}

// DistanceTo returns the great-circle distance, in meters, between the centers of
// this code's cell and another.
func (c Code) DistanceTo(other Code) float64 {
	return c.Coordinate().DistanceTo(other.Coordinate())
}

// MapsLink returns a Google Maps URL pointing at the center of the code's cell.
func (c Code) MapsLink() string {
	return c.Coordinate().MapsLink()
}

// MarshalText implements encoding.TextMarshaler, emitting the canonical form.
// This makes Code marshal as a plain string in JSON and other text encodings.
func (c Code) MarshalText() ([]byte, error) {
	return []byte(c), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, normalizing any accepted
// form to canonical. Returns ErrInvalidCode if the text is not a valid code.
func (c *Code) UnmarshalText(text []byte) error {
	pin, err := normalize(string(text))
	if err != nil {
		return err
	}
	*c = Code(pin)
	return nil
}

// GeoJSON returns the code's cell as a GeoJSON Feature: a Polygon of the cell's
// bounding box with the canonical code under the "digipin" property.
func (c Code) GeoJSON() []byte {
	return feature(c.Bounds(), `"digipin":`+strconv.Quote(string(c)))
}
