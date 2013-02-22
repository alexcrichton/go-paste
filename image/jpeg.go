package image

import "os/exec"
import "github.com/alexcrichton/go-paste"

func init() {
  if hascmd("jpegoptim", "-V") {
    paste.RegisterCompressor(paste.ProcessorFunc(jpegoptim), ".jpg")
    paste.RegisterCompressor(paste.ProcessorFunc(jpegoptim), ".jpeg")
  }
}

func jpegoptim(infile, outfile string) error {
  err := cp(infile, outfile)
  if err != nil {
    return err
  }
  return exec.Command("jpegoptim", "--strip-all", outfile).Run()
}
