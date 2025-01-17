#ifndef GORIO_H
#define GORIO_H

#define _GNU_SOURCE 1

// Core GDAL headers
#include <gdal.h>
#include <gdal_alg.h>
#include <ogr_srs_api.h>
#include <cpl_conv.h>
#include "cpl_port.h"
#include <gdal_frmts.h>
#include <gdalwarper.h>

// Version check
#if GDAL_VERSION_NUM < 3000000
    #error "This code is only compatible with GDAL version >= 3.0"
#endif

// Resampling algorithm definitions if not present
#ifndef GRIORA_Bilinear
    #define GRIORA_Bilinear GRA_Bilinear
#endif

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Opaque handles for GDAL objects
 */
typedef void* DatasetHandle;
typedef void* BandHandle;

typedef struct {
    char *errMessage;
    int handlerIdx;
    int failed;
    char **configOptions;
} cctx;


/**
 * Represents the spatial extent of a dataset
 */
typedef struct {
    double left;    /* Left/west boundary */
    double bottom;  /* Bottom/south boundary */
    double right;   /* Right/east boundary */
    double top;     /* Top/north boundary */
} Bounds;

/**
 * Dataset error codes
 */
typedef enum {
    GORIO_SUCCESS = 0,          /* Operation succeeded */
    GORIO_ERROR_OPEN = 1,       /* Failed to open dataset */
    GORIO_ERROR_BOUNDS = 2,     /* Failed to retrieve bounds */
    GORIO_ERROR_CRS = 3,        /* Failed to retrieve CRS */
    GORIO_ERROR_BAND = 4,       /* Failed to access band */
    GORIO_ERROR_READ = 5,       /* Error reading band data */
    GORIO_ERROR_WRITE = 6,      /* Error writing data */
    GORIO_ERROR_COPY = 7,       /* Error copying dataset */
    GORIO_ERROR_REPROJECT = 8,  /* Error reprojecting dataset */
    GORIO_ERROR_VRT = 9,        /* Error building VRT */
    GORIO_ERROR_INVALID_PARAMS = 10 /* Invalid parameters */
} GOrioError;

/**
 * Dataset info
 */
typedef struct {
    int width;
    int height;
    int bandCount;
} DatasetInfo;

/* Dataset operations */

/**
 * Opens a raster dataset for reading
 * @param filename Path to the dataset
 * @return Handle to the dataset or NULL on failure
 */
DatasetHandle GOrioOpenDataset(const char* filename);

/**
 * Creates a new dataset
 */
DatasetHandle GOrioCreate(const char* filename, int width, int height, int bands,
                         GDALDataType datatype, char** options);

/**
 * Gets the authority code from a WKT string
 */
char* GOrioGetAuthorityCode(const char* wkt);

/**
 * Sets the geotransform for a dataset
 */
int GOrioSetGeoTransform(DatasetHandle dataset, double* transform);

/**
 * Sets the projection for a dataset
 */
int GOrioSetProjection(DatasetHandle dataset, const char* wkt);

/**
 * Sets the nodata value for a band
 */
int GOrioSetNoDataValue(BandHandle band, double nodata);

/**
 * Writes data to a band
 */
int GOrioWriteBand(BandHandle band, int xoff, int yoff, int xsize, int ysize,
                   void* buffer, GDALDataType dtype);

/**
 * Closes a dataset and frees resources
 * @param dataset Handle to the dataset
 */
void GOrioCloseDataset(DatasetHandle dataset);

/**
 * Gets the spatial bounds of a dataset
 * @param dataset Handle to the dataset
 * @param bounds Pointer to Bounds struct to fill
 * @return 0 on success, error code otherwise
 */
int GOrioGetDatasetBounds(DatasetHandle dataset, Bounds* bounds);

/**
 * Warps (reprojects) a dataset to a new coordinate system
 * @param src Source dataset
 * @param dst_crs Target CRS as WKT string
 * @param options Warp options (can be NULL)
 * @return Handle to warped dataset
 */
DatasetHandle GOrioWarp(DatasetHandle src, const char* dst_crs, char** options);

/**
 * Gets the coordinate reference system as WKT
 * @param dataset Handle to the dataset
 * @return WKT string (caller must free) or NULL on failure
 */
char* GOrioGetDatasetCRS(DatasetHandle dataset);

/**
 * Creates a copy of a dataset with optional transformation
 * @param src Source dataset handle
 * @param dst_filename Destination filename
 * @param options NULL-terminated array of strings with options
 * @return Handle to new dataset or NULL on failure
 */
DatasetHandle GOrioCreateCopy(DatasetHandle src, const char* dst_filename, char** options);

/**
 * Reprojects a dataset to a new coordinate reference system
 * @param src Source dataset handle
 * @param dst_crs Target CRS as string (WKT, PROJ.4, or EPSG)
 * @param options NULL-terminated array of strings with options
 * @return Handle to reprojected dataset or NULL on failure
 */
DatasetHandle GOrioReproject(DatasetHandle src, const char* dst_crs, char** options);

/**
 * Builds a VRT from multiple datasets
 * @param filename Output VRT filename
 * @param count Number of input datasets
 * @param datasets Array of dataset handles
 * @param options NULL-terminated array of strings with options
 * @return Handle to VRT dataset or NULL on failure
 */
DatasetHandle GOrioBuildVRT(const char* filename, int count, DatasetHandle* datasets, char** options);

/**
 * Gets the geotransform for a dataset
 */
int GOrioGetGeoTransform(DatasetHandle dataset, double* transform);

/* Band operations */

/**
 * Gets a raster band from a dataset
 * @param dataset Handle to the dataset
 * @param band_num Band number (1-based)
 * @return Handle to the band or NULL on failure
 */
BandHandle GOrioGetRasterBand(DatasetHandle dataset, int band_num);

/**
 * Reads float32 data from a raster band
 * @param band Handle to the band
 * @param xoff X offset
 * @param yoff Y offset
 * @param xsize Width to read
 * @param ysize Height to read
 * @param buffer Pre-allocated buffer to read into
 * @return 0 on success, error code otherwise
 */
int GOrioReadBandFloat32(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer);

/**
 * Reads float64 data from a raster band
 * @param band Handle to the band
 * @param xoff X offset
 * @param yoff Y offset
 * @param xsize Width to read
 * @param ysize Height to read
 * @param buffer Pre-allocated buffer to read into
 * @return 0 on success, error code otherwise
 */
int GOrioReadBandFloat64(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer);

/**
 * Reads int32 data from a raster band
 * @param band Handle to the band
 * @param xoff X offset
 * @param yoff Y offset
 * @param xsize Width to read
 * @param ysize Height to read
 * @param buffer Pre-allocated buffer to read into
 * @return 0 on success, error code otherwise
 */
int GOrioReadBandInt32(BandHandle band, int xoff, int yoff, int xsize, int ysize, void* buffer);

/* Error handling */

/**
 * Gets the last error message
 * @return Error message string or NULL if no error
 */
const char* GOrioGetLastErrorMsg();

/**
 * Clears the last error
 */
void GOrioClearError();

/**
 * Get dataset info
 */
DatasetInfo GOrioGetDatasetInfo(DatasetHandle dataset);

/**
 * Convert dataset to image
 */
int GOrioConvertToImage(DatasetHandle dataset, const char* outputFilename, const char* format, char** options);

#ifdef __cplusplus
}
#endif

#endif /* GORIO_H */
