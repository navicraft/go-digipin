<p align="center">
  <img src="docs/digipin-logo-blue.png" alt="DIGIPIN symbol" width="96">
</p>

# go-digipin

[![Go Reference](https://pkg.go.dev/badge/github.com/navicraft/go-digipin.svg)](https://pkg.go.dev/github.com/navicraft/go-digipin)
[![Go Report Card](https://goreportcard.com/badge/github.com/navicraft/go-digipin)](https://goreportcard.com/report/github.com/navicraft/go-digipin)
[![CI](https://github.com/navicraft/go-digipin/actions/workflows/ci.yml/badge.svg)](https://github.com/navicraft/go-digipin/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/navicraft/go-digipin)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-00ADD8)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**DIGIPIN encoder and decoder for Go**. Convert any latitude/longitude in India to
a short DIGIPIN geocode and back, offline, with zero dependencies.

DIGIPIN (Digital PIN) is India Post's grid-based digital addressing system. It
maps a coordinate to a 10-character alphanumeric code using a hierarchical 4×4
grid, so nearby places share code prefixes, like a geohash for India.

This is an **unofficial** implementation of the
[DIGIPIN Technical Specification](https://github.com/INDIAPOST-gov/digipin/blob/main/docs/DIGIPIN_Technical_Document.md)
(May 2026 revision) published by the Department of Posts, Government of India.
It is tested to produce the same codes as the
[official reference implementation](https://github.com/INDIAPOST-gov/digipin).
It is not affiliated with or endorsed by India Post.

```go
d, _ := digipin.Encode(28.622788, 77.213033) // Dak Bhawan, the spec's worked example
fmt.Println(d.Canonical()) // 39J49LL8T4
fmt.Println(d.Formatted()) // 39J 49LL 8T4

c, _ := digipin.Decode("39J49LL8T4")
fmt.Printf("%.6f, %.6f\n", c.Latitude, c.Longitude) // 28.622793, 77.213049 (cell center)
```

## Features

- **Encode / decode** coordinates ⟷ DIGIPIN, matching the official reference
  implementation, including the grid-line rule (spec §4.4).
- **Spec-conformant formats**: the canonical continuous code (`39JJ567L7P`) for
  storage and interchange, and the 3-4-3 spaced display form (`39J J567 L7P`).
- **Strict validation** (`ValidateCode`) per the spec, plus **lenient parsing**
  (`Parse`, `Decode`) that also tolerates hyphens, lower case, and partial codes.
- **Variable precision**: encode to any level 1–10 (`EncodeLevel`).
- **Spatial helpers**: bounding boxes, adjacent-cell neighbors, parent cells,
  haversine distance, GeoJSON cell polygons, and Google Maps links.
- **Zero dependencies**, pure Go, goroutine-safe. Decoding a canonical code
  allocates nothing.
- **100% test coverage**, fuzz tests, and a conformance test against the
  official algorithm.

## Install

```sh
go get github.com/navicraft/go-digipin
```

```go
import digipin "github.com/navicraft/go-digipin"
```

## Coverage area & code format

DIGIPIN covers India's bounding box, including its maritime Exclusive Economic
Zone: latitude **2.5–38.5**, longitude **63.5–99.5** (EPSG:4326 / WGS84).

Per the specification, a DIGIPIN is exactly 10 upper-case characters from
`2 3 4 5 6 7 8 9 C F J K L M P T` (no 0, 1, I, O). It is continuous, with no
hyphens or other punctuation. For display it may be grouped 3-4-3 with spaces.

| Level | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Cell size | ~1000 km | ~250 km | ~62.5 km | ~15.6 km | ~3.9 km | ~1 km | ~250 m | ~60 m | ~15 m | ~3.8 m |

## Usage

### Encode / decode

`Encode` returns a `Code`, a defined string type holding the canonical form.

```go
c, err := digipin.Encode(12.9716, 77.5946) // c is a digipin.Code value
if err != nil { /* outside India's bounding box, or not finite */ }

c.Canonical() // "4P3JK852C9"
c.Formatted() // "4P3 JK85 2C9"
c.Level()     // 10
c == "4P3JK852C9" // codes compare directly

// Decode accepts both specification forms (and, leniently, hyphens and lower case).
digipin.Decode("4P3JK852C9")
digipin.Decode("4P3 JK85 2C9")
```

### Validation

```go
digipin.ValidateCode("4P3JK852C9")   // nil
digipin.ValidateCode("4P3 JK85 2C9") // nil
digipin.ValidateCode("4P3-JK8-52C9") // ErrInvalidCode: the spec forbids hyphens

digipin.ValidateCoordinates(12.9716, 77.5946) // nil
digipin.ValidateCoordinates(51.5, -0.12)      // ErrInvalidCoordinate: outside India
```

### Variable precision and hierarchy

```go
c, _ := digipin.EncodeLevel(12.9716, 77.5946, 4) // "4P3J"
p, ok := c.Parent()                              // "4P3", true
```

### Bounds, neighbors, distance

```go
c, _ := digipin.Parse("4P3JK852C9")
b := c.Bounds()
b.Center()  // Coordinate
b.Corners() // [4]Coordinate (SW, SE, NE, NW)

ns := c.Neighbors() // up to 8 adjacent cells at the same level

a, _ := digipin.Encode(12.9716, 77.5946)
z, _ := digipin.Encode(28.6139, 77.2090)
m := a.DistanceTo(z) // meters
```

### GeoJSON and Maps links

```go
gj := c.GeoJSON() // GeoJSON Feature (cell polygon + "digipin" property)
link := digipin.Coordinate{Latitude: 12.9716, Longitude: 77.5946}.MapsLink()
// https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946
```

### JSON

`Code` marshals to and from its canonical JSON string. The zero `Code` marshals
as `""` and unmarshals back, so optional fields work.

```go
data, _ := json.Marshal(c)       // "4P3JK852C9"
var x digipin.Code
_ = json.Unmarshal([]byte(`"4P3 JK85 2C9"`), &x) // x == "4P3JK852C9"
```

## Command-line tool

```sh
go install github.com/navicraft/go-digipin/cmd/digipin@latest
```

From v1.1.0 on, prebuilt binaries for Linux, macOS, and Windows are attached
to each [release](https://github.com/navicraft/go-digipin/releases).

```sh
digipin encode 28.622788 77.213033                  # 39J49LL8T4
digipin encode 28.622788 77.213033 --format spaced  # 39J 49LL 8T4
digipin encode 28.622788 77.213033 --maps           # code<TAB>maps URL
digipin decode 39J49LL8T4 --json                    # {"digipin":"39J49LL8T4","latitude":"28.622793","longitude":"77.213049"}
digipin decode 39J 49LL 8T4                         # spaced codes need no quotes
digipin bounds 39J49LL8T4 --geojson
digipin neighbors 39J49LL8T4

# Batch mode reads stdin (lat,lng per line for encode; codes for the rest).
# Output line N always matches input line N; failed lines print an empty line
# and report "error: line N: ..." on stderr.
printf '28.7049,77.1038\n12.9716,77.5946\n' | digipin encode --batch
```

Exit status: 0 on success, 1 if a conversion failed, 2 on a usage error.

## API overview

| Symbol | Description |
| --- | --- |
| `Encode`, `EncodeLevel` | Coordinates → `Code` (full or level 1–10) |
| `Decode` | DIGIPIN string → center `Coordinate` |
| `Parse` | Lenient: any accepted form → canonical `Code` |
| `ValidateCode` | Strict: is this a spec-conformant DIGIPIN? |
| `ValidateCoordinates` | Would `Encode` accept these coordinates? |
| `Code.Canonical` / `.Formatted` / `.Level` / `.Valid` / `.Parent` / `.Neighbors` / `.Coordinate` / `.Bounds` / `.DistanceTo` / `.MapsLink` / `.GeoJSON` | Code helpers |
| `Coordinate.DistanceTo` / `.WithinTolerance` / `.MapsLink` | Coordinate helpers |
| `Bounds.Center` / `.Corners` / `.Width` / `.Height` / `.Contains` / `.GeoJSON` | Bounding-box helpers |

Full documentation: **[pkg.go.dev/github.com/navicraft/go-digipin](https://pkg.go.dev/github.com/navicraft/go-digipin)**.

## License

[MIT](LICENSE). DIGIPIN, its specification, and the DIGIPIN symbol belong to the
Department of Posts, Government of India; see the official repository's
[disclaimer](https://github.com/INDIAPOST-gov/digipin/blob/main/docs/disclaimer.md).
