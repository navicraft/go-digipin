package digipin

import (
	"net/url"
	"strconv"
)

// MapsLink returns a Google Maps URL that points at the given coordinate.
//
// Deprecated: use Coordinate{Latitude: latitude, Longitude: longitude}.MapsLink().
func MapsLink(latitude, longitude float64) string {
	return Coordinate{Latitude: latitude, Longitude: longitude}.MapsLink()
}

// MapsLink returns a Google Maps URL pointing at this coordinate, using the
// documented Maps URL "search" form:
//
//	https://www.google.com/maps/search/?api=1&query=<lat>,<lng>
func (c Coordinate) MapsLink() string {
	query := strconv.FormatFloat(c.Latitude, 'f', -1, 64) + "," +
		strconv.FormatFloat(c.Longitude, 'f', -1, 64)
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(query)
}
