# Gorio: A Go Wrapper for GDAL

Gorio is a Go package designed to provide a wrapper around the powerful Geospatial Data Abstraction Library (GDAL). It facilitates raster data manipulation, format conversion, and geospatial transformations while seamlessly integrating GDAL’s functionality into Go applications.

## Features

- Open and read raster datasets.
- Convert raster datasets to various formats (e.g., PNG, GeoTIFF).
- Retrieve dataset metadata such as size, bands, and bounds.
- Perform geospatial transformations and projections.

---

## Prerequisites

### System Requirements
- **Operating System**: Linux, macOS, or Windows
- **Compiler**: GCC or Clang with C++11 support
- **Go Version**: Go 1.16 or later

### Dependencies

1. **GDAL**
   - Minimum version: **3.0**
   - Ensure GDAL is installed with the required drivers for your use case (e.g., PNG, GeoTIFF).

   #### Installation:
   - On Linux:
     ```bash
     sudo apt-get update
     sudo apt-get install libgdal-dev
     ```
   - On macOS:
     ```bash
     brew install gdal
     ```
   - On Windows:
     Use OSGeo4W or download GDAL binaries from [GIS Internals](https://www.gisinternals.com/).

2. **Go cgo**
   Ensure `cgo` is enabled in your Go build environment.

---

## Installation

### 1. Clone the Repository
```bash
git clone https://github.com/bhuvista/gorio.git
cd gorio
```

### 2. Build the Package
```bash
make build
```

### 3. Test the Package
```bash
make test
```

### 4. Add Gorio to Your Go Project
```bash
go get github.com/bhuvista/gorio
```

---

## Usage

### Open a Dataset
```go
package main

import (
    "fmt"
    "log"

    "github.com/bhuvista/gorio"
)

func main() {
    dataset, err := gorio.Open("example.tif")
    if err != nil {
        log.Fatalf("Failed to open dataset: %v", err)
    }
    defer dataset.Close()

    fmt.Println("Dataset opened successfully")
}
```

### Convert to PNG
```go
package main

import (
    "log"

    "github.com/bhuvista/gorio"
)

func main() {
    dataset, err := gorio.Open("example.tif")
    if err != nil {
        log.Fatalf("Failed to open dataset: %v", err)
    }
    defer dataset.Close()

    err = dataset.ConvertToImage("output.png", "PNG")
    if err != nil {
        log.Fatalf("Failed to convert dataset to PNG: %v", err)
    }

    log.Println("Conversion successful")
}
```

---

## Development

### Code Structure
- **`gorio/gorio.go`**: Main Go wrapper exposing GDAL functionalities.
- **`gorio/gorio.h`**: C header file for GDAL interoperability.
- **`gorio/gorio.cpp`**: C++ implementation of GDAL utilities.

### Build the Project
```bash
make build
```

### Run Tests
```bash
make test
```

---

## Troubleshooting

### Common Errors

#### `Error initializing GDAL`
- **Cause**: GDAL is not properly installed or configured.
- **Solution**: Verify GDAL installation and ensure it is in your system's `PATH`.

#### `cgo` Compilation Issues
- **Cause**: Missing or incorrect GDAL headers.
- **Solution**: Ensure `libgdal-dev` is installed and the headers are accessible.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## Contributing

Contributions are welcome! Please open issues or submit pull requests on the [GitHub repository](https://github.com/bhuvista/gorio).

---

## Acknowledgments

- **GDAL**: The backbone of geospatial data manipulation.
- **Go Community**: For the robust ecosystem and support.

---

Enjoy seamless geospatial data handling with Gorio!

