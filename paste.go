package paste

import "net/http"
import "os"
import "path"
import "path/filepath"
import "regexp"
import "sync"
import "time"

var hashRegex = regexp.MustCompile(`(.*)-([a-f0-9]{32})(\.\w+)`)
var Version = "0.0.0"

type Manifest map[string]string

type assetMeta struct {
  err     error
  Asset
  sync.Mutex
}

type Server interface {
  http.Handler
  AssetPath(string, bool) (string, error)
  Compile(string) error
  Asset(string) (Asset, error)
}

type fileServer struct {
  fsRoot string
  tmpdir string
  assets map[string]*assetMeta
  sync.Mutex
  Version string
}

type Processor interface {
  Process(infile, outfile string) error
}

type ProcessorFunc func(infile, outfile string) error

var processors = make(map[string][]Processor)
var aliases = make(map[string][]string)

func RegisterProcessor(p Processor, ext string) {
  prev, ok := processors[ext]
  if !ok {
    prev = make([]Processor, 0)
  }
  prev = append(prev, p)
  processors[ext] = prev
}

func RegisterAlias(extension, alias string) {
  prev, ok := aliases[extension]
  if !ok {
    prev = make([]string, 0)
  }
  prev = append(prev, alias)
  aliases[extension] = prev
}

func FileServer(path string) Server {
  return &fileServer{ fsRoot: path, assets: make(map[string]*assetMeta),
                      tmpdir: "tmp" }
}

func (p ProcessorFunc) Process(infile, outfile string) error {
  return p(infile, outfile)
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  serveHTTP(s, w, r)
}

func serveHTTP(s Server, w http.ResponseWriter, r *http.Request) {
  dir, file := path.Split(r.URL.Path)
  file, digest := findDigest(file)
  asset, err := s.Asset(path.Join(dir, file))
  if err != nil || (digest != "" && digest != asset.Digest()) {
    http.NotFound(w, r)
    return
  }

  if etagMatches(w, r, asset) {
    return
  }

  headers := w.Header()
  if digest != "" {
    endoftime := time.Now().Add(31536000 * time.Second)
    headers.Set("Cache-Control", "public, max-age=31536000")
    headers.Set("Expires", endoftime.Format(http.TimeFormat))
  } else {
    headers.Set("Cache-Control", "public, must-revalidate")
  }
  http.ServeFile(w, r, asset.Pathname())
}

func etagMatches(w http.ResponseWriter, r *http.Request, a Asset) bool {
  tag := r.Header.Get("If-None-Match")
  if tag == etag(a) {
    w.WriteHeader(http.StatusNotModified)
    return true
  }
  w.Header().Set("ETag", etag(a))
  return false
}

func etag(a Asset) string {
  return `"` + a.Digest() + `"`
}

func findDigest(file string) (string, string) {
  matches := hashRegex.FindStringSubmatch(file)
  if len(matches) == 0 {
    return file, ""
  }
  return matches[1] + matches[3], matches[2]
}

func (s *fileServer) Asset(logical string) (Asset, error) {
  logical = path.Clean("/" + logical)
  s.Lock()
  ret, ok := s.assets[logical]
  if !ok {
    ret = &assetMeta{}
    s.assets[logical] = ret
  }
  ret.Lock()
  defer ret.Unlock()
  s.Unlock()
  if ret.err != nil {
    return nil, ret.err
  } else if ret.Asset == nil || ret.Stale() {
    a, err := s.buildAsset(logical)
    if err == nil {
      ret.Asset = a
    } else {
      s.Lock()
      delete(s.assets, logical)
      s.Unlock()
      ret.err = err
      return nil, err
    }
  }
  return ret, nil
}

func (s *fileServer) buildAsset(logical string) (Asset, error) {
  pathname, err := s.resolve(logical)
  if err != nil {
    return nil, err
  }
  _, ok := processors[path.Ext(logical)]
  if ok {
    return newProcessed(s, logical, pathname)
  }
  return newStatic(s, logical, pathname)
}

func (s *fileServer) resolve(logical string) (string, error) {
  try := filepath.Join(s.fsRoot, logical)
  _, err := os.Stat(try)
  if err == nil {
    return try, nil
  }
  ext := filepath.Ext(logical)
  candidates, ok := aliases[ext]
  if ok {
    for _, cand := range candidates {
      try = filepath.Join(s.fsRoot, logical[:len(logical) - len(ext)] + cand)
      _, err = os.Stat(try)
      if err == nil {
        return try, nil
      }
    }
  }
  return "", err
}

func (s *fileServer) AssetPath(logical string, digest bool) (string, error) {
  asset, err := s.Asset(logical)
  if err != nil {
    return "", err
  }
  if digest {
    dir, file := path.Split(asset.LogicalName())
    ext := path.Ext(file)
    return dir + file[:len(file) - len(ext)] + "-" + asset.Digest() + ext, nil
  }
  return asset.LogicalName(), nil
}
