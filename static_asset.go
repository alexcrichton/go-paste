package paste

import "errors"
import "time"

type staticAsset struct {
  digest string
  pathname string
  mtime time.Time
  logical string
  srv *fileServer
}

func (s *staticAsset) Digest() string { return s.digest }
func (s *staticAsset) Pathname() string { return s.pathname }
func (s *staticAsset) ModTime() time.Time { return s.mtime }
func (s *staticAsset) LogicalName() string { return s.logical }

func (s *staticAsset) Stale() bool {
  /* If the file doesn't exist, we're definitely stale */
  f, stat, err := openStat(s.pathname)
  if err != nil { return true }
  defer f.Close()

  /* If the file hasn't been modified, it's not stale */
  if stat.ModTime().Before(s.mtime) || stat.ModTime().Equal(s.mtime) {
    return false
  }

  /* Same contents? not stale */
  if hexdigest(s.srv, f) == s.digest {
    return false
  }

  /* Otherwise we're stale */
  return true
}

func newStatic(s *fileServer, logical, path string) (*staticAsset, error) {
  asset := &staticAsset { pathname: path, logical: logical, srv: s }
  f, stat, err := openStat(path)
  if err != nil {
    return nil, err
  } else if stat.IsDir() {
    return nil, errors.New("cannot serve a directory")
  }

  asset.mtime = stat.ModTime()
  asset.digest = hexdigest(s, f)

  return asset, nil
}
