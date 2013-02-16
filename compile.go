package paste

import "compress/gzip"
import "encoding/json"
import "errors"
import "fmt"
import "io"
import "net/http"
import "os"
import "path"
import "path/filepath"
import "runtime"
import "sync"
import "time"

type compiledServer struct {
  root string
  precompiled map[string]*precompiledAsset
}

type precompiledAsset struct {
  path    string
  logical string
  digest  string
}

func (s *fileServer) Compile(dest string) error {
  /* Compiling takes awhile, parallelize! */
  paths := make(chan string)
  manifest := make(Manifest)
  var err error
  var wg sync.WaitGroup

  for i := 0; i < runtime.NumCPU(); i++ {
    go func() {
      for path := range paths {
        myerr := s.compileAsset(dest, path, manifest)
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
  } else if err != nil {
    return err
  }

  mfile, err := os.Create(filepath.Join(dest, "manifest.json"))
  if err != nil { return err }
  defer mfile.Close()
  enc := json.NewEncoder(mfile)
  return enc.Encode(manifest)
}

func (s *fileServer) compileAsset(dest, path string, m Manifest) error {
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

  /* Actual compilation of the asset itself */
  logical := path[len(s.fsRoot) + 1 : len(path) - len(ext)] + alias
  asset, err := s.Asset(logical)
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
  if err != nil { return err }

  s.Lock()
  m["/" + logical] = asset.Digest()
  s.Unlock()
  return nil
}

func CompiledFileServer(root string) (Server, error) {
  srv := &compiledServer { root: root,
                           precompiled: make(map[string]*precompiledAsset) }

  manifest := make(Manifest)
  mfile, err := os.Open(filepath.Join(root, "manifest.json"))
  if err != nil {
    return nil, err
  }
  defer mfile.Close()
  enc := json.NewDecoder(mfile)
  err = enc.Decode(&manifest)
  if err != nil { return nil, err }

  for path, digest := range manifest {
    srv.precompiled[path] = &precompiledAsset{ logical: path,
                                               path: filepath.Join(root, path),
                                               digest: digest }
  }

  return srv, nil
}

func (c *compiledServer) AssetPath(logical string, digest bool) (string, error){
  asset, err := c.Asset(logical)
  if err != nil {
    return "", err
  } else if digest {
    logical = asset.LogicalName()
    ext := path.Ext(logical)
    return logical[:len(logical) - len(ext)] + "-" + asset.Digest() + ext, nil
  }
  return asset.LogicalName(), nil
}

func (c *compiledServer) Compile(dst string) error {
  return errors.New("Compiled server can't compile assets again")
}

func (s *compiledServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  serveHTTP(s, w, r)
}

func (s *compiledServer) Asset(logical string) (Asset, error) {
  asset, ok := s.precompiled[path.Clean("/" + logical)]
  if ok {
    return asset, nil
  }
  return nil, errors.New(fmt.Sprintf("asset not precompiled: %s", logical))
}

func (a *precompiledAsset) Digest() string      { return a.digest }
func (a *precompiledAsset) Pathname() string    { return a.path }
func (a *precompiledAsset) LogicalName() string { return a.logical }
func (a *precompiledAsset) Stale() bool         { return false }
func (a *precompiledAsset) ModTime() time.Time  { return time.Now() }
