package paste

import "io/ioutil"
import "os"
import "testing"

func TestCompile(t *testing.T) {
  srv, wd := stubServer(t)
  defer os.RemoveAll(wd)
  dst, err := ioutil.TempDir("", "paste")
  check(t, err)

  stubFile(t, wd, "foo/foo.js", "bar")
  stubFile(t, wd, "foo.css", "bar")

  srv.Compile(dst)

  contains := func(f, contents string) {
    file, err := os.Open(dst + f)
    check(t, err)
    defer file.Close()
    s, err := ioutil.ReadAll(file)
    check(t, err)
    if string(s) != contents {
      t.Errorf("wrong contents:\n%s", string(s))
    }
  }

  contains("/foo/foo.js", "bar")
  contains("/foo.css", "bar")

  asset, err := srv.asset("foo/foo.js")
  check(t, err)

  contains("/foo/foo-" + asset.Digest() + ".js", "bar")
}
