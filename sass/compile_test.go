package sass

import "io/ioutil"
import "os"
import "testing"

func TestCompile(t *testing.T) {
  srv, wd := stubServer(t)
  defer os.RemoveAll(wd)
  dst, err := ioutil.TempDir("", "paste")
  check(t, err)

  stubFile(t, wd, "foo.scss", "#main {}")

  srv.Compile(dst)

  f, err := os.Open(dst + "/foo.css")
  check(t, err)
  f.Close()
}
