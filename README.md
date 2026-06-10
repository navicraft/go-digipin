<p align="center">
  <img src="docs/digipin-logo-blue.png" alt="DIGIPIN" width="96">
</p>

# go-digipin

[![Go Reference](https://pkg.go.dev/badge/github.com/navicraft/go-digipin.svg)](https://pkg.go.dev/github.com/navicraft/go-digipin)
[![Go Report Card](https://goreportcard.com/badge/github.com/navicraft/go-digipin)](https://goreportcard.com/report/github.com/navicraft/go-digipin)
[![CI](https://github.com/navicraft/go-digipin/actions/workflows/ci.yml/badge.svg)](https://github.com/navicraft/go-digipin/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/navicraft/go-digipin)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**DIGIPIN encoder and decoder for Go**. Convert any latitude/longitude in India to
a short DIGIPIN geocode and back, offline, with zero dependencies.

DIGIPIN (Digital Postal Index Number) is India Post's grid-based digital addressing
system. It maps a coordinate to a 10-character alphanumeric code using a hierarchical
4×4 grid, so nearby places share code prefixes, like a geohash for India.

```go
d, _ := digipin.Encode(28.70486848, 77.10376044)
fmt.Println(d.Canonical()) // 39JJ567L7P
fmt.Println(d.Formatted()) // 39J J567 L7P

c, _ := digipin.Decode("39JJ567L7P")
fmt.Printf("%.6f, %.6f\n", c.Latitude, c.Longitude) // 28.704882, 77.103769
```

## Features

- **Encode / decode** coordinates ⟷ DIGIPIN, following the DIGIPIN specification.
- **Both code forms**: canonical continuous (`39JJ567L7P`, for storage and interchange)
  and 3-4-3 display (`39J J567 L7P`).
- **Lenient decoding**: accepts continuous, spaced, and hyphenated input, any case.
- **Variable precision**: encode to any level 1–10 (`EncodeLevel`).
- **Spatial helpers**: bounding boxes, adjacent-cell neighbors, parent cells,
  haversine distance, GeoJSON cell polygons, and Google Maps links.
- **Zero dependencies**, pure Go, allocation-light, goroutine-safe.
- **100% test coverage**.

## Install

```sh
go get github.com/navicraft/go-digipin
```

```go
import digipin "github.com/navicraft/go-digipin"
```

## Coverage area & code format

DIGIPIN covers India's bounding box: latitude **2.5–38.5**, longitude **63.5–99.5**.
Codes use the 16 unambiguous characters `2 3 4 5 6 7 8 9 C F J K L M P T` (no 0, 1,
I, O). The canonical form is continuous; spaces in a 3-4-3 group are display-only.

## Usage

### Encode / decode

`Encode` returns a `Code`, a defined string type holding the canonical form.

```go
c, err := digipin.Encode(12.9716, 77.5946) // c is a digipin.Code value
if err != nil { /* out of range / not finite */ }

c.Canonical() // "4P3JK852C9"
c.Formatted() // "4P3 JK85 2C9"
c.Level()     // 10
c == "4P3JK852C9" // codes compare directly

// Decode accepts any form transparently.
digipin.Decode("4P3JK852C9")
digipin.Decode("4P3 JK85 2C9")
digipin.Decode("4P3-JK8-52C9")
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
m := digipin.Distance(a, z) // meters
```

### GeoJSON and Maps links

```go
gj := c.GeoJSON()           // GeoJSON Feature (cell polygon + "digipin" property)
link := digipin.MapsLink(12.9716, 77.5946)
// https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946
```

### JSON

`Code` marshals to and from its canonical JSON string:

```go
data, _ := json.Marshal(c)       // "4P3JK852C9"
var x digipin.Code
_ = json.Unmarshal([]byte(`"4P3 JK85 2C9"`), &x) // x == "4P3JK852C9"
```

## Command-line tool

```sh
go install github.com/navicraft/go-digipin/cmd/digipin@latest
```

```sh
digipin encode 28.70486848 77.10376044    # 39J J567 L7P
digipin encode 28.70486848 77.10376044 --format plain --maps
digipin decode 39JJ567L7P --json
digipin bounds 39JJ567L7P --geojson
digipin neighbors 39JJ567L7P

# Batch mode reads stdin (lat,lng per line for encode; codes for the rest):
printf '28.7049,77.1038\n12.9716,77.5946\n' | digipin encode --batch
```

## API overview

| Symbol | Description |
| --- | --- |
| `Encode`, `EncodeLevel` | Coordinates → `Code` (full or level 1–10) |
| `Decode` | DIGIPIN string → center `Coordinate` |
| `Parse` | Validate any form → `Code` (canonical) |
| `ValidateCoordinates`, `ValidateCode` | Input validation without encoding/decoding |
| `Distance`, `Coordinate.DistanceTo` | Great-circle distance in meters |
| `MapsLink`, `Coordinate.MapsLink`, `Code.MapsLink` | Google Maps URLs |
| `Code.Canonical` / `.Formatted` / `.Level` / `.Valid` / `.Parent` / `.Neighbors` / `.Coordinate` / `.Bounds` / `.GeoJSON` | Code helpers |
| `Bounds.Center` / `.Corners` / `.Width` / `.Height` / `.Contains` / `.GeoJSON` | Bounding-box helpers |

Full documentation: **[pkg.go.dev/github.com/navicraft/go-digipin](https://pkg.go.dev/github.com/navicraft/go-digipin)**.

## License

[MIT](LICENSE).
