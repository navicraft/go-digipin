package digipin

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// charset lists the 16 approved DIGIPIN characters in order. Digits 0 and 1 and
// letters I and O are intentionally excluded to avoid visual ambiguity.
const charset = "23456789CFJKLMPT"

// digipinGrid is the DIGIPIN 4x4 labeling grid. Row 0 is the northernmost band
// (maximum latitude); column 0 is the westernmost (minimum longitude).
var digipinGrid = [4][4]byte{
	{'F', 'C', '9', '8'},
	{'J', '3', '2', '7'},
	{'K', '4', '5', '6'},
	{'L', 'M', 'P', 'T'},
}

// gridPos is the reverse lookup of digipinGrid: character -> [row, col]. It is
// built once in init so walk needs no per-character search and normalize can
// validate characters in O(1).
var gridPos = make(map[byte][2]int, 16)

func init() {
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			gridPos[digipinGrid[r][c]] = [2]int{r, c}
		}
	}
}

// DIGIPIN coverage area: the bounding box of India and its territorial waters.
const (
	minLat = 2.5
	maxLat = 38.5
	minLng = 63.5
	maxLng = 99.5
)

// Global coordinate limits and code geometry.
const (
	globalMinLat = -90.0
	globalMaxLat = 90.0
	globalMinLng = -180.0
	globalMaxLng = 180.0
	codeLength   = 10
)

// DefaultTolerance is the default coordinate tolerance (in decimal degrees) used
// by Coordinate.WithinTolerance. It is roughly 11 meters at the equator.
const DefaultTolerance = 1e-4

// Errors reported by this package. Use errors.Is to test for them.
var (
	// ErrInvalidCoordinate is returned when coordinates are out of range or not finite.
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
	if err := validateCoordinates(latitude, longitude); err != nil {
		return "", err
	}
	if latitude < minLat || latitude > maxLat || longitude < minLng || longitude > maxLng {
		return "", fmt.Errorf("%w: outside DIGIPIN bounds [%.1f,%.1f]x[%.1f,%.1f]",
			ErrInvalidCoordinate, minLat, maxLat, minLng, maxLng)
	}

	var b strings.Builder
	b.Grow(level)

	lat0, lat1 := minLat, maxLat
	lng0, lng1 := minLng, maxLng

	for i := 0; i < level; i++ {
		latDiv := (lat1 - lat0) / 4.0
		lngDiv := (lng1 - lng0) / 4.0

		row := clampIndex(3 - int(math.Floor((latitude-lat0)/latDiv)))
		col := clampIndex(int(math.Floor((longitude - lng0) / lngDiv)))

		b.WriteByte(digipinGrid[row][col])

		lat0, lat1, lng0, lng1 = narrow(lat0, lat1, lng0, lng1, row, col)
	}

	return Code(b.String()), nil
}

// Parse validates a DIGIPIN string in any accepted form (continuous, spaced, or
// hyphenated, any case) and returns it as a canonical Code. Partial codes of
// 1..10 characters are accepted. Returns ErrInvalidCode if the string is not a
// valid code.
func Parse(s string) (Code, error) {
	pin, err := normalize(s)
	if err != nil {
		return "", err
	}
	return Code(pin), nil
}

// Decode converts a DIGIPIN string to the coordinate at the center of its cell.
// It accepts continuous, spaced, and hyphenated input, any case, and partial
// codes. Returns ErrInvalidCode if the string is malformed.
func Decode(s string) (Coordinate, error) {
	c, err := Parse(s)
	if err != nil {
		return Coordinate{}, err
	}
	return c.Coordinate(), nil
}

// walk replays a normalized (validated, separator-free, upper-case) DIGIPIN
// through the grid and returns the final cell bounds. Every character is
// guaranteed to be in gridPos, so there is no error path.
func walk(pin string) (lat0, lat1, lng0, lng1 float64) {
	lat0, lat1 = minLat, maxLat
	lng0, lng1 = minLng, maxLng

	for i := 0; i < len(pin); i++ {
		rc := gridPos[pin[i]]
		lat0, lat1, lng0, lng1 = narrow(lat0, lat1, lng0, lng1, rc[0], rc[1])
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

// clampIndex constrains a grid index to [0, 3]. Coordinates on the northern or
// eastern edge of a cell can produce an index of -1 or 4 from the floor division;
// clamping keeps them in the boundary cell.
func clampIndex(v int) int {
	if v < 0 {
		return 0
	}
	if v > 3 {
		return 3
	}
	return v
}

// normalize strips spaces and hyphens, upper-cases letters, validates that every
// remaining character is an approved DIGIPIN character, and checks the length is
// in [1, 10]. It returns the canonical continuous code.
func normalize(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '-' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if r > 127 {
			return "", fmt.Errorf("%w: bad character %q", ErrInvalidCode, r)
		}
		if _, ok := gridPos[byte(r)]; !ok {
			return "", fmt.Errorf("%w: bad character %q", ErrInvalidCode, r)
		}
		b.WriteByte(byte(r))
	}
	out := b.String()
	if len(out) < 1 || len(out) > codeLength {
		return "", fmt.Errorf("%w: length %d (want 1..%d)", ErrInvalidCode, len(out), codeLength)
	}
	return out, nil
}

func validateCoordinates(lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return fmt.Errorf("%w: not finite", ErrInvalidCoordinate)
	}
	if lat < globalMinLat || lat > globalMaxLat || lng < globalMinLng || lng > globalMaxLng {
		return fmt.Errorf("%w: latitude [-90,90], longitude [-180,180]", ErrInvalidCoordinate)
	}
	return nil
}

// ValidateCoordinates reports whether latitude and longitude are finite and within
// the global ranges [-90, 90] and [-180, 180]. It does not check the DIGIPIN
// coverage area. Returns ErrInvalidCoordinate on failure.
func ValidateCoordinates(latitude, longitude float64) error {
	return validateCoordinates(latitude, longitude)
}

// ValidateCode reports whether a string is a well-formed full (10-character)
// DIGIPIN. Spaces and hyphens are tolerated and case is ignored; partial codes
// are rejected. Returns ErrInvalidCode on failure.
func ValidateCode(s string) error {
	pin, err := normalize(s)
	if err != nil {
		return err
	}
	if len(pin) != codeLength {
		return fmt.Errorf("%w: must be %d characters, got %d", ErrInvalidCode, codeLength, len(pin))
	}
	return nil
}
