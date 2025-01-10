#define _GNU_SOURCE 1
#include "gorio.h"
#include <gdal.h>
#include <ogr_srs_api.h>
#include <cpl_conv.h>
#include "cpl_port.h"
#include "cpl_string.h"
#include "cpl_vsi.h"
#include "cpl_vsi_virtual.h"
#include <gdal_frmts.h>
#include <ogrsf_frmts.h>
#include <dlfcn.h>
#include <cassert>

#include <gdal_utils.h>
#include <gdal_alg.h>
#include <gdalgrid.h>

#include <string>
#include <vector>

// Data structures for dataset and band handles
typedef void* DatasetHandle;
typedef void* BandHandle;

struct Bounds {
    double left, top, right, bottom;
};

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

DatasetHandle GOrioBuildVRT(const char* filename, int count, DatasetHandle* datasets, char** options) {
    if (!filename || count <= 0 || !datasets) return nullptr;

    // Convert handles to GDAL datasets
    std::vector<GDALDataset*> srcDatasets;
    for (int i = 0; i < count; i++) {
        if (datasets[i]) {
            srcDatasets.push_back(static_cast<GDALDataset*>(datasets[i]));
        }
    }

    if (srcDatasets.empty()) return nullptr;

    // Use the simplified BuildVRT API for GDAL 3.x
    GDALDataset* vrtDS = (GDALDataset*)GDALBuildVRT(
        filename,
        srcDatasets.size(),
        srcDatasets.data(),
        nullptr,  // options
        nullptr   // progress callback
    );

    return static_cast<DatasetHandle>(vrtDS);
}

// Custom error handler function (uses Go error handler)
static void godalErrorHandler(CPLErr e, CPLErrorNum n, const char* msg) {
    cctx *ctx = (cctx*)CPLGetErrorHandlerUserData();
    assert(ctx != nullptr);

    if (ctx->handlerIdx != 0) {
        int ret = goErrorHandler(ctx->handlerIdx, e, n, msg);
        if (ret != 0 && ctx->failed == 0) {
            ctx->failed = 1;
        }
    } else {
        // Strict: treat all warnings as errors
        if (e < CE_Warning) {
            fprintf(stderr, "GDAL: %s\n", msg);
            return;
        }
        if (ctx->errMessage == nullptr) {
            ctx->errMessage = (char*)malloc(strlen(msg) + 1);
            strcpy(ctx->errMessage, msg);
        } else {
            ctx->errMessage = (char*)realloc(ctx->errMessage, strlen(ctx->errMessage) + strlen(msg) + 3);
            strcat(ctx->errMessage, "\n");
            strcat(ctx->errMessage, msg);
        }
    }
}
