package digipin

import "strconv"

// num formats a coordinate component as a compact, round-trippable JSON number.
func num(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// feature builds a GeoJSON Feature whose geometry is the bounds' rectangle as a
// closed Polygon ring (in lon,lat order, per the GeoJSON spec) and whose
// properties object is the given raw JSON (without the enclosing braces).
//
// The output is assembled directly rather than via encoding/json: every value is
// a finite float or an approved-charset string, so the result is always valid
// JSON and there is no error path. Tests round-trip it through encoding/json to
// confirm well-formedness.
func feature(b Bounds, properties string) []byte {
	minLatS, maxLatS := num(b.MinLatitude), num(b.MaxLatitude)
	minLngS, maxLngS := num(b.MinLongitude), num(b.MaxLongitude)

	ring := "[" +
		"[" + minLngS + "," + minLatS + "]," +
		"[" + maxLngS + "," + minLatS + "]," +
		"[" + maxLngS + "," + maxLatS + "]," +
		"[" + minLngS + "," + maxLatS + "]," +
		"[" + minLngS + "," + minLatS + "]" +
		"]"

	return []byte(`{"type":"Feature","geometry":{"type":"Polygon","coordinates":[` +
		ring + `]},"properties":{` + properties + `}}`)
}
