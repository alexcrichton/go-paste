package process

import "bitbucket.org/maxhauser/jsmin"
import "github.com/alexcrichton/go-paste"
import "os"

func init() {
  paste.RegisterProcessor(paste.ProcessorFunc(minify), ".js")
}

func minify(infile, outfile string) error {
  in, err := os.Open(infile)
  if err != nil { return err }
  defer in.Close()

  out, err := os.Create(outfile)
  if err != nil { return err }
  defer out.Close()

  jsmin.Run(in, out)
  return nil
}
