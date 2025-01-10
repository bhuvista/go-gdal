package tests

import (
	"fmt"
	"testing"

	"bhuvista.com/gorio/gorio"
)

func TestOpen(t *testing.T) {
	// Use a sample raster file
	filepath := "../testdata/sample.tif"

	ds, err := gorio.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open raster file: %v", err)
	}

	fmt.Println(ds.GetCRS())
	// fmt.Println(ds.Structure())
	// defer ds.Close()

	// if ds.Structure().SizeX <= 0 || ds.Structure().SizeY <= 0 {
	// 	t.Errorf("Invalid raster dimensions: %dx%d", ds.Structure().SizeX, ds.Structure().SizeY)
	// }
}

// func TestCreate(t *testing.T) {
// 	// Temporary file for testing
// 	tempFile := "../testdata/temp.tif"
// 	// defer os.Remove(tempFile)

// 	width, height := 100, 100
// 	format := "GTiff"

// 	ds, err := gorio.Create(tempFile, width, height, format)
// 	if err != nil {
// 		t.Fatalf("Failed to create raster file: %v", err)
// 	}
// 	defer ds.Close()

// 	if ds.Structure().SizeX != width || ds.Structure().SizeY != height {
// 		t.Errorf("Created raster has incorrect dimensions: %dx%d", ds.Structure().SizeX, ds.Structure().SizeY)
// 	}

// 	// Validate file creation
// 	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
// 		t.Fatalf("Raster file was not created: %s", tempFile)
// 	}
// }
