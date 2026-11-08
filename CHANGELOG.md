# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-11-08

Correctness, performance, and CLI improvements. No exported symbol was removed;
redundant ones are deprecated.

### Added

- Encoder tests covering points on and next to grid lines, the grid-line rule
  (a point on a line belongs to the cell north/east of it, except on the outer
  edges), and the Dak Bhawan example (`39J49LL8T4`). The fuzz test checks
  every input against an independent implementation of the algorithm.
- CLI: `--batch` keeps output aligned with input (one output line per input
  line; failed lines print an empty line), and errors name the line number.
- CLI: binaries installed with `go install …@version` report that version.
- CLI: `-h` on a command exits 0; `--` ends flag parsing; unquoted spaced codes
  (`digipin decode 39J J567 L7P`) work; a UTF-8 BOM and lines of any length
  are accepted in batch input.
- Release workflow publishing prebuilt CLI binaries via GoReleaser.

### Changed

- **Minimum Go version is now 1.24.** The library and CLI use range-over-int
  (Go 1.22); the tests use `testing.B.Loop` (Go 1.24).
- `ValidateCode` is stricter: it accepts only the continuous form or 3-4-3
  spacing, and rejects hyphens. It ignores letter case and surrounding
  whitespace (tabs and line endings included). `Parse` and `Decode` remain
  lenient, and now also ignore surrounding whitespace.
- `ValidateCoordinates` now checks the DIGIPIN coverage area, so it reports
  exactly whether `Encode` would accept the coordinates. Previously it only
  checked the global latitude/longitude range.
- Coordinate errors name the offending axis and value, e.g.
  `latitude 50 out of range [2.5, 38.5]`.
- `Coordinate.Equal` is now exact (it previously compared within 1e-6, which
  made go-cmp comparisons silently approximate).
- Methods on an invalid `Code` no longer report a misleading location:
  `Coordinate` and `Bounds` return zero values, `Neighbors` returns none,
  `DistanceTo` (and the deprecated `Distance`) return NaN, `MapsLink` returns
  "", and `GeoJSON` returns a null geometry with empty properties.
- `Code.MarshalText` returns `ErrInvalidCode` for an invalid non-empty `Code`
  rather than writing a value that cannot be read back.
- `Bounds.GeoJSON` emits a `null` geometry for bounds that are empty or not
  finite, so GeoJSON output is always valid JSON.
- Decoding is about 10x faster and no longer allocates for canonical input.
- CLI:
  - `encode` prints the continuous form by default; use `--format spaced` for
    3-4-3 display. It also accepts the pair as one argument (`"28.7,77.1"`)
    and rejects malformed pairs such as `28.7,,77.1`.
  - `decode --json` includes the `digipin` and returns `latitude`/`longitude`
    as 6-decimal strings (e.g. `"28.704882"`).
  - Invalid `--format` and `--level` values are a usage error (exit 2)
    reported once, even with `--json` or `--batch`.
  - `--maps` output is tab-separated on the same line.
  - `-h` prints to stdout, and all error messages start with `error:`.
  - A failed write to stdout (closed pipe, full disk) is reported and exits 1
    instead of silently losing output.
- CI tests Go 1.24 and the latest stable release, uses golangci-lint v2, and
  runs with read-only permissions.

### Deprecated

- `Distance`: use `Code.DistanceTo`.
- `MapsLink`: use `Coordinate.MapsLink`.
- `Coordinate.Equal`: use `==` or `Coordinate.WithinTolerance`.
- `Bounds.Area`: square degrees are not a meaningful unit of area.

### Fixed

- The zero `Code` now round-trips through JSON (`""` previously failed to
  unmarshal).
- CLI `--json` output no longer escapes `&` in URLs as `\u0026`.
- Package documentation said each character narrows the cell to a quarter; it
  is a sixteenth (a 4×4 sub-grid).

## [1.0.0] - 2026-06-11

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

[1.1.0]: https://github.com/navicraft/go-digipin/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/navicraft/go-digipin/releases/tag/v1.0.0
