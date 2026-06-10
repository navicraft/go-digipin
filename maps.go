package digipin

import (
	"net/url"
	"strconv"
)

// MapsLink returns a Google Maps URL that points at the given coordinate, using
// the documented Maps URL "search" form:
//
//	https://www.google.com/maps/search/?api=1&query=<lat>,<lng>
func MapsLink(latitude, longitude float64) string {
	query := strconv.FormatFloat(latitude, 'f', -1, 64) + "," +
		strconv.FormatFloat(longitude, 'f', -1, 64)
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(query)
}

// MapsLink returns a Google Maps URL pointing at this coordinate.
func (c Coordinate) MapsLink() string {
	return MapsLink(c.Latitude, c.Longitude)
}
