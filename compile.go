package paste

import "compress/gzip"
import "io"
import "path/filepath"
import "os"

func (s *Server) Compile(dest string) error {
  err := filepath.Walk(s.fsRoot,
                       func(path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if info.IsDir() { return nil }

    /* If this file's extension is an alias for another, then we should use the
       alias instead of the actual extension in the output file */
    ext := filepath.Ext(path)
    alias := ext
    for a, possibilities := range aliases {
      for _, p := range possibilities {
        if p == ext {
          alias = a
        }
      }
    }

    logical := path[len(s.fsRoot) + 1 : len(path) - len(ext)] + alias
    asset, err := s.asset(logical)
    if err != nil { return err }

    dst := filepath.Join(dest, logical)
    ext = filepath.Ext(dst)
    digest := dst[:len(dst) - len(ext)] + "-" + asset.Digest() + ext
    os.MkdirAll(filepath.Dir(dst), 0755)

    /* foo.js */
    out, err := os.Create(dst)
    if err != nil { return err }
    defer out.Close()

    /* foo.js.gz */
    _outgz, err := os.Create(dst + ".gz")
    if err != nil { return err }
    defer _outgz.Close()
    outgz, err := gzip.NewWriterLevel(_outgz, gzip.BestCompression)
    if err != nil { return err }
    defer outgz.Close()

    /* foo-hexdigest.js */
    outdigest, err := os.Create(digest)
    if err != nil { return err }
    defer outdigest.Close()

    /* foo-hexdigest.js.gz */
    _outdigestgz, err := os.Create(digest + ".gz")
    if err != nil { return err }
    defer _outdigestgz.Close()
    outdigestgz, err := gzip.NewWriterLevel(_outdigestgz, gzip.BestCompression)
    if err != nil { return err }
    defer outdigestgz.Close()

    /* input file (the compiled asset) */
    _in, err := os.Open(asset.Pathname())
    if err != nil { return err }
    defer _in.Close()

    /* And finally, copy everything from the input */
    in := io.TeeReader(_in, out)
    in = io.TeeReader(in, outdigest)
    in = io.TeeReader(in, outgz)
    _, err = io.Copy(outdigestgz, in)
    return err
  })

  return err
}
