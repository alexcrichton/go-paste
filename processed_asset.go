package paste

import "bufio"
import "io"
import "io/ioutil"
import "os"
import "path"
import "path/filepath"
import "regexp"
import "strings"
import "time"

type processedAsset struct {
  static *staticAsset
  dependencies []Asset

  digest   string
  mtime    time.Time
  pathname string
}

var jsRequires = regexp.MustCompile(`^//=\s*require\s+(\S+)`)

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
  static, err := newStatic(s, logical, path)
  if err != nil {
    return nil, err
  }

  asset := &processedAsset{static: static, dependencies: make([]Asset, 0)}
  paths, err := asset.requiredPaths(jsRequires)
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
  asset.digest = hexdigestString(s, digest)

  compiled := filepath.Join(filepath.Join(s.fsRoot, s.tmpdir), asset.digest)
  compiled += filepath.Ext(logical)
  os.MkdirAll(filepath.Dir(compiled), 0755)
  file, err := os.Create(compiled)
  if err != nil {
    return nil, err
  }
  asset.pathname = file.Name()

  for _, dep := range asset.dependencies {
    copyFile(file, dep.Pathname())
    file.Write([]byte{'\n'})
  }
  copyFile(file, asset.static.pathname)
  file.Close()

  arr, ok := processors[filepath.Ext(asset.static.logical)]
  if ok {
    tmpdir, err := ioutil.TempDir("", "paste")
    defer os.RemoveAll(tmpdir)
    if err != nil {
      return nil, err
    }
    for _, p := range arr {
      err = p.Process(asset.Pathname(), filepath.Join(tmpdir, "foo"))
      os.Rename(filepath.Join(tmpdir, "foo"), asset.Pathname())
      if err != nil {
        return nil, err
      }
    }
  }

  return asset, nil
}

func (s *processedAsset) requiredPaths(rx *regexp.Regexp) ([]string, error) {
  f, err := os.Open(s.static.pathname)
  if err != nil {
    return nil, err
  }
  defer f.Close()
  buf := bufio.NewReader(f)
  paths := make([]string, 0)
  for {
    line, err := buf.ReadString('\n')
    if err == io.EOF {
      break
    } else if err != nil {
      return nil, err
    }
    if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "//") { break }
    matches := rx.FindStringSubmatch(line)
    if len(matches) > 1 {
      match := matches[1]
      if path.Ext(match) == "" {
        match += path.Ext(s.static.logical)
      }
      paths = append(paths, match)
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
