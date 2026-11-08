package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"runtime/debug"
	"strings"
	"testing"
)

func TestMainWrapper(t *testing.T) {
	oldArgs, oldExit := os.Args, exit
	defer func() { os.Args, exit = oldArgs, oldExit }()

	var code int
	exit = func(c int) { code = c }
	os.Args = []string{"digipin", "version"}
	main()
	if code != 0 {
		t.Errorf("main version exit = %d, want 0", code)
	}
}

func capRun(t *testing.T, args []string, stdin string) (out, errOut string, code int) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, strings.NewReader(stdin), &o, &e)
	return o.String(), e.String(), code
}

func TestRunDispatch(t *testing.T) {
	if _, _, code := capRun(t, nil, ""); code != 2 {
		t.Errorf("no args: code = %d, want 2", code)
	}
	if out, _, code := capRun(t, []string{"--version"}, ""); code != 0 || strings.TrimSpace(out) == "" {
		t.Errorf("--version: out=%q code=%d", out, code)
	}
	if out, _, code := capRun(t, []string{"help"}, ""); code != 0 || !strings.Contains(out, "Usage:") {
		t.Errorf("help: out=%q code=%d", out, code)
	}
	if _, errOut, code := capRun(t, []string{"bogus"}, ""); code != 2 || !strings.HasPrefix(errOut, "error: unknown command \"bogus\"") {
		t.Errorf("unknown: errOut=%q code=%d", errOut, code)
	}
}

func TestVersionString(t *testing.T) {
	oldVersion, oldRead := version, readBuildInfo
	defer func() { version, readBuildInfo = oldVersion, oldRead }()

	info := func(v string, ok bool) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, ok
		}
	}
	for _, tt := range []struct {
		ldflags string
		read    func() (*debug.BuildInfo, bool)
		want    string
	}{
		{"v9.9.9", info("v1.0.0", true), "v9.9.9"}, // -ldflags wins
		{"dev", info("v1.1.0", true), "v1.1.0"},    // go install module@version
		{"dev", info("(devel)", true), "dev"},      // local build
		{"dev", info("", true), "dev"},
		{"dev", info("v1.1.0", false), "dev"}, // no build info
	} {
		version, readBuildInfo = tt.ldflags, tt.read
		if got := versionString(); got != tt.want {
			t.Errorf("versionString() with ldflags %q = %q, want %q", tt.ldflags, got, tt.want)
		}
	}
}

func TestEncodeCommand(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		code    int
		out     string // exact stdout (trimmed), if set
		contain string // substring of stdout or stderr
	}{
		{"default is continuous", []string{"encode", "28.70486848", "77.10376044"}, 0, "39JJ567L7P", ""},
		{"spaced", []string{"encode", "28.70486848", "77.10376044", "--format", "spaced"}, 0, "39J J567 L7P", ""},
		{"level", []string{"encode", "12.9716", "77.5946", "--level", "3"}, 0, "4P3", ""},
		{"maps on one line", []string{"encode", "12.9716", "77.5946", "--maps"}, 0,
			"4P3JK852C9\thttps://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946", ""},
		{"json + maps keeps &", []string{"encode", "28.70486848", "77.10376044", "--json", "--maps"}, 0, "",
			`"digipin":"39JJ567L7P","formatted":"39J J567 L7P","maps_url":"https://www.google.com/maps/search/?api=1&query=`},
		{"-- ends flags", []string{"encode", "--level", "2", "--", "12.9716", "77.5946"}, 0, "4P", ""},
		{"bad format", []string{"encode", "12.9716", "77.5946", "--format", "xml"}, 2, "", "invalid --format"},
		{"bad format with json", []string{"encode", "12.9716", "77.5946", "--json", "--format", "xml"}, 2, "", "invalid --format"},
		{"bad level", []string{"encode", "12.9716", "77.5946", "--level", "0"}, 2, "", "invalid --level"},
		{"outside India", []string{"encode", "50", "77"}, 1, "", "latitude 50 out of range"},
		{"bad number", []string{"encode", "x", "77"}, 1, "", "invalid latitude"},
		{"quoted pair", []string{"encode", "12.9716, 77.5946", "--level", "3"}, 0, "4P3", ""},
		{"one number", []string{"encode", "12.9716"}, 1, "", `error: expected lat,lng, got "12.9716"`},
		{"empty field", []string{"encode", "28.7,,77.1"}, 1, "", "error: expected lat,lng"},
		{"three args", []string{"encode", "1", "2", "3"}, 2, "", "error: encode requires <lat> <lng>"},
		{"no args", []string{"encode"}, 2, "", "error: encode requires <lat> <lng>"},
		{"unknown flag", []string{"encode", "--nope"}, 2, "", "error: flag provided but not defined"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := capRun(t, tt.args, "")
			if code != tt.code {
				t.Errorf("code = %d, want %d (stderr %q)", code, tt.code, errOut)
			}
			if tt.out != "" && strings.TrimSpace(out) != tt.out {
				t.Errorf("out = %q, want %q", out, tt.out)
			}
			if !strings.Contains(out+errOut, tt.contain) {
				t.Errorf("output %q / %q missing %q", out, errOut, tt.contain)
			}
		})
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	out, errOut, code := capRun(t, []string{"encode", "-h"}, "")
	if code != 0 || errOut != "" || !strings.Contains(out, "Usage: digipin encode") || !strings.Contains(out, "-level") {
		t.Errorf("encode -h: out=%q errOut=%q code=%d", out, errOut, code)
	}
	// A bad flag shows the same help, but on stderr.
	out, errOut, code = capRun(t, []string{"decode", "--nope"}, "")
	if code != 2 || out != "" || !strings.Contains(errOut, "Usage: digipin decode") {
		t.Errorf("decode --nope: out=%q errOut=%q code=%d", out, errOut, code)
	}
}

