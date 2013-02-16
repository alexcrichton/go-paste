// Package for applying 'jsmin' to JS files for paste.
//
// When imported, this package will automatically register a processor for all
// JS files which runs the jsmin program over javascript.
package jsmin

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
