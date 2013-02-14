package paste

// import "github.com/suapapa/go_sass"
import "net/http"
import "path"
import "regexp"
import "sync"
import "path/filepath"

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

func FileServer(path string) *Server {
  return &Server{ fsRoot: path, assets: make(map[string]*assetMeta) }
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  dir, file := path.Split(path.Clean("/" + r.URL.Path))
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

func (s *Server) asset(upath string) (Asset, error) {
  s.Lock()
  ret, ok := s.assets[upath]
  if !ok {
    ret = &assetMeta{}
    s.assets[upath] = ret
  }
  ret.Lock()
  defer ret.Unlock()
  s.Unlock()
  if ret.err != nil {
    return nil, ret.err
  } else if ret.Asset == nil || ret.Stale() {
    a, err := s.buildAsset(upath)
    if err == nil {
      ret.Asset = a
    } else {
      /* TODO: remove from 'assets' map */
      ret.err = err
      return nil, err
    }
  }
  return ret, nil
}

func (s *Server) buildAsset(upath string) (Asset, error) {
  return newStatic(upath, filepath.Join(s.fsRoot, upath))
}
