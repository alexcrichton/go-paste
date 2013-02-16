package paste

import "io/ioutil"
import "net/http"
import "net/http/httptest"
import "os"
import "path/filepath"
import "strings"
import "testing"
import "time"

func check(t *testing.T, e error) {
  if e != nil {
    t.Fatal(e)
  }
}

func testEq(t *testing.T, a, b string) {
  if a != b {
    t.Errorf("expected %s, got %s", b, a)
  }
}

func TestFindDigest(t *testing.T) {
  a, b := findDigest("foo.js")
  testEq(t, a, "foo.js")
  testEq(t, b, "")
  a, b = findDigest("foo")
  testEq(t, a, "foo")
  testEq(t, b, "")
  a, b = findDigest("foo-bar.js")
  testEq(t, a, "foo-bar.js")
  testEq(t, b, "")
  a, b = findDigest("foo-ba455f38e701f688ace552f2d2cb69d3.js")
  testEq(t, a, "foo.js")
  testEq(t, b, "ba455f38e701f688ace552f2d2cb69d3")
  a, b = findDigest("foo-ba455f38e701f688ace552f2d2cb69dz.js")
  testEq(t, a, "foo-ba455f38e701f688ace552f2d2cb69dz.js")
  testEq(t, b, "")
}

func stubServer(t *testing.T) (*fileServer, string) {
  tmpdir, err := ioutil.TempDir(os.TempDir(), "paste")
  check(t, err)
  return FileServer(tmpdir).(*fileServer), tmpdir
}

func stub(t *testing.T) (*httptest.Server, string) {
  srv, dir := stubServer(t)
  return httptest.NewServer(srv), dir
}

func stubFile(t *testing.T, wd, file, contents string) {
  os.MkdirAll(filepath.Dir(filepath.Join(wd, file)), 0755)
  f, err := os.Create(filepath.Join(wd, file))
  check(t, err)
  f.Write([]byte(contents))
  f.Close()
}

func TestGetNonExist(t *testing.T) {
  srv, wd := stub(t)
  defer srv.Close()
  defer os.RemoveAll(wd)

  /* can't fetch a directory */
  resp, err := http.Get(srv.URL + "/")
  check(t, err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
  /* actual non-existent file */
  resp, err = http.Get(srv.URL + "/foo")
  check(t, err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
  /* make sure relative paths don't go through */
  resp, err = http.Get(srv.URL + "/../../../../../../../etc/hosts")
  check(t, err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
}

func ValidateHeaders(t *testing.T, resp *http.Response,
                     contents, hash string) string {
  if resp.StatusCode != http.StatusOK {
    t.Errorf("expected 200 return, got %d", resp.StatusCode)
  }
  if resp.Header.Get("Last-Modified") == "" {
    t.Errorf("expected non-empty last-modified header")
  }
  etag := resp.Header.Get("ETag")
  if hash != "" && etag != `"` + hash + `"` {
    t.Errorf("wrong etag %s", etag)
  }
  cache := resp.Header.Get("Cache-Control")
  if (hash != "" && !strings.HasPrefix(cache, "public, max-age=")) ||
     (hash == "" && cache != "public, must-revalidate") {
    t.Errorf("wrong cache-control '%s'", cache)
  }

  s, err := ioutil.ReadAll(resp.Body)
  check(t, err)
  if string(s) != contents {
    t.Errorf("wrong contents %s", string(s))
  }
  if etag == "" {
    return ""
  }
  return etag[1:len(etag)-1]
}

func TestGetExist(t *testing.T) {
  var resp *http.Response
  srv, wd := stub(t)
  defer srv.Close()
  defer os.RemoveAll(wd)

  stubFile(t, wd, "foo.js", "asdf")
  stubFile(t, wd, "foo.png", "foo")

  resp, err := http.Get(srv.URL + "/foo.js")
  check(t, err)
  tag := ValidateHeaders(t, resp, "asdf", "")
  resp, err = http.Get(srv.URL + "/foo-" + tag + ".js")
  check(t, err)
  ValidateHeaders(t, resp, "asdf", tag)
  resp, err = http.Get(srv.URL + "/foo.png")
  check(t, err)
  tag = ValidateHeaders(t, resp, "foo", "")
  resp, err = http.Get(srv.URL + "/foo-" + tag + ".png")
  check(t, err)
  ValidateHeaders(t, resp, "foo", tag)
}

func TestGetExistNotModified(t *testing.T) {
  fs, wd := stubServer(t, )
  srv := httptest.NewServer(fs)
  defer srv.Close()
  defer os.RemoveAll(wd)
  stubFile(t, wd, "foo.js", "asdf")
  a, err := fs.Asset("foo.js")
  check(t, err)
  digest := a.Digest()

  req, err := http.NewRequest("GET", srv.URL + "/foo.js", nil)
  check(t, err)
  req.Header.Set("If-None-Match", `"` + digest + `"`)
  resp, err := http.DefaultClient.Do(req)
  check(t, err)
  if resp.StatusCode != http.StatusNotModified {
    t.Errorf("expected 304 return, got %d", resp.StatusCode)
  }

  req, err = http.NewRequest("GET", srv.URL + "/foo.js", nil)
  check(t, err)
  ago := time.Now().UTC().Add(3 * time.Minute)
  req.Header.Set("If-Modified-Since", ago.Format(http.TimeFormat))
  resp, err = http.DefaultClient.Do(req)
  check(t, err)
  if resp.StatusCode != http.StatusNotModified {
    t.Errorf("expected 304 return, got %d", resp.StatusCode)
  }
}

func TestAssetPaths(t *testing.T) {
  srv, wd := stubServer(t)
  defer os.RemoveAll(wd)
  stubFile(t, wd, "foo.js", "asdf")
  a, err := srv.Asset("foo.js")
  check(t, err)
  digest := a.Digest()

  /* nonexistent file */
  _, err = srv.AssetPath("asdf", false)
  if err == nil {
    t.Errorf("expected an error")
  }

  /* non-digested file */
  ret, err := srv.AssetPath("foo.js", false)
  if err != nil {
    t.Errorf("got error: %s", err.Error())
  }
  testEq(t, ret, "/foo.js")

  /* digested file, yum */
  ret, err = srv.AssetPath("foo.js", true)
  if err != nil {
    t.Errorf("got error: %s", err.Error())
  }
  testEq(t, ret, "/foo-" + digest + ".js")
}
