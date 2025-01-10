package gorio

/*
#cgo pkg-config: gdal
#cgo CXXFLAGS: -std=c++11

#ifdef _WIN32
    #include <gdal/gdal.h>
    #include <gdal/gdal_priv.h>
    #include <gdal/cpl_conv.h>
    #include <gdal/ogr_srs_api.h>
#else
    #include "gdal.h"
    #include "gdal_priv.h"
    #include "cpl_conv.h"
    #include "ogr_srs_api.h"
#endif

#include <stdlib.h>

static void _GDALSetSpatialRef(GDALDatasetH hDS, const char* wkt) {
    GDALSetProjection(hDS, wkt);
}

static int _GDALGetDataTypeSize(GDALDataType dt) {
    return GDALGetDataTypeSize(dt);
}

static GDALDataType _GDALGetRasterDataType(GDALRasterBandH band) {
    return GDALGetRasterDataType(band);
}x`x`
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

type Dataset struct {
	ds        *C.GDALDatasetH
	mode      string
	driver    string
	width     int
	height    int
	count     int
	dtype     string
	crs       *CRS
	transform Transform
	closed    bool
	nodata    *float64
}

type DataType int

const (
	Unknown DataType = iota
	Byte
	UInt16
	Int16
	UInt32
	Int32
	Float32
	Float64
)

func init() {
	if runtime.GOOS == "windows" {
		// Windows-specific initialization if needed
	}
	C.GDALAllRegister()
}

func Open(path string) (*Dataset, error) {
	return OpenWithOptions(path, "r", "")
}

func OpenWithOptions(path, mode, driver string) (*Dataset, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var ds C.GDALDatasetH
	switch mode {
	case "r":
		ds = C.GDALOpen(cPath, C.GA_ReadOnly)
	case "w":
		ds = C.GDALOpen(cPath, C.GA_Update)
	default:
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	if ds == nil {
		return nil, fmt.Errorf("failed to open dataset: %s", path)
	}

	dataset := &Dataset{
		ds:     ds,
		mode:   mode,
		driver: driver,
		width:  int(C.GDALGetRasterXSize(ds)),
		height: int(C.GDALGetRasterYSize(ds)),
		count:  int(C.GDALGetRasterCount(ds)),
	}

	var transform [6]float64
	C.GDALGetGeoTransform(ds, (*C.double)(&transform[0]))
	dataset.transform = Transform(transform)

	projWKT := C.GDALGetProjectionRef(ds)
	if projWKT != nil {
		dataset.crs = &CRS{wkt: C.GoString(projWKT)}
	}

	// Get nodata value from first band
	if band := C.GDALGetRasterBand(ds, 1); band != nil {
		var hasNoData C.int
		nodata := C.GDALGetRasterNoDataValue(band, &hasNoData)
		if hasNoData != 0 {
			nodataVal := float64(nodata)
			dataset.nodata = &nodataVal
		}

		// Get data type
		gdalType := C._GDALGetRasterDataType(band)
		dataset.dtype = gdalTypeToString(gdalType)
	}

	runtime.SetFinalizer(dataset, (*Dataset).Close)
	return dataset, nil
}

func Create(path string, width, height, bands int, dtype string, options *CreationOptions) (*Dataset, error) {
	if options == nil {
		options = &CreationOptions{
			Driver: "GTiff",
			NoData: nil,
		}
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cDriver := C.CString(options.Driver)
	defer C.free(unsafe.Pointer(cDriver))

	driver := C.GDALGetDriverByName(cDriver)
	if driver == nil {
		return nil, fmt.Errorf("invalid driver: %s", options.Driver)
	}

	gdalType := stringToGDALType(dtype)
	if gdalType == C.GDT_Unknown {
		return nil, fmt.Errorf("unsupported data type: %s", dtype)
	}

	ds := C.GDALCreate(driver, cPath, C.int(width), C.int(height), C.int(bands), gdalType, nil)
	if ds == nil {
		return nil, fmt.Errorf("failed to create dataset: %s", path)
	}

	dataset := &Dataset{
		ds:     ds,
		mode:   "w",
		driver: options.Driver,
		width:  width,
		height: height,
		count:  bands,
		dtype:  dtype,
	}

	if options.NoData != nil {
		for i := 1; i <= bands; i++ {
			band := C.GDALGetRasterBand(ds, C.int(i))
			C.GDALSetRasterNoDataValue(band, C.double(*options.NoData))
		}
		dataset.nodata = options.NoData
	}

	runtime.SetFinalizer(dataset, (*Dataset).Close)
	return dataset, nil
}

func gdalTypeToString(gdalType C.GDALDataType) string {
	switch gdalType {
	case C.GDT_Byte:
		return "uint8"
	case C.GDT_UInt16:
		return "uint16"
	case C.GDT_Int16:
		return "int16"
	case C.GDT_UInt32:
		return "uint32"
	case C.GDT_Int32:
		return "int32"
	case C.GDT_Float32:
		return "float32"
	case C.GDT_Float64:
		return "float64"
	default:
		return "unknown"
	}
}

func stringToGDALType(dtype string) C.GDALDataType {
	switch dtype {
	case "uint8":
		return C.GDT_Byte
	case "uint16":
		return C.GDT_UInt16
	case "int16":
		return C.GDT_Int16
	case "uint32":
		return C.GDT_UInt32
	case "int32":
		return C.GDT_Int32
	case "float32":
		return C.GDT_Float32
	case "float64":
		return C.GDT_Float64
	default:
		return C.GDT_Unknown
	}
}

func (d *Dataset) Read(bandIndexes []int, window Window) ([]float64, error) {
	if d.closed {
		return nil, fmt.Errorf("dataset is closed")
	}

	nRows := window.Row[1] - window.Row[0]
	nCols := window.Col[1] - window.Col[0]

	buffer := make([]float64, nRows*nCols*len(bandIndexes))

	for i, bandIdx := range bandIndexes {
		band := C.GDALGetRasterBand(d.ds, C.int(bandIdx))
		if band == nil {
			return nil, fmt.Errorf("invalid band index: %d", bandIdx)
		}

		offset := i * nRows * nCols
		err := C.GDALRasterIO(band,
			C.GF_Read,
			C.int(window.Col[0]),
			C.int(window.Row[0]),
			C.int(nCols),
			C.int(nRows),
			unsafe.Pointer(&buffer[offset]),
			C.int(nCols),
			C.int(nRows),
			C.GDT_Float64,
			0,
			0)

		if err != C.CE_None {
			return nil, fmt.Errorf("failed to read band %d", bandIdx)
		}
	}

	return buffer, nil
}

func (d *Dataset) Write(data []float64, bandIndexes []int, window Window) error {
	if d.closed {
		return fmt.Errorf("dataset is closed")
	}

	if d.mode != "w" {
		return fmt.Errorf("dataset not opened in write mode")
	}

	nRows := window.Row[1] - window.Row[0]
	nCols := window.Col[1] - window.Col[0]

	for i, bandIdx := range bandIndexes {
		band := C.GDALGetRasterBand(d.ds, C.int(bandIdx))
		if band == nil {
			return fmt.Errorf("invalid band index: %d", bandIdx)
		}

		offset := i * nRows * nCols
		err := C.GDALRasterIO(band,
			C.GF_Write,
			C.int(window.Col[0]),
			C.int(window.Row[0]),
			C.int(nCols),
			C.int(nRows),
			unsafe.Pointer(&data[offset]),
			C.int(nCols),
			C.int(nRows),
			C.GDT_Float64,
			0,
			0)

		if err != C.CE_None {
			return fmt.Errorf("failed to write band %d", bandIdx)
		}
	}

	return nil
}

func (d *Dataset) Close() error {
	if d.closed {
		return nil
	}
	if d.ds != nil {
		C.GDALClose(d.ds)
	}
	d.closed = true
	return nil
}

func (d *Dataset) SetCRS(wkt string) error {
	if d.closed {
		return fmt.Errorf("dataset is closed")
	}

	cWKT := C.CString(wkt)
	defer C.free(unsafe.Pointer(cWKT))

	C._GDALSetSpatialRef(d.ds, cWKT)
	d.crs = &CRS{wkt: wkt}
	return nil
}

func (d *Dataset) SetTransform(transform Transform) error {
	if d.closed {
		return fmt.Errorf("dataset is closed")
	}

	err := C.GDALSetGeoTransform(d.ds, (*C.double)(&transform[0]))
	if err != C.CE_None {
		return fmt.Errorf("failed to set transform")
	}

	d.transform = transform
	return nil
}

func (d *Dataset) Bounds() Bounds {
	transform := d.Transform()
	return Bounds{
		Left:   transform[0],
		Top:    transform[3],
		Right:  transform[0] + float64(d.width)*transform[1],
		Bottom: transform[3] + float64(d.height)*transform[5],
	}
}

func (d *Dataset) CRS() *CRS {
	return d.crs
}

func (d *Dataset) Transform() Transform {
	return d.transform
}

func (d *Dataset) Width() int {
	return d.width
}

func (d *Dataset) Height() int {
	return d.height
}

func (d *Dataset) Count() int {
	return d.count
}

func (d *Dataset) DataType() string {
	return d.dtype
}
