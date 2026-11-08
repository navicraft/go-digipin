package digipin_test

import (
	"fmt"

	digipin "github.com/navicraft/go-digipin"
)

func ExampleEncode() {
	c, err := digipin.Encode(28.70486848, 77.10376044)
	if err != nil {
		panic(err)
	}
	fmt.Println(c.Canonical()) // storage / interop form
	fmt.Println(c.Formatted()) // 3-4-3 display form
	// Output:
	// 39JJ567L7P
	// 39J J567 L7P
}

func ExampleDecode() {
	// Decode is lenient: besides the specification's continuous and 3-4-3
	// spaced forms, it also tolerates hyphens and lower case.
	for _, code := range []string{"39JJ567L7P", "39J J567 L7P", "39j-j56-7l7p"} {
		c, _ := digipin.Decode(code)
		fmt.Printf("%.6f, %.6f\n", c.Latitude, c.Longitude)
	}
	// Output:
	// 28.704882, 77.103769
	// 28.704882, 77.103769
	// 28.704882, 77.103769
}

func ExampleEncodeLevel() {
	for level := 1; level <= 4; level++ {
		c, _ := digipin.EncodeLevel(12.9716, 77.5946, level)
		fmt.Println(c.Canonical())
	}
	// Output:
	// 4
	// 4P
	// 4P3
	// 4P3J
}

func ExampleCode_Neighbors() {
	c, _ := digipin.Encode(12.9716, 77.5946)
	fmt.Println(len(c.Neighbors()), "neighbors")
	// Output: 8 neighbors
}

func ExampleCode_DistanceTo() {
	a, _ := digipin.Encode(12.9716, 77.5946) // Bangalore
	b, _ := digipin.Encode(28.6139, 77.2090) // Delhi
	fmt.Printf("%.0f km\n", a.DistanceTo(b)/1000)
	// Output: 1740 km
}

func ExampleCoordinate_MapsLink() {
	fmt.Println(digipin.Coordinate{Latitude: 12.9716, Longitude: 77.5946}.MapsLink())
	// Output: https://www.google.com/maps/search/?api=1&query=12.9716%2C77.5946
}

func ExampleValidateCode() {
	// The specification allows only the continuous form or 3-4-3 spacing.
	for _, s := range []string{"39JJ567L7P", "39J J567 L7P", "39J-J56-7L7P"} {
		fmt.Println(s, digipin.ValidateCode(s) == nil)
	}
	// Output:
	// 39JJ567L7P true
	// 39J J567 L7P true
	// 39J-J56-7L7P false
}

func ExampleCode_GeoJSON() {
	c, _ := digipin.EncodeLevel(12.9716, 77.5946, 1)
	fmt.Println(string(c.GeoJSON()))
	// Output: {"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[72.5,11.5],[81.5,11.5],[81.5,20.5],[72.5,20.5],[72.5,11.5]]]},"properties":{"digipin":"4"}}
}
