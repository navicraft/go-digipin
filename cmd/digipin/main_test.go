package main

import (
	"bytes"
	"errors"
	"os"
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
	if out, _, code := capRun(t, []string{"version"}, ""); code != 0 || strings.TrimSpace(out) != "dev" {
		t.Errorf("version: out=%q code=%d", out, code)
	}
	if out, _, code := capRun(t, []string{"--version"}, ""); code != 0 || strings.TrimSpace(out) != "dev" {
		t.Errorf("--version: out=%q code=%d", out, code)
	}
	if out, _, code := capRun(t, []string{"help"}, ""); code != 0 || !strings.Contains(out, "Usage:") {
		t.Errorf("help: out=%q code=%d", out, code)
	}
	if _, errOut, code := capRun(t, []string{"bogus"}, ""); code != 2 || !strings.Contains(errOut, "unknown command") {
		t.Errorf("unknown: errOut=%q code=%d", errOut, code)
	}
}

func TestEncodeCommand(t *testing.T) {
	// spaced (default)
	if out, _, code := capRun(t, []string{"encode", "28.70486848", "77.10376044"}, ""); code != 0 || strings.TrimSpace(out) != "39J J567 L7P" {
		t.Errorf("spaced: out=%q code=%d", out, code)
	}
	// plain
	if out, _, code := capRun(t, []string{"encode", "28.70486848", "77.10376044", "--format", "plain"}, ""); code != 0 || strings.TrimSpace(out) != "39JJ567L7P" {
		t.Errorf("plain: out=%q code=%d", out, code)
	}
	// level
	if out, _, _ := capRun(t, []string{"encode", "12.9716", "77.5946", "--level", "3", "--format", "plain"}, ""); strings.TrimSpace(out) != "4P3" {
		t.Errorf("level3: out=%q", out)
	}
	// json + maps
	out, _, code := capRun(t, []string{"encode", "28.70486848", "77.10376044", "--json", "--maps"}, "")
	if code != 0 || !strings.Contains(out, `"digipin":"39JJ567L7P"`) || !strings.Contains(out, `"maps_url"`) {
		t.Errorf("json+maps: out=%q code=%d", out, code)
	}
	// maps, non-json
	if out, _, _ := capRun(t, []string{"encode", "28.70486848", "77.10376044", "--maps"}, ""); !strings.Contains(out, "google.com/maps") {
		t.Errorf("maps: out=%q", out)
	}
	// invalid format
	if _, errOut, code := capRun(t, []string{"encode", "12.9716", "77.5946", "--format", "xml"}, ""); code != 1 || !strings.Contains(errOut, "invalid format") {
		t.Errorf("bad format: errOut=%q code=%d", errOut, code)
	}
	// out-of-bounds coordinate -> encodeOut error
	if _, errOut, code := capRun(t, []string{"encode", "200", "77"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("oob: errOut=%q code=%d", errOut, code)
	}
	// wrong arg count
	if _, _, code := capRun(t, []string{"encode", "12.9716"}, ""); code != 2 {
		t.Errorf("argcount: code=%d, want 2", code)
	}
	// flag parse error
	if _, _, code := capRun(t, []string{"encode", "--nope"}, ""); code != 2 {
		t.Errorf("flag err: code=%d, want 2", code)
	}
}

func TestEncodeBatch(t *testing.T) {
	stdin := strings.Join([]string{
		"28.70486848,77.10376044", // ok
		"",                        // blank -> skipped
		"200,77",                  // out of bounds -> error
		"nocomma",                 // field count error
		"x,77",                    // bad lat
		"12,y",                    // bad lng
	}, "\n") + "\n"
	out, errOut, code := capRun(t, []string{"encode", "--batch", "--format", "plain"}, stdin)
	if code != 1 {
		t.Errorf("batch code = %d, want 1 (had errors)", code)
	}
	if strings.TrimSpace(out) != "39JJ567L7P" {
		t.Errorf("batch out = %q, want single success line", out)
	}
	for _, frag := range []string{"error:", "lat,lng", "invalid latitude", "invalid longitude"} {
		if !strings.Contains(errOut, frag) {
			t.Errorf("batch errOut missing %q: %q", frag, errOut)
		}
	}
}

// errReader yields one line, then a read error, exercising the scanner.Err path.
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
	if !strings.Contains(e.String(), "simulated read failure") {
		t.Errorf("missing read error in stderr: %q", e.String())
	}
}

