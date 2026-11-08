// Command digipin encodes and decodes India Post DIGIPIN codes from the terminal.
//
// Usage:
//
//	digipin encode <lat> <lng> [--level N] [--format plain|spaced] [--json] [--maps]
//	digipin decode <code> [--json] [--maps]
//	digipin bounds <code> [--geojson] [--json]
//	digipin neighbors <code> [--json]
//	digipin version
//
// With --batch, input is read newline-delimited from stdin instead of arguments:
// "lat,lng" per line for encode, one code per line for decode/bounds/neighbors.
// Output line N always corresponds to input line N; a line that fails prints
// an empty line, and its error goes to stderr.
//
//	echo "28.7049,77.1038" | digipin encode --batch
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	digipin "github.com/navicraft/go-digipin"
)

// version is overridden at build time with -ldflags "-X main.version=<tag>".
// Binaries built by "go install module@version" report the module version.
var version = "dev"

const usage = `digipin - encode and decode India Post DIGIPIN codes

Usage:
  digipin encode <lat> <lng> [--level N] [--format plain|spaced] [--json] [--maps]
  digipin decode <code> [--json] [--maps]
  digipin bounds <code> [--geojson] [--json]
  digipin neighbors <code> [--json]
  digipin version

Codes may be continuous (39JJ567L7P) or spaced (39J J567 L7P, quoted or not).
Every command prints one line per result; --maps appends the URL after a tab.

Flags:
  --batch   read newline-delimited input from stdin instead of arguments
            (encode: "lat,lng" per line; others: one code per line). Output
            line N matches input line N; failed lines print an empty line.
  -h        show a command's flags

Exit status: 0 success, 1 a conversion failed, 2 usage error.
`

// exit and readBuildInfo are indirected so tests can exercise main and the
// version lookup without terminating the process or rebuilding the binary.
var (
	exit          = os.Exit
	readBuildInfo = debug.ReadBuildInfo
)

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "encode":
		return cmdEncode(args[1:], stdin, stdout, stderr)
	case "decode":
		return cmdDecode(args[1:], stdin, stdout, stderr)
	case "bounds":
		return cmdBounds(args[1:], stdin, stdout, stderr)
	case "neighbors":
		return cmdNeighbors(args[1:], stdin, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, versionString())
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}

