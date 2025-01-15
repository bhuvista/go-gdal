// gorio.go
package gorio

/*
#cgo CXXFLAGS: -std=c++11 -I/opt/homebrew/Cellar/gdal/3.10.0_4/include
#cgo LDFLAGS: -L/opt/homebrew/Cellar/gdal/3.10.0_4/lib -lgdal
#cgo darwin LDFLAGS: -Wl,-rpath,/opt/homebrew/Cellar/gdal/3.10.0_4/lib
#include <gdal.h>
#include "gorio.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"
)

type Dataset struct {
	handle C.DatasetHandle
}

type DatasetInfo struct {
	info C.DatasetInfo
}

type Band struct {
	handle C.BandHandle
}

type Bounds struct {
	Left   float64
	Bottom float64
	Right  float64
	Top    float64
}

// DataType represents GDAL data types
type DataType int32

// GDAL data type constants
const (
	Float32 DataType = 6 // GDT_Float32
	Float64 DataType = 7 // GDT_Float64
	Int32   DataType = 5 // GDT_Int32
)

func init() {
	C.GDALAllRegister()
}

func Open(filename string) (*Dataset, error) {
	cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cfilename))

	handle := C.GOrioOpenDataset(cfilename)
	if handle == nil {
		return nil, errors.New("failed to open dataset")
	}

	return &Dataset{handle: handle}, nil
}

func (ds *Dataset) Close() {
	if ds.handle != nil {
		C.GOrioCloseDataset(ds.handle)
		ds.handle = nil
	}
}

func (ds *Dataset) GetBounds() (*Bounds, error) {
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

func (ds *Dataset) GetCRS() (string, error) {
	wkt := C.GOrioGetDatasetCRS(ds.handle)
	if wkt == nil {
		return "", errors.New("failed to get CRS")
	}
	defer C.free(unsafe.Pointer(wkt))
	return C.GoString(wkt), nil
}

func (ds *Dataset) GetEPSGCode() (int, error) {
	wkt := C.GOrioGetDatasetCRS(ds.handle)
	if wkt == nil {
		return 0, errors.New("failed to get CRS")
	}
	defer C.free(unsafe.Pointer(wkt))

	// Convert Go string to C-style string (null-terminated)
	cWkt := C.CString(C.GoString(wkt))
	defer C.free(unsafe.Pointer(cWkt))

	authorityName := C.CString("EPSG")
	defer C.free(unsafe.Pointer(authorityName))

	authorityCode := C.GOrioGetAuthorityCode(cWkt)
	if authorityCode == nil {
		return 0, errors.New("EPSG code not found or invalid")
	}
	defer C.free(unsafe.Pointer(authorityCode))

	epsgCode, err := strconv.Atoi(C.GoString(authorityCode))
	if err != nil {
		return 0, fmt.Errorf("failed to parse EPSG code: %w", err)
	}

	return epsgCode, nil
}

func (ds *Dataset) GetBand(bandNum int) (*Band, error) {
	handle := C.GOrioGetRasterBand(ds.handle, C.int(bandNum))
	if handle == nil {
		return nil, errors.New("failed to get band")
	}
	return &Band{handle: handle}, nil
}

func (b *Band) ReadFloat32(xoff, yoff, xsize, ysize int) ([]float32, error) {
	buffer := make([]float32, xsize*ysize)
	if C.GOrioReadBandFloat32(b.handle, C.int(xoff), C.int(yoff),
		C.int(xsize), C.int(ysize), unsafe.Pointer(&buffer[0])) != 0 {
		return nil, errors.New("failed to read band data")
	}
	return buffer, nil
}

func (b *Band) ReadFloat64(xoff, yoff, xsize, ysize int) ([]float64, error) {
	buffer := make([]float64, xsize*ysize)
	if C.GOrioReadBandFloat64(b.handle, C.int(xoff), C.int(yoff),
		C.int(xsize), C.int(ysize), unsafe.Pointer(&buffer[0])) != 0 {
		return nil, errors.New("failed to read band data")
	}
	return buffer, nil
}

func (b *Band) ReadInt32(xoff, yoff, xsize, ysize int) ([]int32, error) {
	buffer := make([]int32, xsize*ysize)
	if C.GOrioReadBandInt32(b.handle, C.int(xoff), C.int(yoff),
		C.int(xsize), C.int(ysize), unsafe.Pointer(&buffer[0])) != 0 {
		return nil, errors.New("failed to read band data")
	}
	return buffer, nil
}

func Create(filename string, width, height, bands int, datatype DataType) (*Dataset, error) {
	cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cfilename))
	handle := C.GOrioCreate(cfilename, C.int(width), C.int(height), C.int(bands),
		C.GDALDataType(datatype), nil)

	if handle == nil {
		return nil, errors.New("failed to create dataset")
	}
	return &Dataset{handle: handle}, nil
}

func (ds *Dataset) SetGeoTransform(transform [6]float64) error {
	if C.GOrioSetGeoTransform(ds.handle, (*C.double)(&transform[0])) != 0 {
		return errors.New("failed to set geotransform")
	}
	return nil
}

func (ds *Dataset) SetProjection(wkt string) error {
	cwkt := C.CString(wkt)
	defer C.free(unsafe.Pointer(cwkt))
	if C.GOrioSetProjection(ds.handle, cwkt) != 0 {
		return errors.New("failed to set projection")
	}
	return nil
}

func (b *Band) SetNoDataValue(nodata float64) error {
	if C.GOrioSetNoDataValue(b.handle, C.double(nodata)) != 0 {
		return errors.New("failed to set nodata value")
	}
	return nil
}

func (b *Band) Write(xoff, yoff, xsize, ysize int, data interface{}) error {
	var dtype C.GDALDataType

	switch v := data.(type) {
	case []float32:
		dtype = C.GDT_Float32
		if C.GOrioWriteBand(b.handle, C.int(xoff), C.int(yoff), C.int(xsize), C.int(ysize),
			unsafe.Pointer(&v[0]), dtype) != 0 {
			return errors.New("failed to write float32 band data")
		}
	case []float64:
		dtype = C.GDT_Float64
		if C.GOrioWriteBand(b.handle, C.int(xoff), C.int(yoff), C.int(xsize), C.int(ysize),
			unsafe.Pointer(&v[0]), dtype) != 0 {
			return errors.New("failed to write float64 band data")
		}
	case []int32:
		dtype = C.GDT_Int32
		if C.GOrioWriteBand(b.handle, C.int(xoff), C.int(yoff), C.int(xsize), C.int(ysize),
			unsafe.Pointer(&v[0]), dtype) != 0 {
			return errors.New("failed to write int32 band data")
		}
	default:
		return errors.New("unsupported data type")
	}
	return nil
}

func (ds *Dataset) GetGeoTransform() ([6]float64, error) {
	var transform [6]float64
	if C.GOrioGetGeoTransform(ds.handle, (*C.double)(&transform[0])) != 0 {
		return [6]float64{}, errors.New("failed to get geotransform")
	}
	return transform, nil
}

func (ds *Dataset) GetDatasetInfo() (*DatasetInfo, error) {
	info := C.GOrioGetDatasetInfo(ds.handle)
	return &DatasetInfo{info: info}, nil
}

func (ds *Dataset) Width() (int, error) {
	info := C.GOrioGetDatasetInfo(ds.handle)
	return int(info.width), nil
}

func (ds *Dataset) Height() (int, error) {
	info := C.GOrioGetDatasetInfo(ds.handle)
	return int(info.height), nil
}

func (ds *Dataset) Bands() (int, error) {
	info := C.GOrioGetDatasetInfo(ds.handle)
	return int(info.bandCount), nil
}

func (ds *Dataset) ToPng(outputFilename string) error {
	format := "PNG"
	cOutputFilename := C.CString(outputFilename)
	defer C.free(unsafe.Pointer(cOutputFilename))

	result := C.GOrioConvertToImage(ds.handle, cOutputFilename, C.CString(format), nil)
	if result != 0 {
		return fmt.Errorf("failed to convert dataset to %s", format)
	}

	return nil
}
