package paste

import "errors"
import "os"
import "time"

type staticAsset struct {
  digest string
  pathname string
  mtime time.Time
  logical string
}

func (s *staticAsset) Digest() string { return s.digest }
func (s *staticAsset) Pathname() string { return s.pathname }
func (s *staticAsset) ModTime() time.Time { return s.mtime }
func (s *staticAsset) LogicalName() string { return s.logical }

func (s *staticAsset) Stale() bool {
  /* If the file doesn't exist, we're definitely stale */
  f, err := os.Open(s.pathname)
  if err != nil { return true }
  defer f.Close()
  stat, err := f.Stat()
  if err != nil { return true }

  /* If the file hasn't been modified since we last looked at it, it's stale */
  if stat.ModTime().Before(s.mtime) {
    return false
  }

  /* Same contents? not stale */
  if hexdigest(f) == s.digest {
    return false
  }

  /* Otherwise we're stale */
  return true
}

func newStatic(logical, path string) (*staticAsset, error) {
  asset := &staticAsset { pathname: path, logical: logical }
  f, err := os.Open(path)
  if err != nil { return nil, err }
  defer f.Close()

  stat, err := f.Stat()
  if err != nil { return nil, err }
  if stat.IsDir() {
    return nil, errors.New("cannot serve a directory")
  }
  asset.mtime = stat.ModTime()

  asset.digest = hexdigest(f)

  return asset, nil
}
