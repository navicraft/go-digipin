package digipin

import "math"

// earthRadiusMeters is the mean radius of the Earth, used for haversine distance.
const earthRadiusMeters = 6371000.0

// DistanceTo returns the great-circle distance, in meters, between two
// coordinates using the haversine formula.
func (c Coordinate) DistanceTo(other Coordinate) float64 {
	lat1 := c.Latitude * math.Pi / 180.0
	lat2 := other.Latitude * math.Pi / 180.0
	dLat := (other.Latitude - c.Latitude) * math.Pi / 180.0
	dLng := (other.Longitude - c.Longitude) * math.Pi / 180.0

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// Distance returns the great-circle distance, in meters, between the centers of
// two DIGIPIN cells, or NaN if either Code is not Valid.
//
// Deprecated: use a.DistanceTo(b).
func Distance(a, b Code) float64 {
	return a.DistanceTo(b)
}
