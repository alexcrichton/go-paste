package paste

import "crypto/md5"
import "encoding/hex"
import "io"

func hexdigest(f io.Reader) string {
  hash := md5.New()
  io.Copy(hash, f)
  return hex.EncodeToString(hash.Sum(nil))
}
