package tests

import (
	"fmt"
	"log"
	"testing"

	"bhuvista.com/gorio/gorio"
)

func TestOpen(t *testing.T) {
	// Use a sample raster file
	filepath := "../../testdata/sample.tif"

	ds, err := gorio.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open raster file: %v", err)
	}
	defer ds.Close()

	transform, err := ds.GetGeoTransform()
	if err != nil {
		t.Errorf("Failed to get geotransform: %v", err)
	}
	fmt.Println(transform)

	bounds, err := ds.GetBounds()
	if err != nil {
		t.Errorf("Failed to get bounds: %v", err)
	}
	fmt.Println("Bounds: ", *bounds)

	epsgCode, err := ds.GetEPSGCode()
	if err != nil {
		t.Errorf("Failed to get CRS: %v", err)
	}
	fmt.Println("EPSG Code:", epsgCode)

	info, err := ds.Width()
	if err != nil {
		t.Errorf("Failed to get Dataset info: %v", err)
	}
	fmt.Println("Dataset Info:", info)

	err = ds.ToPng("../../testdata/sample.png")
	if err != nil {
		log.Fatalf("Failed to convert to PNG: %v", err)
	} else {
		fmt.Println("Successfully converted dataset to PNG!")
	}
}

func TestCreate(t *testing.T) {
	outfile := "../../testdata/test_create.tif"

	// Create a new dataset
	ds, err := gorio.Create(outfile, 100, 100, 1, gorio.Float32)
	if err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}
	defer ds.Close()

	// Set geotransform and projection
	transform := [6]float64{0, 1, 0, 0, 0, -1}
	if err := ds.SetGeoTransform(transform); err != nil {
		t.Errorf("Failed to set geotransform: %v", err)
	}

	wkt := `GEOGCS["WGS 84",DATUM["WGS_1984",SPHEROID["WGS 84",6378137,298.257223563]],PRIMEM["Greenwich",0],UNIT["degree",0.0174532925199433]]`
	if err := ds.SetProjection(wkt); err != nil {
		t.Errorf("Failed to set projection: %v", err)
	}

	// Write some data
	band, err := ds.GetBand(1)
	if err != nil {
		t.Fatalf("Failed to get band: %v", err)
	}

	data := make([]float32, 100*100)
	for i := range data {
		data[i] = float32(i)
	}

	if err := band.Write(0, 0, 100, 100, data); err != nil {
		t.Errorf("Failed to write data: %v", err)
	}
}
