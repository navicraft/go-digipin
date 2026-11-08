package digipin

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// digipinGrid is the DIGIPIN 4x4 labeling grid. Row 0 is the northernmost band
// (maximum latitude); column 0 is the westernmost (minimum longitude). Its 16
// characters are the approved DIGIPIN set: digits 0 and 1 and letters I and O
// are excluded to avoid visual ambiguity.
var digipinGrid = [4][4]byte{
	{'F', 'C', '9', '8'},
	{'J', '3', '2', '7'},
	{'K', '4', '5', '6'},
	{'L', 'M', 'P', 'T'},
}

// gridPos is the reverse lookup of digipinGrid: byte -> row*4+col, or -1 for a
// byte that is not an approved (upper-case) DIGIPIN character. A flat array
// makes the lookup a single index with no hashing or allocation.
var gridPos = func() (t [256]int8) {
	for i := range t {
		t[i] = -1
	}
	for r := range 4 {
		for c := range 4 {
			t[digipinGrid[r][c]] = int8(r*4 + c)
		}
	}
	return t
}()

// DIGIPIN coverage area: the bounding box of India and its territorial waters.
const (
	minLat = 2.5
	maxLat = 38.5
	minLng = 63.5
	maxLng = 99.5
)

// codeLength is the number of characters in a full DIGIPIN.
const codeLength = 10

// DefaultTolerance is the default coordinate tolerance (in decimal degrees) used
// by Coordinate.WithinTolerance. It is roughly 11 meters at the equator.
const DefaultTolerance = 1e-4

// Errors reported by this package. Use errors.Is to test for them.
var (
	// ErrInvalidCoordinate is returned when coordinates are not finite or lie
	// outside the DIGIPIN coverage area.
	ErrInvalidCoordinate = errors.New("digipin: invalid coordinate")
	// ErrInvalidCode is returned when a DIGIPIN string is malformed.
	ErrInvalidCode = errors.New("digipin: invalid code")
	// ErrInvalidLevel is returned when an encoding level is outside [1, 10].
	ErrInvalidLevel = errors.New("digipin: invalid level")
)

// Encode converts latitude and longitude to a full 10-character DIGIPIN Code.
//
// Coordinates must lie within the DIGIPIN coverage area: latitude [2.5, 38.5]
// and longitude [63.5, 99.5]. Returns ErrInvalidCoordinate if the coordinates
// are out of range or not finite.
func Encode(latitude, longitude float64) (Code, error) {
	return EncodeLevel(latitude, longitude, codeLength)
}

// EncodeLevel converts latitude and longitude to a DIGIPIN Code truncated to the
// given precision level (1..10). Level 1 covers a ~1000 km cell; level 10 the
// full ~3.8 m cell. Encode is EncodeLevel with level 10.
//
// Returns ErrInvalidLevel if level is outside [1, 10], or ErrInvalidCoordinate
// if the coordinates are out of range or not finite.
func EncodeLevel(latitude, longitude float64, level int) (Code, error) {
	if level < 1 || level > codeLength {
		return "", fmt.Errorf("%w: %d (want 1..%d)", ErrInvalidLevel, level, codeLength)
	}
	if err := ValidateCoordinates(latitude, longitude); err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(level)

	lat0, lat1 := minLat, maxLat
	lng0, lng1 := minLng, maxLng

	for range level {
		latDiv := (lat1 - lat0) / 4.0
		lngDiv := (lng1 - lng0) / 4.0

		row := clampIndex(3 - int(math.Floor((latitude-lat0)/latDiv)))
		col := clampIndex(int(math.Floor((longitude - lng0) / lngDiv)))

		b.WriteByte(digipinGrid[row][col])

		lat0, lat1, lng0, lng1 = narrow(lat0, lat1, lng0, lng1, row, col)
	}

	return Code(b.String()), nil
}

// Parse validates a DIGIPIN string and returns it as a canonical Code. It is
// deliberately lenient: besides the specification's continuous form it accepts
// surrounding whitespace, spaces and hyphens anywhere, any letter case, and
// partial codes of 1..10 characters. Use ValidateCode for strict, specification-conformant checking.
// Returns ErrInvalidCode if the string is not a valid code.
func Parse(s string) (Code, error) {
	pin, err := normalize(s)
	if err != nil {
		return "", err
	}
	return Code(pin), nil
}

// Decode converts a DIGIPIN string to the coordinate at the center of its cell.
// It accepts the same lenient forms as Parse. Returns ErrInvalidCode if the
// string is malformed.
func Decode(s string) (Coordinate, error) {
	c, err := Parse(s)
	if err != nil {
		return Coordinate{}, err
	}
	return c.Coordinate(), nil
}

