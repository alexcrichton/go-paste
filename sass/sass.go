// Package for writing/compressing stylesheets with sass.
//
// This package requires that 'libsass' is installed on the system and that it's
// pretty much the current master version (as of this writing). When imported,
// this file will automatically run all css through the sass compressor and it
// will also transform all 'scss' files into 'css' files.
package sass

// #cgo LDFLAGS: -lsass
// #include <sass_interface.h>
import "C"

import "errors"
import "github.com/alexcrichton/go-paste"
import "os"

func init() {
  paste.RegisterProcessor(paste.ProcessorFunc(translate), ".scss", false)
  paste.RegisterProcessor(paste.ProcessorFunc(minify), ".css", true)
  paste.RegisterAlias(".css", ".scss")
}

func compile(infile, outfile string, typ C.int, comments C.int) error {
  ctx := C.sass_new_file_context()
  defer C.sass_free_file_context(ctx)

  ctx.options.output_style = typ
  ctx.options.source_comments = comments
  ctx.input_path = C.CString(infile)

  ret := C.sass_compile_file(ctx)

  if ret != 0 || ctx.error_status != 0 {
    return errors.New(C.GoString(ctx.error_message))
  }

  out, err := os.Create(outfile)
  if err != nil {
    return err
  }
  out.Write([]byte(C.GoString(ctx.output_string)))
  out.Close()

  return nil
}

func translate(infile, outfile string) error {
  return compile(infile, outfile, C.SASS_STYLE_NESTED, 1)
}

func minify(infile, outfile string) error {
  return compile(infile, outfile, C.SASS_STYLE_COMPRESSED, 0)
}
