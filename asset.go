package paste

import "time"

type Asset interface {
  Digest()      string
  Pathname()    string
  ModTime()     time.Time
  LogicalName() string
  Stale()       bool
}