// versionString returns the -ldflags version if set, else the module version
// recorded by "go install module@version", else "dev".
func versionString() string {
	if version != "dev" {
		return version
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func cmdEncode(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("encode")
	level := fs.Int("level", 10, "precision level 1..10")
	format := fs.String("format", "plain", "text output: plain (39JJ567L7P) or spaced (39J J567 L7P)")
	asJSON := fs.Bool("json", false, "output JSON")
	maps := fs.Bool("maps", false, "append a Google Maps link")
	check := func() error {
		if *level < 1 || *level > 10 {
			return fmt.Errorf("invalid --level %d (want 1..10)", *level)
		}
		if *format != "plain" && *format != "spaced" {
			return fmt.Errorf("invalid --format %q (want plain|spaced)", *format)
		}
		return nil
	}
	return execute(fs, args, 2, "<lat> <lng>", check, func(in string) (string, error) {
		lat, lng, err := parseLatLng(in)
		if err != nil {
			return "", err
		}
		return encodeOut(lat, lng, *level, *format == "spaced", *asJSON, *maps)
	}, stdin, stdout, stderr)
}

func cmdDecode(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("decode")
	asJSON := fs.Bool("json", false, "output JSON")
	maps := fs.Bool("maps", false, "append a Google Maps link")
	return execute(fs, args, 0, "<code>", nil, func(code string) (string, error) {
		return decodeOut(code, *asJSON, *maps)
	}, stdin, stdout, stderr)
}

func cmdBounds(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("bounds")
	asJSON := fs.Bool("json", false, "output JSON")
	geo := fs.Bool("geojson", false, "output a GeoJSON Feature")
	return execute(fs, args, 0, "<code>", nil, func(code string) (string, error) {
		return boundsOut(code, *geo, *asJSON)
	}, stdin, stdout, stderr)
}

func cmdNeighbors(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("neighbors")
	asJSON := fs.Bool("json", false, "output JSON")
	return execute(fs, args, 0, "<code>", nil, func(code string) (string, error) {
		return neighborsOut(code, *asJSON)
	}, stdin, stdout, stderr)
}

// newFlagSet returns a flag set that reports nothing itself; execute prints
// errors and help so it can choose the stream and exit status.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// execute adds --batch to fs, parses args, and validates flag values with
// check (if non-nil). It then runs emit once on the positional arguments joined
// by spaces, or, with --batch, once per stdin line. maxArgs caps the number of
// positional arguments (0 means no limit); at least one is always required.
func execute(fs *flag.FlagSet, args []string, maxArgs int, argsUsage string, check func() error,
	emit func(string) (string, error), stdin io.Reader, stdout, stderr io.Writer) int {
	batch := fs.Bool("batch", false, "read newline-delimited input from stdin")
	pos, err := positionalArgs(fs, args)
	if err != nil {
		// -h prints the flags to stdout and succeeds; a bad flag is a usage error.
		w, status := stdout, 0
		if !errors.Is(err, flag.ErrHelp) {
			w, status = stderr, 2
			fmt.Fprintln(stderr, "error:", err)
		}
		fmt.Fprintf(w, "Usage: digipin %s %s [flags]\n", fs.Name(), argsUsage)
		fs.SetOutput(w)
		fs.PrintDefaults()
		return status
	}
	if check != nil {
		if err := check(); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
	}
	if *batch {
		if len(pos) > 0 {
			fmt.Fprintf(stderr, "error: %s --batch reads stdin; unexpected argument %q\n", fs.Name(), pos[0])
			return 2
		}
		return batchLines(stdin, stdout, stderr, emit)
	}
	if len(pos) == 0 || (maxArgs > 0 && len(pos) > maxArgs) {
		fmt.Fprintf(stderr, "error: %s requires %s\n", fs.Name(), argsUsage)
		return 2
	}
	out, err := emit(strings.Join(pos, " "))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return writeLine(stdout, stderr, out)
}

// writeLine prints out on stdout. A failed write (closed pipe, full disk) is
// reported and turns into exit status 1, so lost output is never silent.
func writeLine(stdout, stderr io.Writer, out string) int {
	if _, err := fmt.Fprintln(stdout, out); err != nil {
		fmt.Fprintln(stderr, "error: writing output:", err)
		return 1
	}
	return 0
}

// positionalArgs parses flags that may be interspersed with positional arguments
// (Go's flag package stops at the first non-flag token) and returns the
// positional arguments. Everything after a "--" is positional.
//
// The "--" check looks at the token just before the unparsed rest, so a string
// flag whose value is literally "--" (e.g. "--format --") would also end flag
// parsing. That is harmless today because every string flag rejects "--".
func positionalArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// batchLines runs emit over each line of stdin and writes exactly one stdout
// line per input line, so output stays aligned with input. Blank lines produce
// blank lines; a failing line produces a blank line and an error on stderr
// naming the line number. It returns 1 if any line failed, else 0.
// It stops with status 1 if stdout cannot be written.
func batchLines(stdin io.Reader, stdout, stderr io.Writer, emit func(string) (string, error)) int {
	r := bufio.NewReader(stdin) // unlike bufio.Scanner, no line-length limit
	status := 0
	for n := 1; ; n++ {
		raw, readErr := r.ReadString('\n')
		if raw != "" {
			line := strings.TrimSpace(strings.TrimPrefix(raw, "\uFEFF")) // tolerate a UTF-8 BOM
			out := ""
			if line != "" {
				var err error
				if out, err = emit(line); err != nil {
					fmt.Fprintf(stderr, "error: line %d: %v\n", n, err)
					status = 1
				}
			}
			if writeLine(stdout, stderr, out) != 0 {
				return 1
			}
		}
		if readErr == io.EOF {
			return status
		}
		if readErr != nil {
			fmt.Fprintln(stderr, "error:", readErr)
			return 1
		}
	}
}

func encodeOut(lat, lng float64, level int, spaced, asJSON, maps bool) (string, error) {
	c, err := digipin.EncodeLevel(lat, lng, level)
	if err != nil {
		return "", err
	}
	mapsURL := ""
	if maps {
		mapsURL = digipin.Coordinate{Latitude: lat, Longitude: lng}.MapsLink()
	}
	if asJSON {
		return toJSON(struct {
			Digipin   digipin.Code `json:"digipin"`
			Formatted string       `json:"formatted"`
			MapsURL   string       `json:"maps_url,omitempty"`
		}{c, c.Formatted(), mapsURL}), nil
	}
	out := c.Canonical()
	if spaced {
		out = c.Formatted()
	}
	return withMaps(out, mapsURL), nil
}

func decodeOut(code string, asJSON, maps bool) (string, error) {
	c, err := digipin.Parse(code)
	if err != nil {
		return "", err
	}
	coord := c.Coordinate()
	mapsURL := ""
	if maps {
		mapsURL = coord.MapsLink()
	}
	lat, lng := fmt.Sprintf("%.6f", coord.Latitude), fmt.Sprintf("%.6f", coord.Longitude)
	if asJSON {
		// Coordinates are 6-decimal strings, matching the DecodeResponse of
		// the official DIGIPIN API (github.com/INDIAPOST-gov/digipin).
		return toJSON(struct {
			Digipin   digipin.Code `json:"digipin"`
			Latitude  string       `json:"latitude"`
			Longitude string       `json:"longitude"`
			MapsURL   string       `json:"maps_url,omitempty"`
		}{c, lat, lng, mapsURL}), nil
	}
	return withMaps(lat+", "+lng, mapsURL), nil
}

func boundsOut(code string, geo, asJSON bool) (string, error) {
	c, err := digipin.Parse(code)
	if err != nil {
		return "", err
	}
	b := c.Bounds()
	if geo {
		return string(b.GeoJSON()), nil
	}
	if asJSON {
		return toJSON(b), nil
	}
	return b.String(), nil
}

func neighborsOut(code string, asJSON bool) (string, error) {
	c, err := digipin.Parse(code)
	if err != nil {
		return "", err
	}
	ns := c.Neighbors()
	if asJSON {
		return toJSON(ns), nil
	}
	codes := make([]string, len(ns))
	for i, n := range ns {
		codes[i] = n.Canonical()
	}
	return strings.Join(codes, " "), nil
}

// withMaps appends a tab and the maps URL to out, if there is one.
func withMaps(out, mapsURL string) string {
	if mapsURL == "" {
		return out
	}
	return out + "\t" + mapsURL
}

// toJSON renders v as compact JSON without HTML escaping, so URLs keep a
// literal "&". The values passed here always marshal.
func toJSON(v any) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}

// parseLatLng parses "lat,lng" (spaces around the comma allowed) or "lat lng"
// into two float64 values.
func parseLatLng(s string) (lat, lng float64, err error) {
	parts := strings.Fields(s)
	if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	}
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected lat,lng, got %q", strings.TrimSpace(s))
	}
	parts[0], parts[1] = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	lat, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude %q", parts[0])
	}
	lng, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude %q", parts[1])
	}
	return lat, lng, nil
}
