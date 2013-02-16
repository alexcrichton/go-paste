package paste

import "crypto/md5"
import "encoding/hex"
import "io"
import "os"

func hexdigest(srv *fileServer, f io.Reader) string {
  hash := md5.New()
  hash.Write([]byte(Version))
  hash.Write([]byte(srv.config.Version))
  io.Copy(hash, f)
  return hex.EncodeToString(hash.Sum(nil))
}

func hexdigestString(srv *fileServer, s string) string {
  hash := md5.New()
  hash.Write([]byte(Version))
  hash.Write([]byte(srv.config.Version))
  hash.Write([]byte(s))
  return hex.EncodeToString(hash.Sum(nil))
}

func openStat(path string) (*os.File, os.FileInfo, error) {
  f, err := os.Open(path)
  if err != nil { return nil, nil, err }
  stat, err := f.Stat()
  if err != nil {
    f.Close()
    return nil, nil, err
  }
  return f, stat, nil
}
