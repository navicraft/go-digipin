# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Unreleased

First stable release.

This release adopts an idiomatic, value-based API. The location type is `Code`
(a defined string type), so codes compare with `==` and work as map keys; all
types use value semantics rather than pointers.

### Added

- `Code` type (defined string) with methods `Canonical`, `Formatted`, `String`,
  `Level`, `Valid`, `Coordinate`, `Bounds`, `Parent`, `Neighbors`, `DistanceTo`,
  `MapsLink`, `GeoJSON`, and `MarshalText`/`UnmarshalText`.
- `EncodeLevel` for variable-precision encoding (levels 1–10).
- `Parse` to build a validated `Code` from any accepted form.
- `Distance` and `Coordinate.DistanceTo` (haversine, meters).
- `Bounds.Center`, `Bounds.Corners`, `Bounds.Width`, `Bounds.Height`, `Bounds.GeoJSON`.
- `Coordinate.WithinTolerance`, `Coordinate.MapsLink`.
- `MapsLink` (Google Maps URLs) and `ValidateCode`.
- A `digipin` command-line tool (`cmd/digipin`) with encode/decode/bounds/neighbors,
  `--json`, `--maps`, `--geojson`, and `--batch` (stdin) modes.
- Native Go fuzz tests and runnable example tests.

### Changed

- `Encode` returns a `Code` value holding the canonical continuous code (no
  separators); use `Formatted` for the 3-4-3 display string. Pointers (`*DIGIPIN`,
  `*Coordinate`, `*Bounds`) are gone in favor of value semantics.
- `Decode` accepts continuous, spaced, and hyphenated input, any case, and partial
  (1–9 character) codes, returning a `Coordinate` value.
- Errors renamed: `ErrInvalidCode` (was `ErrInvalidDIGIPIN`); added `ErrInvalidLevel`.
- Minimum Go version lowered to 1.21.
- Test coverage raised to 100% of statements across all packages.

[1.0.0]: https://github.com/navicraft/go-digipin/releases/tag/v1.0.0
