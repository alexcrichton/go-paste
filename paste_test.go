package paste

import "io/ioutil"
import "net/http"
import "net/http/httptest"
import "os"
import "path/filepath"
import "strings"
import "testing"
import "time"

func check(e error) {
  if e != nil {
    panic(e)
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

func stubServer() (*Server, string) {
  cwd, err := os.Getwd()
  check(err)
  tmpdir, err := ioutil.TempDir(cwd, "paste")
  check(err)
  return FileServer(tmpdir), tmpdir
}

func stub() (*httptest.Server, string) {
  srv, dir := stubServer()
  return httptest.NewServer(srv), dir
}

func stubFile(wd, file, contents string) {
  f, err := os.Create(filepath.Join(wd, file))
  check(err)
  f.Write([]byte(contents))
  f.Close()
}

func TestGetNonExist(t *testing.T) {
  srv, wd := stub()
  defer srv.Close()
  defer os.RemoveAll(wd)

  /* can't fetch a directory */
  resp, err := http.Get(srv.URL + "/")
  check(err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
  /* actual non-existent file */
  resp, err = http.Get(srv.URL + "/foo")
  check(err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
  /* make sure relative paths don't go through */
  resp, err = http.Get(srv.URL + "/../../../../../../../etc/hosts")
  check(err)
  if resp.StatusCode != http.StatusNotFound {
    t.Errorf("expected 404 return, got %d", resp.StatusCode)
  }
}

func TestGetExist(t *testing.T) {
  var resp *http.Response
  srv, wd := stub()
  defer srv.Close()
  defer os.RemoveAll(wd)

  stubFile(wd, "foo.js", "asdf")

  ensure := func(requestedHash bool) {
    if resp.StatusCode != http.StatusOK {
      t.Errorf("expected 200 return, got %d", resp.StatusCode)
    }
    if resp.Header.Get("Last-Modified") == "" {
      t.Errorf("expected non-empty last-modified header")
    }
    if resp.Header.Get("ETag") != `"912ec803b2ce49e4a541068d495ab570"` {
      t.Errorf("wrong etag %s", resp.Header.Get("ETag"))
    }
    cache := resp.Header.Get("Cache-Control")
    if (requestedHash && !strings.HasPrefix(cache, "public, max-age=")) ||
       (!requestedHash && cache != "public, must-revalidate") {
      t.Errorf("wrong cache-control '%s'", cache)
    }

    s, err := ioutil.ReadAll(resp.Body)
    check(err)
    if string(s) != "asdf" {
      t.Errorf("wrong contents %s", string(s))
    }
  };

  resp, err := http.Get(srv.URL + "/foo.js")
  check(err)
  ensure(false)
  resp, err = http.Get(srv.URL + "/foo-912ec803b2ce49e4a541068d495ab570.js")
  check(err)
  ensure(true)
}

func TestGetExistNotModified(t *testing.T) {
  srv, wd := stub()
  defer srv.Close()
  defer os.RemoveAll(wd)
  stubFile(wd, "foo.js", "asdf")

  req, err := http.NewRequest("GET", srv.URL + "/foo.js", nil)
  check(err)
  req.Header.Set("If-None-Match", `"912ec803b2ce49e4a541068d495ab570"`)
  resp, err := http.DefaultClient.Do(req)
  check(err)
  if resp.StatusCode != http.StatusNotModified {
    t.Errorf("expected 304 return, got %d", resp.StatusCode)
  }

  req, err = http.NewRequest("GET", srv.URL + "/foo.js", nil)
  check(err)
  ago := time.Now().UTC().Add(3 * time.Minute)
  req.Header.Set("If-Modified-Since", ago.Format(http.TimeFormat))
  resp, err = http.DefaultClient.Do(req)
  check(err)
  if resp.StatusCode != http.StatusNotModified {
    t.Errorf("expected 304 return, got %d", resp.StatusCode)
  }
}

func TestAssetPaths(t *testing.T) {
  srv, wd := stubServer()
  defer os.RemoveAll(wd)
  stubFile(wd, "foo.js", "asdf")

  /* nonexistent file */
  _, err := srv.AssetPath("asdf", false)
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
  testEq(t, ret, "/foo-912ec803b2ce49e4a541068d495ab570.js")
}
