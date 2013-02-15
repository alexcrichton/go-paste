package paste

import "testing"
import "os"
import "io/ioutil"
import "time"
import "path/filepath"

func init() {
  RegisterProcessor(ProcessorFunc(func(infile, outfile string) error {
    return nil
  }), ".js")
}

func TestProcessedSingleFile(t *testing.T) {
  srv, wd := stubServer(t)
  stubFile(t, wd, "foo.js", "bar")
  file := filepath.Join(wd, "foo.js")

  asset, err := newProcessed(srv, "foo.js", file)
  if err != nil {
    t.Fatalf("ran into error: %s", err.Error())
  }

  bits, err := ioutil.ReadFile(asset.Pathname())
  check(t, err)
  if string(bits) != "bar" {
    t.Errorf("should contain 'bar'")
  }
  if asset.Stale() {
    t.Errorf("shouldn't be stale when just created")
  }
  future := time.Now().Add(5 * time.Second)
  check(t, os.Chtimes(file, future, future))
  if asset.Stale() {
    t.Errorf("shouldn't be stale with old contents")
  }

  check(t, ioutil.WriteFile(file, []byte("foo"), 0644))
  check(t, os.Chtimes(file, future, future))
  if !asset.Stale() {
    t.Errorf("should be stale now with new contents")
  }
}

func TestProcessedConcatenates(t *testing.T) {
  srv, wd := stubServer(t)
  stubFile(t, wd, "foo.js", "//= require bar\nfoo")
  stubFile(t, wd, "bar.js", "//= require baz.js\nbar")
  stubFile(t, wd, "baz.js", "baz")
  file := filepath.Join(wd, "foo.js")

  asset, err := newProcessed(srv, "foo.js", file)
  if err != nil {
    t.Fatalf("ran into error: %s", err.Error())
  }
  bits, err := ioutil.ReadFile(asset.Pathname())
  check(t, err)
  if string(bits) != "baz\n//= require baz.js\nbar\n//= require bar\nfoo" {
    t.Errorf("wrong contents:\n%s", string(bits))
  }
  if asset.Stale() {
    t.Errorf("shouldn't be stale")
  }
  future := time.Now().Add(5 * time.Second)
  check(t, os.Chtimes(filepath.Join(wd, "bar.js"), future, future))
  if asset.Stale() {
    t.Errorf("shouldn't be stale with same contents")
  }

  check(t, ioutil.WriteFile(filepath.Join(wd, "baz.js"), []byte("baz2"), 0644))
  check(t, os.Chtimes(filepath.Join(wd, "baz.js"), future, future))
  if !asset.Stale() {
    t.Errorf("should be stale now with new contents")
  }
}
