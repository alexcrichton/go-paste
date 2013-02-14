package paste

import "time"
import "os"
import "crypto/md5"
import "io"
import "encoding/hex"

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

func (s *staticAsset) Stale() bool { return true }

func newStatic(logical, path string) (*staticAsset, error) {
  asset := &staticAsset { pathname: path, logical: logical }
  f, err := os.Open(path)
  if err != nil { return nil, err }
  defer f.Close()

  stat, err := f.Stat()
  if err != nil { return nil, err }
  asset.mtime = stat.ModTime()

  hash := md5.New()
  _, err = io.Copy(hash, f)
  if err != nil { return nil, err }
  asset.digest = hex.EncodeToString(hash.Sum(nil))

  return asset, nil
}

