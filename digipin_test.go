package digipin

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

var testCoordinates = []struct {
	name      string
	latitude  float64
	longitude float64
}{
	{"Bangalore, India", 12.9716, 77.5946},
	{"New Delhi, India", 28.6139, 77.2090},
	{"Mumbai, India", 19.0760, 72.8777},
	{"Chennai, India", 13.0827, 80.2707},
	{"Kolkata, India", 22.5726, 88.3639},
	{"Kochi, India", 9.9312, 76.2673},
	{"Jaipur, India", 26.9124, 75.7873},
	{"Hyderabad, India", 17.3850, 78.4867},
	{"Pune, India", 18.5204, 73.8567},
	{"Ahmedabad, India", 23.0225, 72.5714},
	{"Srinagar, India", 34.0837, 74.7973},
	{"Guwahati, India", 26.1445, 91.7362},
	{"High precision test", 12.123456, 77.654321},
}

func TestEncode(t *testing.T) {
	for _, tc := range testCoordinates {
		t.Run(tc.name, func(t *testing.T) {
			digipin, err := Encode(tc.latitude, tc.longitude)
			if err != nil {
				t.Fatalf("Encode(%f, %f) returned error: %v", tc.latitude, tc.longitude, err)
			}
			if digipin == nil {
				t.Fatal("Encode returned nil DIGIPIN")
			}
			if len(digipin.Code) != 12 {
				t.Errorf("Expected code length 12, got %d", len(digipin.Code))
			}
			parts := strings.Split(digipin.Code, "-")
			if len(parts) != 3 {
				t.Errorf("Expected 3 parts separated by dashes, got %d", len(parts))
			}
			if len(parts[0]) != 3 || len(parts[1]) != 3 || len(parts[2]) != 4 {
				t.Errorf("Expected format XXX-XXX-XXXX, got %s", digipin.Code)
			}
			cleanCode := strings.ReplaceAll(digipin.Code, "-", "")
			for i, char := range cleanCode {
				if !strings.ContainsRune(charset, char) {
					t.Errorf("Invalid character '%c' at position %d in code %s", char, i, digipin.Code)
				}
			}
			t.Logf("Coordinates (%.6f, %.6f) encoded to %s", tc.latitude, tc.longitude, digipin.Code)
		})
	}
}
func TestDecode(t *testing.T) {
	for _, tc := range testCoordinates {
		t.Run(tc.name, func(t *testing.T) {
			digipin, err := Encode(tc.latitude, tc.longitude)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			coord, err := Decode(digipin.Code)
			if err != nil {
				t.Fatalf("Decode(%s) returned error: %v", digipin.Code, err)
			}
			if coord == nil {
				t.Fatal("Decode returned nil coordinate")
			}
			const precision = 1e-2
			if math.Abs(coord.Latitude-tc.latitude) > precision {
				t.Errorf("Latitude mismatch: encoded %.6f, decoded %.6f, diff %.6f",
					tc.latitude, coord.Latitude, math.Abs(coord.Latitude-tc.latitude))
			}
			if math.Abs(coord.Longitude-tc.longitude) > precision {
				t.Errorf("Longitude mismatch: encoded %.6f, decoded %.6f, diff %.6f",
					tc.longitude, coord.Longitude, math.Abs(coord.Longitude-tc.longitude))
			}
			t.Logf("Code %s decoded to (%.6f, %.6f)", digipin.Code, coord.Latitude, coord.Longitude)
		})
	}
}
func TestEncodeInvalidCoordinates(t *testing.T) {
	testCases := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{"Latitude too high", 91.0, 0.0},
		{"Latitude too low", -91.0, 0.0},
		{"Longitude too high", 0.0, 181.0},
		{"Longitude too low", 0.0, -181.0},
		{"Both out of range", 100.0, 200.0},
		{"NaN latitude", math.NaN(), 0.0},
		{"NaN longitude", 0.0, math.NaN()},
		{"Inf latitude", math.Inf(1), 0.0},
		{"Inf longitude", 0.0, math.Inf(-1)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			digipin, err := Encode(tc.latitude, tc.longitude)
			if err == nil {
				t.Errorf("Expected error for coordinates (%.6f, %.6f), got DIGIPIN: %v",
					tc.latitude, tc.longitude, digipin)
			}
			if !errors.Is(err, ErrInvalidCoordinate) {
				t.Errorf("Expected ErrInvalidCoordinate, got %v", err)
			}
		})
	}
}
func TestDecodeInvalidDIGIPIN(t *testing.T) {
	testCases := []struct {
		name string
		code string
	}{
		{"Empty string", ""},
		{"Too short", "ABC-DEF"},
		{"Too long", "ABC-DEF-GHIJ-KLM"},
		{"Invalid characters", "ABC-DE1-0GHI"},
		{"No separators", "ABCDEFGHIJ"},
		{"Wrong separator positions", "AB-CDEF-GHIJ"},
		{"Invalid format", "ABC_DEF_GHIJ"},
		{"Mixed case with invalid chars", "abc-de1-0ghi"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coord, err := Decode(tc.code)
			if err == nil {
				t.Errorf("Expected error for code %s, got coordinate: %v", tc.code, coord)
			}
			if !errors.Is(err, ErrInvalidDIGIPIN) {
				t.Errorf("Expected ErrInvalidDIGIPIN, got %v", err)
			}
		})
	}
}
func TestRoundTrip(t *testing.T) {
	testCases := []struct {
		lat, lng float64
	}{
		{12.9716, 77.5946},
		{28.6139, 77.2090},
		{19.0760, 72.8777},
		{13.0827, 80.2707},
		{22.5726, 88.3639},
		{9.9312, 76.2673},
		{26.9124, 75.7873},
		{17.3850, 78.4867},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%.4f,%.4f", tc.lat, tc.lng), func(t *testing.T) {
			digipin, err := Encode(tc.lat, tc.lng)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			coord, err := Decode(digipin.Code)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			const maxError = 1e-1
			latError := math.Abs(coord.Latitude - tc.lat)
			lngError := math.Abs(coord.Longitude - tc.lng)
			if latError > maxError {
				t.Errorf("Latitude error %.6f > %.6f", latError, maxError)
			}
			if lngError > maxError {
				t.Errorf("Longitude error %.6f > %.6f", lngError, maxError)
			}
			t.Logf("Original: (%.6f, %.6f) -> %s -> (%.6f, %.6f)",
				tc.lat, tc.lng, digipin.Code, coord.Latitude, coord.Longitude)
		})
	}
}
func TestValidateCoordinates(t *testing.T) {
	validCases := []struct {
		lat, lng float64
	}{
		{0.0, 0.0},
		{90.0, 180.0},
		{-90.0, -180.0},
		{45.0, 90.0},
	}
	for _, tc := range validCases {
		if err := validateCoordinates(tc.lat, tc.lng); err != nil {
			t.Errorf("validateCoordinates(%.6f, %.6f) should be valid, got error: %v",
				tc.lat, tc.lng, err)
		}
	}
	invalidCases := []struct {
		lat, lng float64
	}{
		{91.0, 0.0},
		{-91.0, 0.0},
		{0.0, 181.0},
		{0.0, -181.0},
		{math.NaN(), 0.0},
		{0.0, math.NaN()},
		{math.Inf(1), 0.0},
		{0.0, math.Inf(-1)},
	}
	for _, tc := range invalidCases {
		if err := validateCoordinates(tc.lat, tc.lng); err == nil {
			t.Errorf("validateCoordinates(%.6f, %.6f) should be invalid", tc.lat, tc.lng)
		}
	}
}
func TestValidateDIGIPIN(t *testing.T) {
	validCases := []string{
		"CFJ-KLM-PT23",
		"234-567-89CF",
		"FTP-CJK-LM45",
	}
	for _, code := range validCases {
		if err := validateDIGIPIN(code); err != nil {
			t.Errorf("validateDIGIPIN(%s) should be valid, got error: %v", code, err)
		}
	}
	invalidCases := []string{
		"",
		"AB-DEF-GHJK",
		"ABCD-EF-GHJK",
		"CFJ-DE1-KLMN",
		"CFJ-DE0-KLMN",
		"CFJ-DEI-KLMN",
		"CFJ-DEO-KLMN",
	}
	for _, code := range invalidCases {
		if err := validateDIGIPIN(code); err == nil {
			t.Errorf("validateDIGIPIN(%s) should be invalid", code)
		}
	}
}
func TestCoordinateString(t *testing.T) {
	coord := &Coordinate{
		Latitude:  12.123456,
		Longitude: 77.654321,
	}
	expected := "(12.123456, 77.654321)"
	result := coord.String()
	if result != expected {
		t.Errorf("Coordinate.String() = %s, expected %s", result, expected)
	}
}
func TestCoordinateEqual(t *testing.T) {
	coord1 := &Coordinate{Latitude: 12.123456, Longitude: 77.654321}
	coord2 := &Coordinate{Latitude: 12.123456, Longitude: 77.654321}
	coord3 := &Coordinate{Latitude: 12.123457, Longitude: 77.654321}
	if !coord1.Equal(coord2) {
		t.Error("Equal coordinates should be equal")
	}
	if coord1.Equal(coord3) {
		t.Error("Different coordinates should not be equal")
	}
}
func TestDIGIPINString(t *testing.T) {
	digipin := &DIGIPIN{Code: "ABC-DEF-GHIJ"}
	expected := "ABC-DEF-GHIJ"
	result := digipin.String()
	if result != expected {
		t.Errorf("DIGIPIN.String() = %s, expected %s", result, expected)
	}
}
func BenchmarkEncode(b *testing.B) {
	lat, lng := 12.9716, 77.5946
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Encode(lat, lng)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkDecode(b *testing.B) {
	digipin, err := Encode(12.9716, 77.5946)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decode(digipin.Code)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkRoundTrip(b *testing.B) {
	lat, lng := 12.9716, 77.5946
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		digipin, err := Encode(lat, lng)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Decode(digipin.Code)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func TestConstants(t *testing.T) {
	if len(charset) != 16 {
		t.Errorf("charset length = %d, expected 16", len(charset))
	}
	if codeLength != 10 {
		t.Errorf("codeLength = %d, expected 10", codeLength)
	}
	if charsetSize != len(charset) {
		t.Errorf("charsetSize = %d, expected %d", charsetSize, len(charset))
	}
	ambiguous := "01IOilo"
	for _, char := range ambiguous {
		if strings.ContainsRune(charset, char) {
			t.Errorf("charset contains ambiguous character '%c'", char)
		}
	}
}
func TestPublicValidateCoordinates(t *testing.T) {
	validCases := []struct {
		lat, lng float64
	}{
		{0.0, 0.0},
		{90.0, 180.0},
		{-90.0, -180.0},
		{45.0, 90.0},
		{12.9716, 77.5946},
	}
	for _, tc := range validCases {
		if err := ValidateCoordinates(tc.lat, tc.lng); err != nil {
			t.Errorf("ValidateCoordinates(%.6f, %.6f) should be valid, got error: %v",
				tc.lat, tc.lng, err)
		}
	}
	invalidCases := []struct {
		lat, lng float64
	}{
		{91.0, 0.0},
		{-91.0, 0.0},
		{0.0, 181.0},
		{0.0, -181.0},
		{math.NaN(), 0.0},
		{0.0, math.NaN()},
		{math.Inf(1), 0.0},
		{0.0, math.Inf(-1)},
	}
	for _, tc := range invalidCases {
		if err := ValidateCoordinates(tc.lat, tc.lng); err == nil {
			t.Errorf("ValidateCoordinates(%.6f, %.6f) should be invalid", tc.lat, tc.lng)
		}
	}
}
func TestPublicValidateDIGIPIN(t *testing.T) {
	validCases := []string{
		"CFJ-KLM-PT23",
		"234-567-89CF",
		"FTP-CJK-LM45",
		"C9T-87K-L53F",
		"4MK-8T8-T862",
		"CFJKLMPT23",
	}
	for _, code := range validCases {
		if err := ValidateDIGIPIN(code); err != nil {
			t.Errorf("ValidateDIGIPIN(%s) should be valid, got error: %v", code, err)
		}
	}
	invalidCases := []string{
		"",
		"AB-DEF-GHJK",
		"ABCD-EF-GHJK",
		"CFJ-DE1-KLMN",
		"CFJ-DE0-KLMN",
		"CFJ-DEI-KLMN",
		"CFJ-DEO-KLMN",
		"ABC_DEF_GHJK",
		"ABCDEFGHJKL",
	}
	for _, code := range invalidCases {
		if err := ValidateDIGIPIN(code); err == nil {
			t.Errorf("ValidateDIGIPIN(%s) should be invalid", code)
		}
	}
}
func TestBoundaryCoordinates(t *testing.T) {
	boundaries := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"Northern India", 37.5, 75.0},
		{"Southern India", 3.0, 77.0},
		{"Eastern India", 25.0, 98.0},
		{"Western India", 25.0, 64.0},
		{"Northeast Corner", 38.0, 99.0},
		{"Southwest Corner", 3.0, 64.0},
	}
	for _, tc := range boundaries {
		t.Run(tc.name, func(t *testing.T) {
			digipin, err := Encode(tc.lat, tc.lng)
			if err != nil {
				t.Fatalf("Failed to encode boundary coordinate: %v", err)
			}
			coord, err := Decode(digipin.Code)
			if err != nil {
				t.Fatalf("Failed to decode boundary coordinate: %v", err)
			}
			const tolerance = 1e-1
			if math.Abs(coord.Latitude-tc.lat) > tolerance {
				t.Errorf("Latitude precision loss: %.6f vs %.6f", coord.Latitude, tc.lat)
			}
			if math.Abs(coord.Longitude-tc.lng) > tolerance {
				t.Errorf("Longitude precision loss: %.6f vs %.6f", coord.Longitude, tc.lng)
			}
		})
	}
}
func TestCoordinateBelongsTo(t *testing.T) {
	coord := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	digipin, err := Encode(coord.Latitude, coord.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate: %v", err)
	}
	belongs, err := coord.BelongsTo(digipin, 0)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	if !belongs {
		t.Error("Coordinate should belong to its own DIGIPIN code")
	}
	belongs, err = coord.BelongsTo(digipin, 1e-3)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	if !belongs {
		t.Error("Coordinate should belong to its own DIGIPIN code with custom tolerance")
	}
	_, err = coord.BelongsTo(digipin, 1e-5)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	invalidDIGIPIN := &DIGIPIN{Code: "INVALID-CODE"}
	belongs, err = coord.BelongsTo(invalidDIGIPIN, 0)
	if err == nil {
		t.Error("Expected error for invalid DIGIPIN code")
	}
	if belongs {
		t.Error("Should not belong to invalid DIGIPIN")
	}
}
func TestDIGIPINBelongsTo(t *testing.T) {
	coord := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	digipin, err := Encode(coord.Latitude, coord.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate: %v", err)
	}
	belongs, err := digipin.BelongsTo(coord, 0)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	if !belongs {
		t.Error("DIGIPIN should belong to its original coordinate")
	}
	belongs, err = digipin.BelongsTo(coord, 1e-3)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	if !belongs {
		t.Error("DIGIPIN should belong to its original coordinate with custom tolerance")
	}
	differentCoord := &Coordinate{Latitude: 13.0000, Longitude: 78.0000}
	belongs, err = digipin.BelongsTo(differentCoord, 1e-4)
	if err != nil {
		t.Errorf("BelongsTo returned error: %v", err)
	}
	if belongs {
		t.Error("DIGIPIN should not belong to significantly different coordinate")
	}
	invalidDIGIPIN := &DIGIPIN{Code: "INVALID-CODE"}
	_, err = invalidDIGIPIN.BelongsTo(coord, 0)
	if err == nil {
		t.Error("Expected error for invalid DIGIPIN code")
	}
}
func TestCoordinateEqualWithTolerance(t *testing.T) {
	coord1 := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	coord2 := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	coord3 := &Coordinate{Latitude: 12.9717, Longitude: 77.5947}
	if !coord1.EqualWithTolerance(coord2, 1e-6) {
		t.Error("Identical coordinates should be equal")
	}
	if !coord1.EqualWithTolerance(coord3, 1e-2) {
		t.Error("Coordinates within tolerance should be equal")
	}
	if coord1.EqualWithTolerance(coord3, 1e-5) {
		t.Error("Coordinates outside tolerance should not be equal")
	}
	if !coord1.EqualWithTolerance(coord2, 0) {
		t.Error("Should use default tolerance when tolerance is 0")
	}
}
func TestDIGIPINEqual(t *testing.T) {
	coord := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	digipin1, err := Encode(coord.Latitude, coord.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate: %v", err)
	}
	digipin2, err := Encode(coord.Latitude, coord.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate: %v", err)
	}
	equal, err := digipin1.Equal(digipin2)
	if err != nil {
		t.Errorf("Equal returned error: %v", err)
	}
	if !equal {
		t.Error("DIGIPIN codes for same coordinate should be equal")
	}
	differentCoord := &Coordinate{Latitude: 13.0000, Longitude: 78.0000}
	digipin3, err := Encode(differentCoord.Latitude, differentCoord.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode different coordinate: %v", err)
	}
	equal, err = digipin1.Equal(digipin3)
	if err != nil {
		t.Errorf("Equal returned error: %v", err)
	}
	if equal {
		t.Error("DIGIPIN codes for different coordinates should not be equal")
	}
	invalidDIGIPIN := &DIGIPIN{Code: "INVALID-CODE"}
	_, err = digipin1.Equal(invalidDIGIPIN)
	if err == nil {
		t.Error("Expected error for invalid DIGIPIN code")
	}
}
func TestDIGIPINEqualWithTolerance(t *testing.T) {
	coord1 := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	coord2 := &Coordinate{Latitude: 12.9717, Longitude: 77.5947}
	digipin1, err := Encode(coord1.Latitude, coord1.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate 1: %v", err)
	}
	digipin2, err := Encode(coord2.Latitude, coord2.Longitude)
	if err != nil {
		t.Fatalf("Failed to encode coordinate 2: %v", err)
	}
	equal, err := digipin1.EqualWithTolerance(digipin2, 1e-2)
	if err != nil {
		t.Errorf("EqualWithTolerance returned error: %v", err)
	}
	if !equal {
		t.Error("DIGIPIN codes should be equal with large tolerance")
	}
	equal, err = digipin1.EqualWithTolerance(digipin2, 1e-5)
	if err != nil {
		t.Errorf("EqualWithTolerance returned error: %v", err)
	}
	if equal {
		t.Error("DIGIPIN codes should not be equal with small tolerance")
	}
	equal, err = digipin1.EqualWithTolerance(digipin1, 0)
	if err != nil {
		t.Errorf("EqualWithTolerance returned error: %v", err)
	}
	if !equal {
		t.Error("Should use default tolerance when tolerance is 0")
	}
	invalidDIGIPIN := &DIGIPIN{Code: "INVALID-CODE"}
	_, err = digipin1.EqualWithTolerance(invalidDIGIPIN, 1e-3)
	if err == nil {
		t.Error("Expected error for invalid DIGIPIN code")
	}
}
func TestDefaultTolerance(t *testing.T) {
	if DefaultTolerance <= 0 {
		t.Error("DefaultTolerance should be positive")
	}
	if DefaultTolerance > 1e-2 {
		t.Error("DefaultTolerance should be small enough for reasonable precision")
	}
	coord1 := &Coordinate{Latitude: 12.9716, Longitude: 77.5946}
	coord2 := &Coordinate{Latitude: 12.9716 + DefaultTolerance/2, Longitude: 77.5946}
	if !coord1.EqualWithTolerance(coord2, 0) {
		t.Error("Coordinates within DefaultTolerance should be equal when tolerance is 0")
	}
}
func TestGetBounds(t *testing.T) {
	testCases := []struct {
		name      string
		digipin   string
		shouldErr bool
	}{
		{"Valid DIGIPIN", "C9T-87K-L53F", false},
		{"Another valid DIGIPIN", "4MK-8T8-T862", false},
		{"Invalid DIGIPIN", "INVALID-CODE", true},
		{"Empty string", "", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bounds, err := GetBounds(tc.digipin)
			if tc.shouldErr {
				if err == nil {
					t.Error("Expected error for invalid DIGIPIN")
				}
				return
			}
			if err != nil {
				t.Errorf("GetBounds returned error: %v", err)
				return
			}
			if bounds == nil {
				t.Error("GetBounds returned nil bounds")
				return
			}
			if bounds.MinLatitude >= bounds.MaxLatitude {
				t.Error("MinLatitude should be less than MaxLatitude")
			}
			if bounds.MinLongitude >= bounds.MaxLongitude {
				t.Error("MinLongitude should be less than MaxLongitude")
			}
			if bounds.MinLatitude < -90 || bounds.MaxLatitude > 90 {
				t.Error("Latitude bounds are outside valid range")
			}
			if bounds.MinLongitude < -180 || bounds.MaxLongitude > 180 {
				t.Error("Longitude bounds are outside valid range")
			}
			t.Logf("DIGIPIN %s bounds: %s", tc.digipin, bounds.String())
		})
	}
}
func TestDIGIPINGetBounds(t *testing.T) {
	digipin := &DIGIPIN{Code: "C9T-87K-L53F"}
	bounds, err := digipin.GetBounds()
	if err != nil {
		t.Errorf("DIGIPIN.GetBounds returned error: %v", err)
	}
	if bounds == nil {
		t.Error("DIGIPIN.GetBounds returned nil bounds")
	}
	invalidDIGIPIN := &DIGIPIN{Code: "INVALID"}
	_, err = invalidDIGIPIN.GetBounds()
	if err == nil {
		t.Error("Expected error for invalid DIGIPIN code")
	}
}
func TestBoundsString(t *testing.T) {
	bounds := &Bounds{
		MinLatitude:  12.970000,
		MaxLatitude:  12.972000,
		MinLongitude: 77.594000,
		MaxLongitude: 77.596000,
	}
	expected := "(12.970000,77.594000) to (12.972000,77.596000)"
	result := bounds.String()
	if result != expected {
		t.Errorf("Bounds.String() = %s, expected %s", result, expected)
	}
}
func TestBoundsArea(t *testing.T) {
	bounds := &Bounds{
		MinLatitude:  12.970000,
		MaxLatitude:  12.972000,
		MinLongitude: 77.594000,
		MaxLongitude: 77.596000,
	}
	expectedArea := (12.972000 - 12.970000) * (77.596000 - 77.594000)
	area := bounds.Area()
	if math.Abs(area-expectedArea) > 1e-9 {
		t.Errorf("Bounds.Area() = %f, expected %f", area, expectedArea)
	}
}
func TestBoundsContains(t *testing.T) {
	bounds := &Bounds{
		MinLatitude:  12.970000,
		MaxLatitude:  12.972000,
		MinLongitude: 77.594000,
		MaxLongitude: 77.596000,
	}
	testCases := []struct {
		name     string
		coord    *Coordinate
		expected bool
	}{
		{"Inside bounds", &Coordinate{12.971000, 77.595000}, true},
		{"On min boundary", &Coordinate{12.970000, 77.594000}, true},
		{"On max boundary", &Coordinate{12.972000, 77.596000}, true},
		{"Outside latitude", &Coordinate{12.973000, 77.595000}, false},
		{"Outside longitude", &Coordinate{12.971000, 77.597000}, false},
		{"Outside both", &Coordinate{12.969000, 77.593000}, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := bounds.Contains(tc.coord)
			if result != tc.expected {
				t.Errorf("Contains(%s) = %v, expected %v", tc.coord.String(), result, tc.expected)
			}
		})
	}
}
func TestBoundsRoundTrip(t *testing.T) {
	testCoords := []struct {
		lat, lng float64
	}{
		{12.9716, 77.5946},
		{28.6139, 77.2090},
		{19.0760, 72.8777},
		{13.0827, 80.2707},
		{22.5726, 88.3639},
	}
	for _, tc := range testCoords {
		t.Run(fmt.Sprintf("%.4f,%.4f", tc.lat, tc.lng), func(t *testing.T) {
			digipin, err := Encode(tc.lat, tc.lng)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			bounds, err := GetBounds(digipin.Code)
			if err != nil {
				t.Fatalf("GetBounds failed: %v", err)
			}
			coord := &Coordinate{Latitude: tc.lat, Longitude: tc.lng}
			decoded, err := Decode(digipin.Code)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if !bounds.Contains(decoded) {
				t.Errorf("Bounds %s should contain decoded coordinate %s", bounds.String(), decoded.String())
			}
			t.Logf("Original: %s -> DIGIPIN: %s -> Bounds: %s -> Decoded: %s",
				coord.String(), digipin.Code, bounds.String(), decoded.String())
		})
	}
}
func TestOfficialDIGIPINSamples(t *testing.T) {
	samples := []struct {
		name      string
		digipin   string
		latitude  float64
		longitude float64
	}{
		{
			name:      "Delhi Sample 1",
			digipin:   "39J-J56-7L7P",
			latitude:  28.70486848,
			longitude: 77.10376044,
		},
		{
			name:      "Delhi Sample 2",
			digipin:   "39J-J56-7MJL",
			latitude:  28.704879,
			longitude: 77.103840,
		},
	}
	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			encoded, err := Encode(sample.latitude, sample.longitude)
			if err != nil {
				t.Errorf("Failed to encode coordinates (%.8f, %.8f): %v",
					sample.latitude, sample.longitude, err)
			} else {
				if encoded.Code != sample.digipin {
					t.Errorf("Encoding mismatch: got %s, expected %s",
						encoded.Code, sample.digipin)
				} else {
					t.Logf("✅ Encoding match: (%.8f, %.8f) -> %s",
						sample.latitude, sample.longitude, encoded.Code)
				}
			}
			decoded, err := Decode(sample.digipin)
			if err != nil {
				t.Errorf("Failed to decode DIGIPIN %s: %v", sample.digipin, err)
			} else {
				latDiff := math.Abs(decoded.Latitude - sample.latitude)
				lngDiff := math.Abs(decoded.Longitude - sample.longitude)
				const precision = 0.0001
				if latDiff > precision || lngDiff > precision {
					t.Errorf("Decoding precision too low: lat diff=%.8f, lng diff=%.8f (tolerance=%.8f)",
						latDiff, lngDiff, precision)
				} else {
					t.Logf("✅ Decoding precision: %s -> (%.8f, %.8f), diff=(%.8f, %.8f)",
						sample.digipin, decoded.Latitude, decoded.Longitude, latDiff, lngDiff)
				}
			}
			encoded, err = Encode(sample.latitude, sample.longitude)
			if err != nil {
				t.Fatalf("Round-trip encode failed: %v", err)
			}
			decoded, err = Decode(encoded.Code)
			if err != nil {
				t.Fatalf("Round-trip decode failed: %v", err)
			}
			latError := math.Abs(decoded.Latitude - sample.latitude)
			lngError := math.Abs(decoded.Longitude - sample.longitude)
			const roundTripTolerance = 0.001
			if latError > roundTripTolerance || lngError > roundTripTolerance {
				t.Errorf("Round-trip error too large: lat=%.8f, lng=%.8f",
					latError, lngError)
			} else {
				t.Logf("✅ Round-trip: (%.8f, %.8f) -> %s -> (%.8f, %.8f)",
					sample.latitude, sample.longitude, encoded.Code,
					decoded.Latitude, decoded.Longitude)
			}
		})
	}
}