func TestDecodeCommand(t *testing.T) {
	if out, _, code := capRun(t, []string{"decode", "39J J567 L7P"}, ""); code != 0 || !strings.Contains(out, "28.704882") {
		t.Errorf("decode: out=%q code=%d", out, code)
	}
	if out, _, _ := capRun(t, []string{"decode", "39JJ567L7P", "--json", "--maps"}, ""); !strings.Contains(out, `"latitude"`) || !strings.Contains(out, `"maps_url"`) {
		t.Errorf("decode json+maps: out=%q", out)
	}
	if out, _, _ := capRun(t, []string{"decode", "39JJ567L7P", "--maps"}, ""); !strings.Contains(out, "google.com/maps") {
		t.Errorf("decode maps: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"decode", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("decode bad: errOut=%q code=%d", errOut, code)
	}
	if _, _, code := capRun(t, []string{"decode"}, ""); code != 2 {
		t.Errorf("decode no arg: code=%d, want 2", code)
	}
	if _, _, code := capRun(t, []string{"decode", "--nope"}, ""); code != 2 {
		t.Errorf("decode flag err: code=%d", code)
	}
	// batch
	if out, _, code := capRun(t, []string{"decode", "--batch"}, "39JJ567L7P\n"); code != 0 || !strings.Contains(out, "28.704882") {
		t.Errorf("decode batch: out=%q code=%d", out, code)
	}
}

func TestBoundsCommand(t *testing.T) {
	if out, _, code := capRun(t, []string{"bounds", "39JJ567L7P"}, ""); code != 0 || !strings.Contains(out, " to ") {
		t.Errorf("bounds: out=%q code=%d", out, code)
	}
	if out, _, _ := capRun(t, []string{"bounds", "39JJ567L7P", "--json"}, ""); !strings.Contains(out, `"min_latitude"`) {
		t.Errorf("bounds json: out=%q", out)
	}
	if out, _, _ := capRun(t, []string{"bounds", "39JJ567L7P", "--geojson"}, ""); !strings.Contains(out, `"Feature"`) {
		t.Errorf("bounds geojson: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"bounds", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("bounds bad: errOut=%q code=%d", errOut, code)
	}
	if _, _, code := capRun(t, []string{"bounds"}, ""); code != 2 {
		t.Errorf("bounds no arg: code=%d", code)
	}
	if _, _, code := capRun(t, []string{"bounds", "--nope"}, ""); code != 2 {
		t.Errorf("bounds flag err: code=%d", code)
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
	if out, _, _ := capRun(t, []string{"neighbors", "39JJ567L7P", "--json"}, ""); !strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Errorf("neighbors json: out=%q", out)
	}
	if _, errOut, code := capRun(t, []string{"neighbors", "bad!"}, ""); code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("neighbors bad: errOut=%q code=%d", errOut, code)
	}
	if _, _, code := capRun(t, []string{"neighbors"}, ""); code != 2 {
		t.Errorf("neighbors no arg: code=%d", code)
	}
	if _, _, code := capRun(t, []string{"neighbors", "--nope"}, ""); code != 2 {
		t.Errorf("neighbors flag err: code=%d", code)
	}
	if out, _, code := capRun(t, []string{"neighbors", "--batch"}, "39JJ567L7P\n"); code != 0 || len(strings.Fields(out)) != 8 {
		t.Errorf("neighbors batch: out=%q code=%d", out, code)
	}
}
