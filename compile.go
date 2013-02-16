package paste

import "compress/gzip"
import "io"
import "os"
import "path/filepath"
import "runtime"
import "sync"

func (s *Server) Compile(dest string) error {
  /* Compiling takes awhile, parallelize! */
  paths := make(chan string)
  var err error
  var wg sync.WaitGroup

  for i := 0; i < runtime.NumCPU(); i++ {
    go func() {
      for path := range paths {
        myerr := s.compileAsset(dest, path)
        if myerr != nil {
          err = myerr
        }
      }
      wg.Done()
    }()
    wg.Add(1)
  }

  myerr := filepath.Walk(s.fsRoot,
                         func(path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if info.IsDir() { return nil }
    paths <- path
    return nil
  })

  close(paths)
  wg.Wait()

  /* Doesn't really matter what error to return as long as some error is
     returned if there was an error somewhere */
  if myerr != nil {
    return myerr
  }
  return err
}

func (s *Server) compileAsset(dest, path string) error {
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
}
