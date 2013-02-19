package image

import "os/exec"
import "github.com/alexcrichton/go-paste"

func init() {
  if hascmd("optipng") {
    paste.RegisterCompressor(paste.ProcessorFunc(optipng), ".png")
    paste.RegisterCompressor(paste.ProcessorFunc(optipng), ".gif")
    paste.RegisterCompressor(paste.ProcessorFunc(optipng), ".bmp")
    paste.RegisterCompressor(paste.ProcessorFunc(optipng), ".tiff")
  } else if hascmd("pngcrush", "-h") {
    paste.RegisterCompressor(paste.ProcessorFunc(pngcrush), ".png")
  }
}

func optipng(infile, outfile string) error {
  return exec.Command("optipng", "-clobber", "-out", outfile, infile).Run()
}

func pngcrush(infile, outfile string) error {
  return exec.Command("pngcrush", "-ow", infile, outfile).Run()
}
