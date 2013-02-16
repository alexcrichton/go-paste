package paste

import "compress/gzip"
import "io"
import "io/ioutil"
import "os"
import "testing"
import "strings"

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
    var input io.Reader = file
    if strings.HasSuffix(f, ".gz") {
      input, err = gzip.NewReader(file)
      check(t, err)
    }
    s, err := ioutil.ReadAll(input)
    check(t, err)
    if string(s) != contents {
      t.Errorf("wrong contents:\n%s", string(s))
    }
  }

  contains("/foo/foo.js", "bar")
  contains("/foo/foo.js.gz", "bar")
  contains("/foo.css", "bar")
  contains("/foo.css.gz", "bar")

  asset, err := srv.asset("foo/foo.js")
  check(t, err)

  contains("/foo/foo-" + asset.Digest() + ".js", "bar")
  contains("/foo/foo-" + asset.Digest() + ".js.gz", "bar")
}
