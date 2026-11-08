package digipin

import (
	"fmt"
	"math"
)

// Bounds is a geographic bounding box in WGS84 decimal degrees, describing the
// rectangular cell covered by a DIGIPIN code.
type Bounds struct {
	MinLatitude  float64 `json:"min_latitude"`
	MaxLatitude  float64 `json:"max_latitude"`
	MinLongitude float64 `json:"min_longitude"`
	MaxLongitude float64 `json:"max_longitude"`
}

// String returns a human-readable representation, e.g.
// "(12.970000,77.594000) to (12.972000,77.596000)".
func (b Bounds) String() string {
	return fmt.Sprintf("(%.6f,%.6f) to (%.6f,%.6f)",
		b.MinLatitude, b.MinLongitude, b.MaxLatitude, b.MaxLongitude)
}

// Area returns the box area in square degrees. This is a planar approximation
// and does not account for the Earth's curvature.
//
// Deprecated: square degrees are not a meaningful unit of area (a degree of
// longitude shrinks toward the poles). Use Width and Height instead.
func (b Bounds) Area() float64 {
	return (b.MaxLatitude - b.MinLatitude) * (b.MaxLongitude - b.MinLongitude)
}

// Width returns the longitudinal span of the bounds in degrees.
func (b Bounds) Width() float64 {
	return b.MaxLongitude - b.MinLongitude
}

// Height returns the latitudinal span of the bounds in degrees.
func (b Bounds) Height() float64 {
	return b.MaxLatitude - b.MinLatitude
}

// Center returns the coordinate at the center of the bounds.
func (b Bounds) Center() Coordinate {
	return Coordinate{
		Latitude:  (b.MinLatitude + b.MaxLatitude) / 2.0,
		Longitude: (b.MinLongitude + b.MaxLongitude) / 2.0,
	}
}

// Corners returns the four corners in the order SW, SE, NE, NW.
func (b Bounds) Corners() [4]Coordinate {
	return [4]Coordinate{
		{Latitude: b.MinLatitude, Longitude: b.MinLongitude},
		{Latitude: b.MinLatitude, Longitude: b.MaxLongitude},
		{Latitude: b.MaxLatitude, Longitude: b.MaxLongitude},
		{Latitude: b.MaxLatitude, Longitude: b.MinLongitude},
	}
}

// Contains reports whether the coordinate lies within the bounds (inclusive).
func (b Bounds) Contains(coord Coordinate) bool {
	return coord.Latitude >= b.MinLatitude &&
		coord.Latitude <= b.MaxLatitude &&
		coord.Longitude >= b.MinLongitude &&
		coord.Longitude <= b.MaxLongitude
}

// GeoJSON returns the bounds as a GeoJSON Feature with an empty properties
// object. Bounds that are not a finite, non-empty box (for example the zero
// Bounds, or ones holding NaN) yield a Feature with a null geometry.
func (b Bounds) GeoJSON() []byte {
	return feature(b, map[string]string{})
}

// isBox reports whether b is a finite box with positive width and height.
func (b Bounds) isBox() bool {
	for _, v := range [4]float64{b.MinLatitude, b.MaxLatitude, b.MinLongitude, b.MaxLongitude} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return b.MinLatitude < b.MaxLatitude && b.MinLongitude < b.MaxLongitude
}
