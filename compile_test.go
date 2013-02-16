package paste

import "compress/gzip"
import "io"
import "io/ioutil"
import "net/http"
import "net/http/httptest"
import "os"
import "testing"
import "strings"

func stubCompiledServer(t *testing.T) (*compiledServer, string) {
  srv, wd := stubServer(t)
  defer os.RemoveAll(wd)
  dst, err := ioutil.TempDir("", "paste")
  check(t, err)

  stubFile(t, wd, "foo/foo.js", "bar1")
  stubFile(t, wd, "foo.css", "bar2")
  stubFile(t, wd, "foo.png", "bar3")

  check(t, srv.Compile(dst))
  csrv, err := CompiledFileServer(dst)
  check(t, err)
  return csrv.(*compiledServer), dst
}

func TestCompile(t *testing.T) {
  srv, dst := stubCompiledServer(t)
  defer os.RemoveAll(dst)

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

  contains("/foo/foo.js", "bar1")
  contains("/foo/foo.js.gz", "bar1")
  contains("/foo.css", "bar2")
  contains("/foo.css.gz", "bar2")
  println(dst)

  asset, err := srv.Asset("foo/foo.js")
  check(t, err)

  contains("/foo/foo-" + asset.Digest() + ".js", "bar1")
  contains("/foo/foo-" + asset.Digest() + ".js.gz", "bar1")
}

func TestCompiledServer(t *testing.T) {
  srv, dst := stubCompiledServer(t)
  defer os.RemoveAll(dst)
  hsrv := httptest.NewServer(srv)
  defer hsrv.Close()

  resp, err := http.Get(hsrv.URL + "/foo/foo.js")
  check(t, err)
  tag := ValidateHeaders(t, resp, "bar1", "")
  resp, err = http.Get(hsrv.URL + "/foo/foo-" + tag + ".js")
  check(t, err)
  ValidateHeaders(t, resp, "bar1", tag)
  resp, err = http.Get(hsrv.URL + "/foo.png")
  check(t, err)
  tag = ValidateHeaders(t, resp, "bar3", "")
  resp, err = http.Get(hsrv.URL + "/foo-" + tag + ".png")
  check(t, err)
  ValidateHeaders(t, resp, "bar3", tag)
}

func TestCompiledServerAssetPath(t *testing.T) {
  srv, dst := stubCompiledServer(t)
  defer os.RemoveAll(dst)

  check := func(path string, digest bool, err bool) {
    _, e := srv.AssetPath(path, digest)
    if err && e == nil {
      t.Errorf("expected an error for %s", path)
    } else if !err && e != nil {
      t.Errorf("unexpected error %s", e.Error())
    }
  }

  check("foo.js", false, true)
  check("foo.js", true, true)
  check("foo/foo.js", false, false)
  check("foo/foo.js", true, false)
  check("/foo/foo.js", false, false)
  check("/foo/foo.js", true, false)
}
