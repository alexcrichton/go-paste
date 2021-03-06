package paste

import "testing"
import "os"
import "io/ioutil"
import "time"

func TestStaticStale(t *testing.T) {
  srv, wd := stubServer(t)
  defer os.RemoveAll(wd)
  f, err := ioutil.TempFile(wd, "paste")
  check(t, err)
  name := f.Name()
  defer os.Remove(name)
  f.Write([]byte("foo"))
  f.Close()

  asset, err := newStatic(srv, "bar", name)
  if err != nil {
    t.Errorf("ran into error: %s", err.Error())
  }
  if asset.Stale() {
    t.Errorf("shouldn't be stale when just created")
  }
  future := time.Now().Add(5 * time.Second)
  check(t, os.Chtimes(name, future, future))
  if asset.Stale() {
    t.Errorf("shouldn't be stale with old contents")
  }

  check(t, ioutil.WriteFile(name, []byte("bar"), 0644))
  check(t, os.Chtimes(name, future, future))
  if !asset.Stale() {
    t.Errorf("should be stale now with new contents")
  }
}
