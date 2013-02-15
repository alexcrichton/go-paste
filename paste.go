package paste

import "net/http"
import "path"
import "path/filepath"
import "regexp"
import "sync"

var hashRegex = regexp.MustCompile(`(.*)-([a-f0-9]{32})(\.\w+)`)

type assetMeta struct {
  err     error
  Asset
  sync.Mutex
}

type Server struct {
  fsRoot string
  assets map[string]*assetMeta
  sync.Mutex
}

type Processor interface {
  Process(infile, outfile string) error
}

type ProcessorFunc func(infile, outfile string) error

var processors = make(map[string][]Processor)

func RegisterProcessor(p Processor, ext string) {
  prev, ok := processors[ext]
  if !ok {
    prev = make([]Processor, 0)
  }
  prev = append(prev, p)
  processors[ext] = prev
}

func FileServer(path string) *Server {
  return &Server{ fsRoot: path, assets: make(map[string]*assetMeta) }
}

func (p ProcessorFunc) Process(infile, outfile string) error {
  return p(infile, outfile)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  dir, file := path.Split(r.URL.Path)
  file, digest := findDigest(file)
  asset, err := s.asset(path.Join(dir, file))
  if err != nil || (digest != "" && digest != asset.Digest()) {
    http.NotFound(w, r)
    return
  }

  if s.etagMatches(w, r, asset) {
    return
  }

  headers := w.Header()
  if digest != "" {
    headers.Set("Cache-Control", "public, max-age=31536000")
  } else {
    headers.Set("Cache-Control", "public, must-revalidate")
  }
  http.ServeFile(w, r, asset.Pathname())
}

func (s *Server) etagMatches(w http.ResponseWriter, r *http.Request,
                             a Asset) bool {
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

func (s *Server) asset(logical string) (Asset, error) {
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

func (s *Server) buildAsset(logical string) (Asset, error) {
  return newStatic(logical, filepath.Join(s.fsRoot, logical))
}

func (s *Server) AssetPath(logical string, digest bool) (string, error) {
  asset, err := s.asset(logical)
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