// walk replays a valid canonical DIGIPIN through the grid and returns the final
// cell bounds. Callers must check validity first.
func walk(pin string) (lat0, lat1, lng0, lng1 float64) {
	lat0, lat1 = minLat, maxLat
	lng0, lng1 = minLng, maxLng

	for i := range len(pin) {
		p := int(gridPos[pin[i]])
		lat0, lat1, lng0, lng1 = narrow(lat0, lat1, lng0, lng1, p/4, p%4)
	}
	return lat0, lat1, lng0, lng1
}

// narrow returns the sub-cell bounds for grid position (row, col) within the
// cell [lat0,lat1]x[lng0,lng1]. Row 0 is the northernmost (max-latitude) band
// and column 0 the westernmost (min-longitude). It is the single source of truth
// for the quadtree recurrence shared by EncodeLevel and walk.
func narrow(lat0, lat1, lng0, lng1 float64, row, col int) (nLat0, nLat1, nLng0, nLng1 float64) {
	latDiv := (lat1 - lat0) / 4.0
	lngDiv := (lng1 - lng0) / 4.0
	nLat1 = lat1 - latDiv*float64(row)
	nLat0 = lat1 - latDiv*float64(row+1)
	nLng0 = lng0 + lngDiv*float64(col)
	nLng1 = lng0 + lngDiv*float64(col+1)
	return nLat0, nLat1, nLng0, nLng1
}

// clampIndex constrains a grid index to [0, 3]. Together with the floor
// division in EncodeLevel it implements the specification's grid-line rule
// (section 4.4): a point on an interior grid line belongs to the cell to its
// north and/or east, except on the outer 38.5N and 99.5E lines, where the
// index would fall outside the grid and is clamped back to the southern or
// western cell.
func clampIndex(v int) int {
	if v < 0 {
		return 0
	}
	if v > 3 {
		return 3
	}
	return v
}

// upper upper-cases an ASCII letter and returns any other byte unchanged.
func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}

// normalize trims surrounding whitespace (as the reference decoder does),
// strips spaces and hyphens, upper-cases letters, validates that every
// remaining character is an approved DIGIPIN character, and checks the length is
// in [1, 10]. It returns the canonical continuous code.
func normalize(s string) (string, error) {
	s = strings.TrimSpace(s)
	if Code(s).Valid() {
		return s, nil // already canonical: no allocation
	}
	var b strings.Builder
	b.Grow(codeLength)
	for _, r := range s {
		if r == ' ' || r == '-' {
			continue
		}
		if r > 127 || gridPos[upper(byte(r))] < 0 {
			return "", fmt.Errorf("%w: bad character %q", ErrInvalidCode, r)
		}
		if b.Len() == codeLength {
			return "", fmt.Errorf("%w: longer than %d characters", ErrInvalidCode, codeLength)
		}
		b.WriteByte(upper(byte(r)))
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("%w: empty", ErrInvalidCode)
	}
	return b.String(), nil
}

// ValidateCoordinates reports whether latitude and longitude are finite and lie
// within the DIGIPIN coverage area (latitude [2.5, 38.5], longitude
// [63.5, 99.5]), i.e. whether Encode would accept them. Returns
// ErrInvalidCoordinate on failure.
func ValidateCoordinates(latitude, longitude float64) error {
	if math.IsNaN(latitude) || math.IsNaN(longitude) || math.IsInf(latitude, 0) || math.IsInf(longitude, 0) {
		return fmt.Errorf("%w: not finite", ErrInvalidCoordinate)
	}
	if latitude < minLat || latitude > maxLat {
		return fmt.Errorf("%w: latitude %v out of range [%v, %v]", ErrInvalidCoordinate, latitude, minLat, maxLat)
	}
	if longitude < minLng || longitude > maxLng {
		return fmt.Errorf("%w: longitude %v out of range [%v, %v]", ErrInvalidCoordinate, longitude, minLng, maxLng)
	}
	return nil
}

// ValidateCode reports whether s is a full DIGIPIN in a form the specification
// permits: 10 approved characters, either continuous ("39JJ567L7P") or grouped
// 3-4-3 with single spaces ("39J J567 L7P"). Like the reference decoder, it
// ignores letter case and surrounding whitespace. Hyphens, other separators,
// and partial codes are rejected; use Parse for lenient input. Returns
// ErrInvalidCode on failure.
func ValidateCode(s string) error {
	s = strings.TrimSpace(s)
	pin := s
	if len(s) == codeLength+2 && s[3] == ' ' && s[8] == ' ' {
		pin = s[:3] + s[4:8] + s[9:]
	}
	if len(pin) != codeLength {
		return fmt.Errorf("%w: %q is not a %d-character code (continuous or 3-4-3 spaced)",
			ErrInvalidCode, s, codeLength)
	}
	for i := range len(pin) {
		if gridPos[upper(pin[i])] < 0 {
			return fmt.Errorf("%w: bad character %q", ErrInvalidCode, pin[i])
		}
	}
	return nil
}
