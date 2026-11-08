// Package digipin encodes and decodes DIGIPIN codes, India Post's offline,
// grid-based digital addressing system for India.
//
// DIGIPIN (Digital PIN) is a 10-character alphanumeric geocode developed by the
// Department of Posts in collaboration with IIT Hyderabad and NRSC, ISRO. It
// maps any latitude/longitude inside India to a short, human-readable code and
// back, with no network lookup or external database required. It is a
// geohash-like hierarchical encoding: each character refines the location to a
// sixteenth (a 4x4 sub-grid) of the previous cell, so shared prefixes mean
// nearby places.
//
// This package is an unofficial, pure-Go, zero-dependency implementation of the
// DIGIPIN Technical Specification (May 2026 revision) published by the
// Department of Posts at https://github.com/INDIAPOST-gov/digipin. Its encoder
// is tested to produce the same codes as the reference implementation.
//
// # The Code type
//
// A [Code] is a defined string type holding the canonical form of a DIGIPIN:
// 10 upper-case characters from 2-9, C, F, J, K, L, M, P, T with no
// separators, e.g. "39JJ567L7P". Because it is a value, two codes compare with
// == and a Code works as a map key or struct field. For display, the
// specification also permits grouping 3-4-3 with spaces, which [Code.Formatted]
// produces. Hyphens are not permitted by the specification.
//
//	Canonical : 39JJ567L7P
//	Formatted : 39J J567 L7P   (3-4-3, display only)
//
// [ValidateCode] checks a string against the specification's format,
// ignoring case and surrounding whitespace as the reference decoder does.
// [Parse] and [Decode] are deliberately lenient: they also tolerate hyphens,
// stray spaces, and partial codes.
//
// # Coverage area
//
// DIGIPIN covers the bounding box of India and its maritime Exclusive Economic
// Zone: latitude [2.5, 38.5] and longitude [63.5, 99.5], in EPSG:4326 (WGS84)
// decimal degrees. Encode rejects coordinates outside it. A point on a grid
// line belongs to the cell to its north and/or east, except on the outer 38.5N
// and 99.5E lines (specification section 4.4).
//
// # Precision levels
//
// A full code has 10 levels; [EncodeLevel] can stop earlier. Partial codes are
// an extension of this package, not part of the specification.
//
//	Level      1      2       3        4        5       6     7      8     9     10
//	Cell     ~1000km ~250km ~62.5km ~15.6km  ~3.9km  ~1km  ~250m  ~60m  ~15m  ~3.8m
//
// # Basic usage
//
//	c, err := digipin.Encode(28.70486848, 77.10376044)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(c.Canonical()) // 39JJ567L7P    (storage / interop)
//	fmt.Println(c)             // 39J J567 L7P  (String == Formatted)
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
