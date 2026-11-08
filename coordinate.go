package digipin

import (
	"fmt"
	"math"
)

// Coordinate is a geographic point in WGS84 decimal degrees. Latitude is in
// [-90, +90] and longitude in [-180, +180].
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// String returns the coordinate with six decimal places, e.g.
// "(12.971600, 77.594600)".
func (c Coordinate) String() string {
	return fmt.Sprintf("(%.6f, %.6f)", c.Latitude, c.Longitude)
}

// Equal reports whether two coordinates are exactly equal, the same as ==.
//
// Before v1.1.0 Equal compared within 1e-6, which silently made go-cmp (which
// calls Equal methods) approximate. It is now exact.
//
// Deprecated: compare Coordinates with ==, or use WithinTolerance for an
// approximate comparison.
func (c Coordinate) Equal(other Coordinate) bool {
	return c == other
}

// WithinTolerance reports whether two coordinates are within tolerance decimal
// degrees on both axes. A tolerance of 0 or less uses DefaultTolerance.
func (c Coordinate) WithinTolerance(other Coordinate, tolerance float64) bool {
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	return math.Abs(c.Latitude-other.Latitude) < tolerance &&
		math.Abs(c.Longitude-other.Longitude) < tolerance
}