// failWriter fails every write, like stdout closed or a full disk.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestWriteFailureIsAnError(t *testing.T) {
	for _, args := range [][]string{{"decode", "39JJ567L7P"}, {"decode", "--batch"}} {
		var e bytes.Buffer
		code := run(args, strings.NewReader("39JJ567L7P\n39JJ567L7P\n"), failWriter{}, &e)
		if code != 1 || strings.Count(e.String(), "disk full") != 1 {
			t.Errorf("%v: code=%d stderr=%q; want 1 and one write error", args, code, e.String())
		}
	}
}

func TestBatchEdgeInput(t *testing.T) {
	for _, tt := range []struct{ name, in, out string }{
		{"empty stdin", "", ""},
		{"blank lines only", "\n\n", "\n\n"},
		{"CRLF", "39JJ567L7P\r\n4\r\n", "28.704882, 77.103769\n16.000000, 77.000000\n"},
	} {
		out, errOut, code := capRun(t, []string{"decode", "--batch"}, tt.in)
		if code != 0 || out != tt.out || errOut != "" {
			t.Errorf("%s: out=%q errOut=%q code=%d, want out=%q", tt.name, out, errOut, code, tt.out)
		}
	}
}

func TestEncodeBatch(t *testing.T) {
	stdin := "\uFEFF28.70486848,77.10376044\n" + // BOM on the first line is ignored
		"\n" + // blank -> blank
		"200,77\n" + // out of bounds
		"nocomma\n" + // field count
		"x,77\n" + // bad lat
		"12,y\n" + // bad lng
		"12.9716 77.5946" // space-separated, no trailing newline
	out, errOut, code := capRun(t, []string{"encode", "--batch"}, stdin)
	if code != 1 {
		t.Errorf("batch code = %d, want 1 (had errors)", code)
	}
	if want := "39JJ567L7P\n\n\n\n\n\n4P3JK852C9\n"; out != want {
		t.Errorf("batch out = %q, want %q (one line per input line)", out, want)
	}
	for _, frag := range []string{"line 3: digipin: invalid coordinate", "line 4: expected lat,lng",
		"line 5: invalid latitude", "line 6: invalid longitude"} {
		if !strings.Contains(errOut, frag) {
			t.Errorf("batch errOut missing %q: %q", frag, errOut)
		}
	}
}

func TestBatchFlagErrorsAreUpfront(t *testing.T) {
	out, errOut, code := capRun(t, []string{"encode", "--batch", "--format", "xml"}, "12.9,77.5\n13,77\n")
	if code != 2 || out != "" || strings.Count(errOut, "invalid --format") != 1 {
		t.Errorf("out=%q errOut=%q code=%d; want one usage error, exit 2", out, errOut, code)
	}
	if _, errOut, code := capRun(t, []string{"decode", "--batch", "39JJ567L7P"}, ""); code != 2 || !strings.HasPrefix(errOut, "error: decode --batch reads stdin") {
		t.Errorf("batch + positional: errOut=%q code=%d", errOut, code)
	}
}

func TestBatchLongLine(t *testing.T) {
	// bufio.Scanner would abort the run at 64 KiB; every line must still be processed.
	stdin := strings.Repeat("F", 100_000) + "\n39JJ567L7P\n"
	out, errOut, code := capRun(t, []string{"decode", "--batch"}, stdin)
	if code != 1 || !strings.HasSuffix(out, "28.704882, 77.103769\n") || !strings.Contains(errOut, "line 1:") {
		t.Errorf("out=%q errOut=%.80q code=%d", out, errOut, code)
	}
}

// errReader yields one line, then a read error, exercising the read-error path.
type errReader struct{ done bool }

func (r *errReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, errors.New("simulated read failure")
	}
	r.done = true
	n := copy(p, "39JJ567L7P\n")
	return n, nil
}

