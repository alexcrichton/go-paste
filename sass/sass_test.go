package sass

import "github.com/alexcrichton/go-paste"
import "net/http"
import "net/http/httptest"
import "io/ioutil"
import "os"
import "path/filepath"
import "testing"

func check(t *testing.T, e error) {
  if e != nil {
    t.Fatal(e)
  }
}

func stubServer(t *testing.T) (*httptest.Server, string) {
  tmpdir, err := ioutil.TempDir(os.TempDir(), "paste")
  check(t, err)
  return httptest.NewServer(paste.FileServer(tmpdir)), tmpdir
}

func stubFile(t *testing.T, wd, file, contents string) {
  f, err := os.Create(filepath.Join(wd, file))
  check(t, err)
  f.Write([]byte(contents))
  f.Close()
}

func TestSass(t *testing.T) {
  srv, wd := stubServer(t)
  stubFile(t, wd, "foo.css", "#foo {\nwidth: 100px;\n}")
  stubFile(t, wd, "bar.scss", "#foo {\nwidth: 100px;\n}")
  compressed := "#foo{width:100px;}\n"

  resp, err := http.Get(srv.URL + "/foo.css")
  check(t, err)
  s, err := ioutil.ReadAll(resp.Body)
  check(t, err)
  if string(s) != compressed {
    t.Errorf("wrong contents:\n%s", string(s))
  }

  /* Be sure that lookup of 'bar.css' finds the 'bar.scss' file */
  resp, err = http.Get(srv.URL + "/bar.css")
  check(t, err)
  s, err = ioutil.ReadAll(resp.Body)
  check(t, err)
  if string(s) != compressed {
    t.Errorf("wrong contents:\n%s", string(s))
  }
}