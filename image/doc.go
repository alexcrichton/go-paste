// A package for paste implementing compression of various image formats.
//
// This package will register as many image compressors as possible so long as
// they're all installed on the system being currently run on. Currently the
// programs optipng, pngcrush, and jpegoptim are recognized and are used to
// compress their respective image formats.
//
// When importing this package, all available compressors will be registered
// when the resulting binary is first run (via init functions).
package image
