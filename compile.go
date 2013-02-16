package paste

import "io"
import "path/filepath"
import "os"

func (s *Server) Compile(dest string) error {
  filepath.Walk(s.fsRoot, func(path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if info.IsDir() { return nil }

    logical := path[len(s.fsRoot) + 1:]
    asset, err := s.asset(logical)
    if err != nil { return err }

    dst := filepath.Join(dest, logical)
    ext := filepath.Ext(dst)
    digest := dst[:len(dst) - len(ext)] + "-" + asset.Digest() + ext
    os.MkdirAll(filepath.Dir(dst), 0755)
    out, err := os.Create(dst)
    if err != nil { return err }
    defer out.Close()

    outdigest, err := os.Create(digest)
    if err != nil { return err }
    defer outdigest.Close()

    in, err := os.Open(asset.Pathname())
    if err != nil { return err }
    defer in.Close()

    io.Copy(out, io.TeeReader(in, outdigest))

    return nil
  })

  return nil
}
