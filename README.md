# DIGIPIN - Digital Addressing System in India

[![Go Reference](https://pkg.go.dev/badge/github.com/navicraft/go-digipin.svg)](https://pkg.go.dev/github.com/navicraft/go-digipin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance, pure Go implementation of the DIGIPIN geocoding system developed by India Post in collaboration with IIT Hyderabad and NRSC, ISRO. DIGIPIN converts geographic coordinates to compact, human-readable 10-character alphanumeric codes using a hierarchical 4×4 grid system optimized for the Indian subcontinent.

## Features

- **Zero Dependencies**: Pure Go implementation
- **High Performance**: ~101ns encode, ~232ns decode
- **Thread-Safe**: Safe for concurrent use
- **Comprehensive**: Encoding, decoding, validation, bounds, equality checks
- **Well-Tested**: 100% test coverage with edge cases

## Installation

```bash
go get github.com/navicraft/go-digipin
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/navicraft/go-digipin"
)

func main() {
    // Encode coordinates to DIGIPIN
    digipin, err := digipin.Encode(12.9716, 77.5946) // Bangalore
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Encoded: %s\n", digipin.Code) // 4P3-JK8-52C9

    // Decode back to coordinates
    coord, err := digipin.Decode(digipin.Code)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Decoded: %s\n", coord.String()) // (12.971601, 77.594584)
}
```

## Coordinate Bounds

DIGIPIN is optimized for the Indian subcontinent and supports coordinates within:

- **Latitude**: 2.5° to 38.5° North
- **Longitude**: 63.5° to 99.5° East

This covers the entire Indian territory including maritime zones. Coordinates outside these bounds will return an error.

## Code Format

DIGIPIN codes are 10-character alphanumeric strings in the format `XXX-XXX-XXXX`, using only these characters:

- **Digits**: 2, 3, 4, 5, 6, 7, 8, 9
- **Letters**: C, F, J, K, L, M, P, T

This character set avoids ambiguous characters (0, 1, I, O) for better readability and reduces confusion in manual transcription.

## API Reference

### Core Functions

```go
// Encode coordinates to DIGIPIN
func Encode(latitude, longitude float64) (*DIGIPIN, error)

// Decode DIGIPIN to coordinates
func Decode(digipinCode string) (*Coordinate, error)

// Validate coordinates without encoding
func ValidateCoordinates(latitude, longitude float64) error

// Validate DIGIPIN format without decoding
func ValidateDIGIPIN(digipinCode string) error

// Get bounding box for a DIGIPIN
func GetBounds(digipinCode string) (*Bounds, error)
```

### Types

#### `Coordinate`

```go
type Coordinate struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

// Methods
func (c *Coordinate) String() string
func (c *Coordinate) Equal(other *Coordinate) bool
func (c *Coordinate) EqualWithTolerance(other *Coordinate, tolerance float64) bool
func (c *Coordinate) BelongsTo(digipin *DIGIPIN, tolerance float64) (bool, error)
```

#### `DIGIPIN`

```go
type DIGIPIN struct {
    Code string `json:"digipin"`
}

// Methods
func (d *DIGIPIN) String() string
func (d *DIGIPIN) Equal(other *DIGIPIN) (bool, error)
func (d *DIGIPIN) EqualWithTolerance(other *DIGIPIN, tolerance float64) (bool, error)
func (d *DIGIPIN) BelongsTo(coord *Coordinate, tolerance float64) (bool, error)
func (d *DIGIPIN) GetBounds() (*Bounds, error)
```

#### `Bounds`

```go
type Bounds struct {
    MinLatitude  float64 `json:"min_latitude"`
    MaxLatitude  float64 `json:"max_latitude"`
    MinLongitude float64 `json:"min_longitude"`
    MaxLongitude float64 `json:"max_longitude"`
}

// Methods
func (b *Bounds) String() string
func (b *Bounds) Area() float64
func (b *Bounds) Contains(coord *Coordinate) bool
```

## Usage Examples

### Coordinate Validation

```go
err := digipin.ValidateCoordinates(12.9716, 77.5946)
if err != nil {
    fmt.Println("Invalid coordinates:", err)
}
```

### Tolerance-Based Comparison

```go
coord1 := &digipin.Coordinate{Latitude: 12.9716, Longitude: 77.5946}
coord2 := &digipin.Coordinate{Latitude: 12.9717, Longitude: 77.5947}

// Check if coordinates belong to same DIGIPIN area
digipin1, _ := digipin.Encode(12.9716, 77.5946)
belongs, err := coord2.BelongsTo(digipin1, 1e-3) // custom tolerance

// Compare DIGIPIN codes
digipin2, _ := digipin.Encode(12.9717, 77.5947)
equal, err := digipin1.EqualWithTolerance(digipin2, 1e-3)
```

### Bounding Box Operations

```go
// Get area covered by a DIGIPIN
bounds, err := digipin.GetBounds("4P3-JK8-52C9")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Area: %s\n", bounds.String())
fmt.Printf("Size: %.8f sq degrees\n", bounds.Area())

// Check if coordinate is within bounds
coord := &digipin.Coordinate{Latitude: 12.9716, Longitude: 77.5946}
if bounds.Contains(coord) {
    fmt.Println("Coordinate is within DIGIPIN area")
}
```

## Constants & Errors

- `DefaultTolerance = 1e-4` - Default tolerance (~11.1 meters at equator)
- `ErrInvalidCoordinate` - Coordinates out of valid range
- `ErrInvalidDIGIPIN` - Malformed DIGIPIN code

**Valid Ranges:**

- Latitude: 2.5 to 38.5 degrees North (DIGIPIN bounds)
- Longitude: 63.5 to 99.5 degrees East (DIGIPIN bounds)
- Format: XXX-XXX-XXXX (using characters: 2-9, C, F, J, K, L, M, P, T)

## Performance

Benchmarks on Apple M3 Pro:

```
BenchmarkEncode-11      11014057    101.6 ns/op    80 B/op    5 allocs/op
BenchmarkDecode-11       5034634    232.5 ns/op    96 B/op    4 allocs/op
BenchmarkRoundTrip-11    3394249    346.2 ns/op   176 B/op    9 allocs/op
```

## About DIGIPIN

DIGIPIN is a geocoding system developed by India Post in collaboration with IIT Hyderabad and NRSC, ISRO. It provides:

- **Compact Encoding**: 10-character codes vs lengthy coordinates
- **Spatial Locality**: Similar locations have similar codes
- **Standardized**: National addressing standard for India
- **Reversible**: Lossless encoding and decoding

## Testing

```bash
go test -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/name`)
3. Add tests for new functionality
4. Ensure 100% test coverage
5. Submit pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Algorithm Compliance

This implementation follows the uses a hierarchical 4×4 grid system. It matches the exact algorithm used by India Post and produces identical DIGIPIN codes for the same coordinates.

## Special mentions

- Department of Posts, Government of India
- Indian Institute of Technology, Hyderabad
- National Remote Sensing Centre, ISRO
