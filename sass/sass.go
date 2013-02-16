package sass
// +build cgo

// #cgo LDFLAGS: -lsass
// #include <sass_interface.h>
import "C"

import "errors"
import "github.com/alexcrichton/go-paste"
import "os"

func init() {
  paste.RegisterProcessor(paste.ProcessorFunc(compile), ".css")
  paste.RegisterAlias(".css", ".scss")
}

func compile(infile, outfile string) error {
  ctx := C.sass_new_file_context()
  defer C.sass_free_file_context(ctx)

  ctx.options.output_style = C.SASS_STYLE_COMPRESSED
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
