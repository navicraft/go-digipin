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
//
//	echo "28.7049,77.1038" | digipin encode --batch
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	digipin "github.com/navicraft/go-digipin"
)

// version is overridden at build time with -ldflags "-X main.version=<tag>".
var version = "dev"

const usage = `digipin - encode and decode India Post DIGIPIN codes

Usage:
  digipin encode <lat> <lng> [--level N] [--format plain|spaced] [--json] [--maps]
  digipin decode <code> [--json] [--maps]
  digipin bounds <code> [--geojson] [--json]
  digipin neighbors <code> [--json]
  digipin version

Flags:
  --batch   read newline-delimited input from stdin instead of arguments
            (encode: "lat,lng" per line; others: one code per line)
`

// exit is indirected so tests can invoke main without terminating the process.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// positionalArgs parses flags that may be interspersed with positional arguments
// (Go's flag package stops at the first non-flag token). It returns the collected
// positional arguments, or ok=false if a flag failed to parse.
func positionalArgs(fs *flag.FlagSet, args []string) (pos []string, ok bool) {
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		if fs.NArg() == 0 {
			return pos, true
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
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
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}

func cmdEncode(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("encode", flag.ContinueOnError)
	fs.SetOutput(stderr)
	level := fs.Int("level", 10, "precision level 1..10")
	format := fs.String("format", "spaced", "output format: plain|spaced")
	asJSON := fs.Bool("json", false, "output JSON")
	maps := fs.Bool("maps", false, "also print a Google Maps link")
	batch := fs.Bool("batch", false, "read lat,lng lines from stdin")
	pos, ok := positionalArgs(fs, args)
	if !ok {
		return 2
	}

	emit := func(line string) (string, error) {
		lat, lng, err := parseLatLng(line)
		if err != nil {
			return "", err
		}
		return encodeOut(lat, lng, *level, *format, *asJSON, *maps)
	}

	if *batch {
		return batchLines(stdin, stdout, stderr, emit)
	}
	if len(pos) != 2 {
		fmt.Fprintln(stderr, "encode requires <lat> <lng>")
		return 2
	}
	return single(emit(pos[0]+","+pos[1]))(stdout, stderr)
}

func cmdDecode(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("decode", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	maps := fs.Bool("maps", false, "also print a Google Maps link")
	batch := fs.Bool("batch", false, "read codes from stdin")
	pos, ok := positionalArgs(fs, args)
	if !ok {
		return 2
	}

	emit := func(code string) (string, error) {
		return decodeOut(code, *asJSON, *maps)
	}

	if *batch {
		return batchLines(stdin, stdout, stderr, emit)
	}
	if len(pos) != 1 {
		fmt.Fprintln(stderr, "decode requires <code>")
		return 2
	}
	return single(emit(pos[0]))(stdout, stderr)
}

func cmdBounds(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bounds", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	geo := fs.Bool("geojson", false, "output a GeoJSON Feature")
	batch := fs.Bool("batch", false, "read codes from stdin")
	pos, ok := positionalArgs(fs, args)
	if !ok {
		return 2
	}

	emit := func(code string) (string, error) {
		return boundsOut(code, *geo, *asJSON)
	}

	if *batch {
		return batchLines(stdin, stdout, stderr, emit)
	}
	if len(pos) != 1 {
		fmt.Fprintln(stderr, "bounds requires <code>")
		return 2
	}
	return single(emit(pos[0]))(stdout, stderr)
}

func cmdNeighbors(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("neighbors", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	batch := fs.Bool("batch", false, "read codes from stdin")
	pos, ok := positionalArgs(fs, args)
	if !ok {
		return 2
	}

	emit := func(code string) (string, error) {
		return neighborsOut(code, *asJSON)
	}

	if *batch {
		return batchLines(stdin, stdout, stderr, emit)
	}
	if len(pos) != 1 {
		fmt.Fprintln(stderr, "neighbors requires <code>")
		return 2
	}
	return single(emit(pos[0]))(stdout, stderr)
}

// single adapts a one-shot (output, error) result into a writer/exit-code action.
func single(out string, err error) func(stdout, stderr io.Writer) int {
	return func(stdout, stderr io.Writer) int {
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, out)
		return 0
	}
}

// batchLines runs emit over each non-blank line of stdin, writing results to
// stdout and errors to stderr. It returns 1 if any line failed, else 0.
func batchLines(stdin io.Reader, stdout, stderr io.Writer, emit func(string) (string, error)) int {
	scanner := bufio.NewScanner(stdin)
	exit := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		out, err := emit(line)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			exit = 1
			continue
		}
		fmt.Fprintln(stdout, out)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return exit
}

func encodeOut(lat, lng float64, level int, format string, asJSON, maps bool) (string, error) {
	d, err := digipin.EncodeLevel(lat, lng, level)
	if err != nil {
		return "", err
	}
	mapsURL := ""
	if maps {
		mapsURL = digipin.MapsLink(lat, lng)
	}
	if asJSON {
		s := `{"digipin":` + strconv.Quote(d.Canonical()) +
			`,"formatted":` + strconv.Quote(d.Formatted())
		if maps {
			s += `,"maps_url":` + strconv.Quote(mapsURL)
		}
		return s + "}", nil
	}
	var out string
	switch format {
	case "plain":
		out = d.Canonical()
	case "spaced":
		out = d.Formatted()
	default:
		return "", fmt.Errorf("invalid format %q (want plain|spaced)", format)
	}
	if maps {
		out += "\n" + mapsURL
	}
	return out, nil
}

func decodeOut(code string, asJSON, maps bool) (string, error) {
	c, err := digipin.Decode(code)
	if err != nil {
		return "", err
	}
	mapsURL := ""
	if maps {
		mapsURL = c.MapsLink()
	}
	if asJSON {
		s := `{"latitude":` + ff(c.Latitude) + `,"longitude":` + ff(c.Longitude)
		if maps {
			s += `,"maps_url":` + strconv.Quote(mapsURL)
		}
		return s + "}", nil
	}
	out := fmt.Sprintf("%.6f, %.6f", c.Latitude, c.Longitude)
	if maps {
		out += "\n" + mapsURL
	}
	return out, nil
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
		return `{"min_latitude":` + ff(b.MinLatitude) +
			`,"max_latitude":` + ff(b.MaxLatitude) +
			`,"min_longitude":` + ff(b.MinLongitude) +
			`,"max_longitude":` + ff(b.MaxLongitude) + `}`, nil
	}
	return b.String(), nil
}

func neighborsOut(code string, asJSON bool) (string, error) {
	c, err := digipin.Parse(code)
	if err != nil {
		return "", err
	}
	ns := c.Neighbors()
	codes := make([]string, len(ns))
	for i, n := range ns {
		codes[i] = n.Canonical()
	}
	if asJSON {
		quoted := make([]string, len(codes))
		for i, c := range codes {
			quoted[i] = strconv.Quote(c)
		}
		return "[" + strings.Join(quoted, ",") + "]", nil
	}
	return strings.Join(codes, " "), nil
}

// parseLatLng parses a "lat,lng" string into two float64 values.
func parseLatLng(s string) (lat, lng float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected lat,lng, got %q", s)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude %q", parts[0])
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude %q", parts[1])
	}
	return lat, lng, nil
}

func ff(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
