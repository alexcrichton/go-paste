package paste

import "time"

type Asset interface {
  // Returns the digest for this asset, some hex-encoded string
  Digest() string

  // Returns the pathname on the filesystem for the compiled version of this
  // asset, not the original source file.
  Pathname() string

  // Returns the last known modification time for this asset
  ModTime() time.Time

  // Returns the 'logical name' of the asset which may not resemble the actual
  // filename of this asset. The logical name is used when looking up
  // information about this asset
  LogicalName() string

  // Returns whether the asset's compiled version is stale with respect to the
  // non-compiled version and all it's dependencies
  Stale() bool
}
