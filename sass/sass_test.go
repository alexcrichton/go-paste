package sass

import "github.com/alexcrichton/go-paste"
import "io/ioutil"
import "net/http"
import "net/http/httptest"
import "os"
import "path/filepath"
import "strings"
import "testing"

func check(t *testing.T, e error) {
  if e != nil {
    t.Fatal(e)
  }
}

func stubServer(t *testing.T, c bool) (paste.Server, string) {
  tmpdir, err := ioutil.TempDir(os.TempDir(), "paste")
  check(t, err)
  return paste.FileServer(paste.Config{Root: tmpdir, Compressed: c}), tmpdir
}

func stub(t *testing.T, compress bool) (*httptest.Server, string) {
  srv, wd := stubServer(t, compress)
  return httptest.NewServer(srv), wd
}

func stubFile(t *testing.T, wd, file, contents string) {
  f, err := os.Create(filepath.Join(wd, file))
  check(t, err)
  f.Write([]byte(contents))
  f.Close()
}

func TestSass(t *testing.T) {
  srv, wd := stub(t, true)
  defer os.RemoveAll(wd)
  defer srv.Close()
  stubFile(t, wd, "foo.css", "#foo {\nwidth: 100px;\n}")
  stubFile(t, wd, "bar.scss", "#foo {\n#bar { width: 100px;\n}\n}")

  resp, err := http.Get(srv.URL + "/foo.css")
  check(t, err)
  s, err := ioutil.ReadAll(resp.Body)
  check(t, err)
  if string(s) != "#foo{width:100px;}\n" {
    t.Errorf("wrong contents:\n%s", string(s))
  }

  /* Be sure that lookup of 'bar.css' finds the 'bar.scss' file */
  resp, err = http.Get(srv.URL + "/bar.css")
  check(t, err)
  s, err = ioutil.ReadAll(resp.Body)
  check(t, err)
  if string(s) != "#foo #bar{width:100px;}\n" {
    t.Errorf("wrong contents:\n%s", string(s))
  }
  if !strings.Contains(resp.Header.Get("Content-Type"), "text/css") {
    t.Errorf("wrong content type: %s", resp.Header.Get("Content-Type"))
  }
}

func TestSassUncompressed(t *testing.T) {
  srv, wd := stub(t, false)
  defer os.RemoveAll(wd)
  defer srv.Close()
  stubFile(t, wd, "bar.scss", "#foo {\n#bar { width: 100px;\n}\n}")

  resp, err := http.Get(srv.URL + "/bar.css")
  check(t, err)
  s, err := ioutil.ReadAll(resp.Body)
  check(t, err)
  if !strings.Contains(string(s), "#foo #bar") {
    t.Errorf("wrong contents:\n%s", string(s))
  }
}