func TestBatchReadError(t *testing.T) {
	var o, e bytes.Buffer
	code := run([]string{"decode", "--batch"}, &errReader{}, &o, &e)
	if code != 1 {
		t.Errorf("batch read-error code = %d, want 1", code)
	}
	if !strings.Contains(o.String(), "28.704882") {
		t.Errorf("line before the error not processed: %q", o.String())
	}
	if !strings.Contains(e.String(), "simulated read failure") {
		t.Errorf("missing read error in stderr: %q", e.String())
	}
}

func TestDecodeCommand(t *testing.T) {
	if out, _, code := capRun(t, []string{"decode", "39JJ567L7P"}, ""); code != 0 || strings.TrimSpace(out) != "28.704882, 77.103769" {
		t.Errorf("decode: out=%q code=%d", out, code)
	}
	// An unquoted spaced code arrives as three arguments.
	if out, _, code := capRun(t, []string{"decode", "39J", "J567", "L7P"}, ""); code != 0 || !strings.Contains(out, "28.704882") {
		t.Errorf("decode unquoted: out=%q code=%d", out, code)
	}
	out, _, _ := capRun(t, []string{"decode", "39JJ567L7P", "--json", "--maps"}, "")
	var got struct {
		Digipin   string `json:"digipin"`
		Latitude  string `json:"latitude"` // strings, like the official API
		Longitude string `json:"longitude"`
		MapsURL   string `json:"maps_url"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Digipin != "39JJ567L7P" ||
		got.Latitude != "28.704882" || got.Longitude != "77.103769" ||
		!strings.Contains(got.MapsURL, "&query=") || strings.ContainsRune(out, '\\') { // no HTML escaping such as \u0026
		t.Errorf("decode json+maps: out=%q err=%v", out, err)
	}
	if out, _, _ := capRun(t, []string{"decode", "39JJ567L7P", "--maps"}, ""); strings.Count(out, "\n") != 1 || !strings.Contains(out, "\thttps://") {
		t.Errorf("decode maps should be one tab-separated line: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"decode", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("decode bad: errOut=%q code=%d", errOut, code)
	}
	if _, errOut, code := capRun(t, []string{"decode"}, ""); code != 2 || errOut != "error: decode requires <code>\n" {
		t.Errorf("decode no arg: errOut=%q code=%d", errOut, code)
	}
	// After "--", "--json" is input, not a flag, so the code is invalid.
	if out, errOut, code := capRun(t, []string{"decode", "39J", "--", "--json"}, ""); code != 1 || out != "" || !strings.Contains(errOut, "bad character") {
		t.Errorf("decode -- --json: out=%q errOut=%q code=%d", out, errOut, code)
	}
	if out, _, code := capRun(t, []string{"decode", "--batch"}, "39JJ567L7P\n"); code != 0 || !strings.Contains(out, "28.704882") {
		t.Errorf("decode batch: out=%q code=%d", out, code)
	}
}

func TestBoundsCommand(t *testing.T) {
	if out, _, code := capRun(t, []string{"bounds", "39JJ567L7P"}, ""); code != 0 || !strings.Contains(out, " to ") {
		t.Errorf("bounds: out=%q code=%d", out, code)
	}
	if out, _, _ := capRun(t, []string{"bounds", "4", "--json"}, ""); strings.TrimSpace(out) !=
		`{"min_latitude":11.5,"max_latitude":20.5,"min_longitude":72.5,"max_longitude":81.5}` {
		t.Errorf("bounds json: out=%q", out)
	}
	if out, _, _ := capRun(t, []string{"bounds", "39JJ567L7P", "--geojson"}, ""); !strings.Contains(out, `"Feature"`) {
		t.Errorf("bounds geojson: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"bounds", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("bounds bad: errOut=%q code=%d", errOut, code)
	}
	if out, _, code := capRun(t, []string{"bounds", "--batch"}, "39JJ567L7P\n"); code != 0 || !strings.Contains(out, " to ") {
		t.Errorf("bounds batch: out=%q code=%d", out, code)
	}
}

func TestNeighborsCommand(t *testing.T) {
	out, _, code := capRun(t, []string{"neighbors", "39JJ567L7P"}, "")
	if code != 0 || len(strings.Fields(out)) != 8 {
		t.Errorf("neighbors: out=%q code=%d", out, code)
	}
	if out, _, _ := capRun(t, []string{"neighbors", "8", "--json"}, ""); strings.TrimSpace(out) != `["9","2","7"]` {
		t.Errorf("neighbors json: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"neighbors", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("neighbors bad: errOut=%q code=%d", errOut, code)
	}
	if out, _, code := capRun(t, []string{"neighbors", "--batch"}, "39JJ567L7P\n"); code != 0 || len(strings.Fields(out)) != 8 {
		t.Errorf("neighbors batch: out=%q code=%d", out, code)
	}
}
