package paste

import "bufio"
import "io"
import "os"
import "time"
import "path/filepath"
import "strings"

type processedAsset struct {
  static *staticAsset
  dependencies []Asset

  digest   string
  mtime    time.Time
  pathname string
}

func (s *processedAsset) Digest() string      { return s.digest }
func (s *processedAsset) Pathname() string    { return s.pathname }
func (s *processedAsset) ModTime() time.Time  { return s.mtime }
func (s *processedAsset) LogicalName() string { return s.static.logical }

func (s *processedAsset) Stale() bool {
  if s.static.Stale() { return true }
  for _, d := range s.dependencies {
    if d.Stale() {
      return true
    }
  }
  return false
}

func newProcessed(s *Server, logical, path string) (Asset, error) {
  static, err := newStatic(logical, path)
  if err != nil {
    return nil, err
  }

  asset := &processedAsset{static: static, dependencies: make([]Asset, 0)}
  paths, err := asset.requiredPaths()
  if err != nil {
    return nil, err
  }

  digest := asset.static.digest
  asset.mtime = asset.static.mtime

  for _, dep := range paths {
    d, err := s.asset(dep)
    if err != nil {
      return nil, err
    }
    asset.dependencies = append(asset.dependencies, d)

    digest += d.Digest()
    if d.ModTime().After(asset.mtime) {
      asset.mtime = d.ModTime()
    }
  }
  asset.digest = hexdigestString(digest)

  compiled := filepath.Join(filepath.Join(s.fsRoot, s.tmpdir), logical)
  os.MkdirAll(filepath.Dir(compiled), 0755)
  file, err := os.Create(compiled)
  if err != nil {
    return nil, err
  }
  defer file.Close()
  asset.pathname = file.Name()

  copyFile(file, asset.static.pathname)
  for _, dep := range asset.dependencies {
    copyFile(file, dep.Pathname())
  }

  return asset, nil
}

func (s *processedAsset) requiredPaths() ([]string, error) {
  f, err := os.Open(s.static.pathname)
  if err != nil {
    return nil, err
  }
  defer f.Close()
  buf := bufio.NewReader(f)
  paths := make([]string, 0)
  for {
    s, err := buf.ReadString('\n')
    if err == io.EOF {
      break
    } else if err != nil {
      return nil, err
    }
    if strings.TrimSpace(s) != "" && !strings.HasPrefix(s, "//") { break }
    if !strings.HasPrefix(s, "//= require") {
      paths = append(paths, s[11:])
    }
  }
  return paths, nil
}

func copyFile(w io.Writer, path string) {
  f, err := os.Open(path)
  if err != nil { panic(err) }
  io.Copy(w, f)
  f.Close()
}