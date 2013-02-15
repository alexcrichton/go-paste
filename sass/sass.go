package sass

// +build cgo

import "github.com/suapapa/go_sass"
import "github.com/alexcrichton/go-paste"
import "os"

type Compiler struct {
  sass.Compiler
}

func init() {
  c := &Compiler{}
  c.OutputStyle = sass.STYLE_COMPRESSED
  c.SourceComments = true

  paste.RegisterProcessor(c, ".css")
}

func (c *Compiler) Process(infile, outfile string) error {
  out, err := c.CompileFile(infile)
  if err != nil {
    return err
  }

  f, err := os.Create(outfile)
  if err != nil {
    return err
  }
  f.Write([]byte(out))
  f.Close()
  return nil
}
