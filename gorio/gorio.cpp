#define _GNU_SOURCE 1

#include "gorio.h"
#include <string>
#include <gdal.h>
#include <ogr_srs_api.h>
#include <cpl_conv.h>
#include "cpl_port.h"
#include "cpl_string.h"
#include <cpl_error.h>
#include "cpl_vsi.h"
#include "cpl_vsi_virtual.h"
#include <gdal_frmts.h>
#include <ogrsf_frmts.h>
#include <gdal_priv.h>
#include <gdal_utils.h>
#include <gdal_alg.h>
#include <gdalgrid.h>
#include <dlfcn.h>
#include <cassert>
#include <iostream>

#include <gdal_utils.h>
#include <gdal_alg.h>
#include <gdalgrid.h>
#include <vector>

// Data structures for dataset and band handles
typedef void* DatasetHandle;
typedef void* BandHandle;


inline int failed(cctx *ctx) {
	if (ctx->errMessage!=nullptr || ctx->failed!=0) {
		return 1;
	}
	return 0;
}

inline void forceError(cctx *ctx) {
	if (ctx->errMessage == nullptr && ctx->failed==0) {
		CPLError(CE_Failure, CPLE_AppDefined, "unknown error");
	}
}

DatasetHandle GOrioOpenDataset(const char* filename) {
    GDALAllRegister();
    GDALDataset* dataset = (GDALDataset*)GDALOpen(filename, GA_ReadOnly);
    return static_cast<DatasetHandle>(dataset);
}

void GOrioCloseDataset(DatasetHandle dataset) {
    if (dataset) {
        GDALClose(static_cast<GDALDataset*>(dataset));
    }
}

int GOrioGetDatasetBounds(DatasetHandle dataset, Bounds* bounds) {
    if (!dataset || !bounds) return 1;

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    double adfGeoTransform[6];
    
    if (ds->GetGeoTransform(adfGeoTransform) != CE_None) {
        return 1;
    }

    int width = ds->GetRasterXSize();
    int height = ds->GetRasterYSize();

    bounds->left = adfGeoTransform[0];
    bounds->top = adfGeoTransform[3];
    bounds->right = adfGeoTransform[0] + width * adfGeoTransform[1];
    bounds->bottom = adfGeoTransform[3] + height * adfGeoTransform[5];

    return 0;
}

char* GOrioGetDatasetCRS(DatasetHandle dataset) {
    if (!dataset) return nullptr;

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    const OGRSpatialReference* srs = ds->GetSpatialRef();
    if (!srs) return nullptr;

    char* wkt = nullptr;
    srs->exportToWkt(&wkt);
    return wkt;
}

BandHandle GOrioGetRasterBand(DatasetHandle dataset, int band_num) {
    if (!dataset) return nullptr;

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    GDALRasterBand* band = ds->GetRasterBand(band_num);
    return static_cast<BandHandle>(band);
}

int GOrioReadBandFloat32(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer) {
    if (!band || !buffer) return 1;

    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    CPLErr err = rb->RasterIO(GF_Read, xoff, yoff, xsize, ysize,
                             buffer, xsize, ysize, GDT_Float32,
                             0, 0);
    return (err == CE_None) ? 0 : 1;
}

int GOrioReadBandFloat64(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer) {
    if (!band || !buffer) return 1;

    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    CPLErr err = rb->RasterIO(GF_Read, xoff, yoff, xsize, ysize,
                             buffer, xsize, ysize, GDT_Float64,
                             0, 0);
    return (err == CE_None) ? 0 : 1;
}

int GOrioReadBandInt32(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer) {
    if (!band || !buffer) return 1;

    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    CPLErr err = rb->RasterIO(GF_Read, xoff, yoff, xsize, ysize,
                             buffer, xsize, ysize, GDT_Int32,
                             0, 0);
    return (err == CE_None) ? 0 : 1;
}

DatasetHandle GOrioCreateCopy(DatasetHandle src, const char* dst_filename, char** options) {
    if (!src) return nullptr;

    GDALDataset* srcDS = static_cast<GDALDataset*>(src);
    GDALDriver* driver = GetGDALDriverManager()->GetDriverByName("GTiff");
    if (!driver) return nullptr;

    GDALDataset* dstDS = driver->CreateCopy(dst_filename, srcDS, FALSE,
                                           const_cast<const char**>(options),
                                           nullptr, nullptr);
    return static_cast<DatasetHandle>(dstDS);
}

DatasetHandle GOrioReproject(DatasetHandle src, const char* dst_crs, char** options) {
    if (!src || !dst_crs) return nullptr;

    GDALDataset* srcDS = static_cast<GDALDataset*>(src);
    
    // Create warped VRT using the new API
    GDALDataset* warpedDS = (GDALDataset*)GDALAutoCreateWarpedVRT(
        srcDS,
        nullptr,  // source SRS (use the one from source dataset)
        dst_crs,  // target SRS
        GRIORA_Bilinear,  // Updated from GRA_Bilinear to GRIORA_Bilinear
        5.0,  // max error
        nullptr  // options
    );

    return static_cast<DatasetHandle>(warpedDS);
}

