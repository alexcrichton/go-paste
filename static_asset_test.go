package paste

import "testing"
import "os"
import "io/ioutil"
import "time"

func TestStale(t *testing.T) {
  println("yay")
  f, err := ioutil.TempFile(os.TempDir(), "paste")
  check(err)
  name := f.Name()
  defer os.Remove(name)
  f.Write([]byte("foo"))
  f.Close()

  asset, err := newStatic("bar", name)
  if err != nil {
    t.Errorf("ran into error: %s", err.Error())
  }
  if asset.Stale() {
    t.Errorf("shouldn't be stale when just created")
  }
  past := time.Now().Add(-5 * time.Second)
  check(os.Chtimes(name, past, past))
  if asset.Stale() {
    t.Errorf("shouldn't be stale with old contents")
  }

  check(ioutil.WriteFile(name, []byte("bar"), 0644))
  if !asset.Stale() {
    t.Errorf("should be stale now with new contents")
  }
}
