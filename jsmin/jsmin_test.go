package jsmin

import "github.com/alexcrichton/go-paste"
import "net/http"
import "net/http/httptest"
import "io/ioutil"
import "os"
import "path/filepath"
import "strings"
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

func TestJsmin(t *testing.T) {
  srv, wd := stubServer(t)
  stubFile(t, wd, "foo.js", "var longname = 0x1;\nvar foo = longname;")

  resp, err := http.Get(srv.URL + "/foo.js")
  check(t, err)

  /* Should at least remove the newline... */
  s, err := ioutil.ReadAll(resp.Body)
  check(t, err)
  if !strings.Contains(string(s), "0x1;var") {
    t.Errorf("wrong contents:\n%s", string(s))
  }
  ctype := resp.Header.Get("Content-Type")
  if !strings.Contains(ctype, "application/javascript") {
    t.Errorf("wrong content type: %s", ctype)
  }
}