DatasetHandle GOrioWarp(DatasetHandle src, const char* dst_crs, char** options) {
    if (!src) return nullptr;
    
    GDALDataset* srcDS = static_cast<GDALDataset*>(src);
    
    // Create warped VRT
    GDALDataset* warpedDS = (GDALDataset*)GDALAutoCreateWarpedVRT(
        srcDS,
        nullptr,  // source SRS (use the one from source dataset)
        dst_crs,  // target SRS
        GRA_Bilinear,  // resampling algorithm
        0.0,  // max error
        nullptr  // transform options
    );
    
    return static_cast<DatasetHandle>(warpedDS);
}

DatasetHandle GOrioCreate(const char* filename, int width, int height, int bands,
                         GDALDataType datatype, char** options) {
    GDALDriver* driver = GetGDALDriverManager()->GetDriverByName("GTiff");
    if (!driver) return nullptr;
    
    GDALDataset* ds = driver->Create(filename, width, height, bands, datatype, options);
    return static_cast<DatasetHandle>(ds);
}

int GOrioWriteBand(BandHandle band, int xoff, int yoff, int xsize, int ysize, 
                   void* buffer, GDALDataType dtype) {
    if (!band || !buffer) return 1;
    
    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    CPLErr err = rb->RasterIO(GF_Write, xoff, yoff, xsize, ysize,
                             buffer, xsize, ysize, dtype,
                             0, 0);
    return (err == CE_None) ? 0 : 1;
}

int GOrioSetNoDataValue(BandHandle band, double nodata) {
    if (!band) return 1;
    
    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    CPLErr err = rb->SetNoDataValue(nodata);
    return (err == CE_None) ? 0 : 1;
}

int GOrioSetGeoTransform(DatasetHandle dataset, double* transform) {
    if (!dataset || !transform) return 1;
    
    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    CPLErr err = ds->SetGeoTransform(transform);
    return (err == CE_None) ? 0 : 1;
}

int GOrioSetProjection(DatasetHandle dataset, const char* wkt) {
    if (!dataset || !wkt) return 1;
    
    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    CPLErr err = ds->SetProjection(wkt);
    return (err == CE_None) ? 0 : 1;
}

int GOrioGetBlockSize(BandHandle band, int* xsize, int* ysize) {
    if (!band || !xsize || !ysize) return 1;
    
    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    rb->GetBlockSize(xsize, ysize);
    return 0;
}

double GOrioGetNoDataValue(BandHandle band, int* success) {
    if (!band) {
        if (success) *success = 0;
        return 0;
    }
    
    GDALRasterBand* rb = static_cast<GDALRasterBand*>(band);
    int has_nodata = 0;
    double nodata = rb->GetNoDataValue(&has_nodata);
    if (success) *success = has_nodata;
    return nodata;
}

int GOrioGetGeoTransform(DatasetHandle dataset, double* transform) {
    if (!dataset || !transform) return 1;

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    return ds->GetGeoTransform(transform);
}
// For OGRSpatialReference

char* GOrioGetAuthorityCode(const char* wkt) {
    if (!wkt) {
        return nullptr;
    }

    // Create an OGRSpatialReference object
    OGRSpatialReference srs;
    
    // Import WKT into the spatial reference object
    OGRErr err = srs.importFromWkt(wkt);
    if (err != OGRERR_NONE) {
        return nullptr;
    }

    // Retrieve the authority code
    const char* code = srs.GetAuthorityCode(nullptr);
    if (!code) {
        return nullptr;
    }

    // Copy the authority code into a newly allocated string
    char* codeCopy = static_cast<char*>(CPLMalloc(strlen(code) + 1));
    if (!codeCopy) {
        return nullptr;
    }

    strcpy(codeCopy, code);

    return codeCopy;
}

DatasetInfo GOrioGetDatasetInfo(DatasetHandle dataset) {
    DatasetInfo info = {0, 0, 0};
    if (!dataset) {
        return info;
    }

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);
    info.width = ds->GetRasterXSize();       // Get the width of the dataset
    info.height = ds->GetRasterYSize();      // Get the height of the dataset
    info.bandCount = ds->GetRasterCount();   // Get the number of bands in the dataset

    return info;
}


int GOrioConvertToImage(DatasetHandle dataset, const char* outputFilename, const char* format, char** options) {
    if (!dataset) {
        CPLError(CE_Failure, CPLE_AppDefined, "Invalid dataset handle.");
        return 1;
    }

    GDALDataset* ds = static_cast<GDALDataset*>(dataset);

    // Translate options
    GDALTranslateOptions* translateOptions = GDALTranslateOptionsNew(options, nullptr);
    if (!translateOptions) {
        CPLError(CE_Failure, CPLE_AppDefined, "Failed to create GDALTranslate options.");
        return 1;
    }

    // Perform the translation
    GDALDatasetH translatedDSH = GDALTranslate(outputFilename, static_cast<GDALDatasetH>(ds), translateOptions, nullptr);
    if (!translatedDSH) {
        CPLError(CE_Failure, CPLE_AppDefined, "Failed to translate dataset to the specified format.");
        GDALTranslateOptionsFree(translateOptions);
        return 1;
    }

    // Cast GDALDatasetH to GDALDataset* for proper cleanup
    GDALDataset* translatedDS = static_cast<GDALDataset*>(translatedDSH);

    // Cleanup
    GDALClose(translatedDS);
    GDALTranslateOptionsFree(translateOptions);
    return 0;
}



