package digipin

import "encoding/json"

// geoJSONFeature is the wire shape of a GeoJSON Feature (RFC 7946).
type geoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   *geoJSONPolygon   `json:"geometry"`
	Properties map[string]string `json:"properties"`
}

type geoJSONPolygon struct {
	Type        string         `json:"type"`
	Coordinates [][][2]float64 `json:"coordinates"`
}

// feature builds a GeoJSON Feature whose geometry is the bounds' rectangle as a
// closed, counter-clockwise Polygon ring in lon,lat order, or null if the bounds
// are not a finite, non-empty box.
func feature(b Bounds, properties map[string]string) []byte {
	f := geoJSONFeature{Type: "Feature", Properties: properties}
	if b.isBox() {
		sw := [2]float64{b.MinLongitude, b.MinLatitude}
		se := [2]float64{b.MaxLongitude, b.MinLatitude}
		ne := [2]float64{b.MaxLongitude, b.MaxLatitude}
		nw := [2]float64{b.MinLongitude, b.MaxLatitude}
		f.Geometry = &geoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{sw, se, ne, nw, sw}}}
	}
	out, _ := json.Marshal(f) // cannot fail: isBox rules out NaN and Inf
	return out
}
