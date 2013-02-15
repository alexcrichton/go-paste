package paste

import "crypto/md5"
import "encoding/hex"
import "io"
import "os"

func hexdigest(srv *Server, f io.Reader) string {
  hash := md5.New()
  hash.Write([]byte(Version))
  hash.Write([]byte(srv.Version))
  io.Copy(hash, f)
  return hex.EncodeToString(hash.Sum(nil))
}

func hexdigestString(srv *Server, s string) string {
  hash := md5.New()
  hash.Write([]byte(Version))
  hash.Write([]byte(srv.Version))
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
