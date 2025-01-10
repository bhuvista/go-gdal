package gorio

/*
#include "../gorio.h"
#include <stdlib.h>

#cgo pkg-config: gdal
#cgo CXXFLAGS: -std=c++15
#cgo LDFLAGS: -ldl
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// Dataset represents a raster dataset
type Dataset struct {
	handle C.DatasetHandle
}

// RasterBand represents a single band in a dataset
type RasterBand struct {
	handle  C.BandHandle
	dataset *Dataset
}

// Bounds represents the spatial extent of a dataset
type Bounds struct {
	Left   float64
	Bottom float64
	Right  float64
	Top    float64
}

// Open opens a raster dataset for reading
func Open(filename string) (*Dataset, error) {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	handle := C.GOrioOpenDataset(cFilename)
	if handle == nil {
		return nil, errors.New("failed to open dataset")
	}

	ds := &Dataset{handle: handle}
	runtime.SetFinalizer(ds, (*Dataset).Close)
	return ds, nil
}

// Close closes the dataset and frees resources
func (ds *Dataset) Close() {
	if ds.handle != nil {
		C.GOrioCloseDataset(ds.handle)
		ds.handle = nil
	}
}

// Bounds returns the spatial bounds of the dataset
func (ds *Dataset) Bounds() (*Bounds, error) {
	var bounds C.Bounds
	if C.GOrioGetDatasetBounds(ds.handle, &bounds) != 0 {
		return nil, errors.New("failed to get bounds")
	}

	return &Bounds{
		Left:   float64(bounds.left),
		Bottom: float64(bounds.bottom),
		Right:  float64(bounds.right),
		Top:    float64(bounds.top),
	}, nil
}

// GetCRS returns the coordinate reference system as WKT
func (ds *Dataset) GetCRS() (string, error) {
	crs := C.GOrioGetDatasetCRS(ds.handle)
	if crs == nil {
		return "", errors.New("failed to get CRS")
	}
	defer C.free(unsafe.Pointer(crs))

	return C.GoString(crs), nil
}

// GetBand returns a specific raster band (1-based index)
func (ds *Dataset) GetBand(bandNum int) (*RasterBand, error) {
	handle := C.GOrioGetRasterBand(ds.handle, C.int(bandNum))
	if handle == nil {
		return nil, fmt.Errorf("failed to get band %d", bandNum)
	}

	return &RasterBand{
		handle:  handle,
		dataset: ds,
	}, nil
}

// Read reads data from the specified window into the provided buffer
func (rb *RasterBand) Read(xOff, yOff, xSize, ySize int, buffer interface{}) error {
	switch buf := buffer.(type) {
	case []float32:
		return rb.readFloat32(xOff, yOff, xSize, ySize, buf)
	case []float64:
		return rb.readFloat64(xOff, yOff, xSize, ySize, buf)
	case []int32:
		return rb.readInt32(xOff, yOff, xSize, ySize, buf)
	default:
		return fmt.Errorf("unsupported buffer type")
	}
}

func (rb *RasterBand) readFloat32(xOff, yOff, xSize, ySize int, buffer []float32) error {
	if len(buffer) < xSize*ySize {
		return errors.New("buffer too small")
	}

	if C.GOrioReadBandFloat32(rb.handle, C.int(xOff), C.int(yOff),
		C.int(xSize), C.int(ySize), unsafe.Pointer(&buffer[0])) != 0 {
		return errors.New("failed to read band data")
	}
	return nil
}

func (rb *RasterBand) readFloat64(xOff, yOff, xSize, ySize int, buffer []float64) error {
	if len(buffer) < xSize*ySize {
		return errors.New("buffer too small")
	}

	if C.GOrioReadBandFloat64(rb.handle, C.int(xOff), C.int(yOff),
		C.int(xSize), C.int(ySize), unsafe.Pointer(&buffer[0])) != 0 {
		return errors.New("failed to read band data")
	}
	return nil
}

func (rb *RasterBand) readInt32(xOff, yOff, xSize, ySize int, buffer []int32) error {
	if len(buffer) < xSize*ySize {
		return errors.New("buffer too small")
	}

	if C.GOrioReadBandInt32(rb.handle, C.int(xOff), C.int(yOff),
		C.int(xSize), C.int(ySize), unsafe.Pointer(&buffer[0])) != 0 {
		return errors.New("failed to read band data")
	}
	return nil
}

// CreateCopy creates a copy of the dataset with optional transformation
func CreateCopy(srcDS *Dataset, dstFilename string, options map[string]string) (*Dataset, error) {
	cDstFilename := C.CString(dstFilename)
	defer C.free(unsafe.Pointer(cDstFilename))

	// Convert options to C strings
	cOptions := make([]*C.char, 0, len(options)+1)
	for k, v := range options {
		opt := fmt.Sprintf("%s=%s", k, v)
		cOpt := C.CString(opt)
		cOptions = append(cOptions, cOpt)
	}
	cOptions = append(cOptions, nil)
	defer func() {
		for _, cOpt := range cOptions {
			if cOpt != nil {
				C.free(unsafe.Pointer(cOpt))
			}
		}
	}()

	handle := C.GOrioCreateCopy(srcDS.handle, cDstFilename, (**C.char)(&cOptions[0]))
	if handle == nil {
		return nil, errors.New("failed to create dataset copy")
	}

	ds := &Dataset{handle: handle}
	runtime.SetFinalizer(ds, (*Dataset).Close)
	return ds, nil
}

// Reproject reprojects the dataset to a new coordinate reference system
func (ds *Dataset) Reproject(dstCRS string, options map[string]string) (*Dataset, error) {
	cDstCRS := C.CString(dstCRS)
	defer C.free(unsafe.Pointer(cDstCRS))

	// Convert options similar to CreateCopy
	cOptions := make([]*C.char, 0, len(options)+1)
	for k, v := range options {
		opt := fmt.Sprintf("%s=%s", k, v)
		cOpt := C.CString(opt)
		cOptions = append(cOptions, cOpt)
	}
	cOptions = append(cOptions, nil)
	defer func() {
		for _, cOpt := range cOptions {
			if cOpt != nil {
				C.free(unsafe.Pointer(cOpt))
			}
		}
	}()

	handle := C.GOrioReproject(ds.handle, cDstCRS, (**C.char)(&cOptions[0]))
	if handle == nil {
		return nil, errors.New("failed to reproject dataset")
	}

	newDS := &Dataset{handle: handle}
	runtime.SetFinalizer(newDS, (*Dataset).Close)
	return newDS, nil
}
