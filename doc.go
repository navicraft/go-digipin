// Package digipin encodes and decodes DIGIPIN codes, India Post's offline,
// grid-based digital postal addressing system for India.
//
// DIGIPIN (Digital Postal Index Number) is a 10-character alphanumeric geocode
// developed by India Post in collaboration with IIT Hyderabad and NRSC, ISRO.
// It maps any latitude/longitude inside India to a short, human-readable code
// and back, with no network lookup or external database required. It is a
// geohash-like hierarchical encoding: each character refines the location to a
// quarter of the previous cell, so shared prefixes mean nearby places.
//
// This package is a pure-Go, zero-dependency, allocation-light implementation of
// the DIGIPIN specification.
//
// # The Code type
//
// A [Code] is a defined string type holding the canonical, continuous form of a
// DIGIPIN, e.g. "39JJ567L7P". Because it is a value, two codes compare with ==
// and a Code works as a map key or struct field. For display, [Code.Formatted]
// groups it 3-4-3 with spaces. Hyphens are not part of the specification, but
// the parsing functions accept them (and spaces, and any letter case).
//
//	Canonical : 39JJ567L7P
//	Formatted : 39J J567 L7P   (3-4-3, display only)
//
// # Coverage area
//
// DIGIPIN covers the bounding box of India and its territorial waters: latitude
// [2.5, 38.5] and longitude [63.5, 99.5]. Encode rejects coordinates outside it.
//
// # Basic usage
//
//	c, err := digipin.Encode(28.70486848, 77.10376044)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(c)             // 39J J567 L7P  (String == Formatted)
//	fmt.Println(c.Canonical()) // 39JJ567L7P    (storage / interop)
//
//	coord, err := digipin.Decode("39JJ567L7P")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("%.6f, %.6f\n", coord.Latitude, coord.Longitude)
//
// # Beyond the basics
//
// In addition to [Encode] and [Decode], the package offers variable-precision
// encoding ([EncodeLevel]), parsing of any form to canonical ([Parse]), and
// methods on [Code] for hierarchy ([Code.Parent]), adjacent cells
// ([Code.Neighbors]), bounding boxes ([Code.Bounds]), great-circle distance
// ([Code.DistanceTo]), GeoJSON cell polygons ([Code.GeoJSON]), and Google Maps
// links ([Code.MapsLink]).
//
// All functions and methods are pure and safe for concurrent use.
package digipin
