// Paste is a package for dealing with static assets in web applications.
//
// This package is based off the 'sprockets' gem for Rails. The goal is to
// provide similar functionality while keeping to go-like principles instead of
// rails-like principles. This package deals with serving assets while applying
// filters.
package paste

import "net/http"
import "os"
import "path"
import "path/filepath"
import "regexp"
import "sync"
import "time"

// Version string which is prepended to all hashes generated. This doesn't
// necessarily reflect the version number of the package, but rather when it
// changes it forces all regenerated assets' hashes to change.
const Version = "0.0.0"

// Regex for finding the md5 hash in a requested filename (if any)
var hashRegex = regexp.MustCompile(`(.*)-([a-f0-9]{32})(\.\w+)`)

type assetMeta struct {
  err     error
  Asset
  sync.Mutex
}

// A paste Server instance is used to interact with the assets on the
// filesystem. It impelments http.Handler to be mounted at any path, and it will
// serve up the assets requested on that path.
type Server interface {
  // Implemented to be mountable at a path via http.Handler
  http.Handler

  // Given the 'logical' name of an asset, returns the pathname of the asset.
  // This should be used for url-generation of assets.
  //
  // An error is returned if the asset could not be found or could not be
  // processed for one reason or another
  AssetPath(logical string) (string, error)

  // Compiles all assets into the 'dst' directory. This is intended to be
  // invoked before deploying an application. The compiled assets are generated
  // in four different forms:
  //
  //    foo.js           - processed asset, initial filename
  //    foo.js.gz        - same as above but gzipped
  //    foo-md5hash.js   - same as 'foo.js' but with the hash in the filename
  //    foo-m5hash.js.gz - same as above but gzipped
  //
  // The gzipped versions of files are generated for web servers which can serve
  // up a gzipped file by default instead of having to re-gzip all assets all
  // the time. All generated gzip files have the maximum compression enabled.
  Compile(dst string) error

  // Fetches an Asset instance for a given logical path, returning any errors
  // encountered along the way
  Asset(logical string) (Asset, error)

  // Returns the configuration of this server so it can be modified for all
  // future work the server does
  Config() *Config
}

// Version of a server which watches for file names and regenerates files as
// necessary.
type fileServer struct {
  assets map[string]*assetMeta
  config Config
  sync.Mutex
}

// Configuration for when creating a FileServer
type Config struct {
  // String to prepend to the contents of everything being digested. When this
  // changes all digests change, so this acts as a 'cache buster'
  Version string

  // Location in the filesystem which all assets are to be derived from
  Root string

  // Flag if output should be compressed or not
  Compressed bool

  // Location to put intermediate files when compiling
  TempDir string
}

// A processor is a method of putting an asset through a 'pipeline' of
// modifications for things like compression, preprocessing, etc.
type Processor interface {
  // Given the input file of the asset, run the processor and write the output
  // to the given output file, returning any error encountered along the way
  Process(infile, outfile string) error
}

// Easy way of implementing a processor as just a function
type ProcessorFunc func(infile, outfile string) error

// Global registries modified by 'Register*'
var processors = make(map[string]Processor)
var compressors = make(map[string]Processor)
var aliases = make(map[string][]string)

// Registers a processor to run for the given extension whenever files are
// processed. It is considered an error to register more than one processor for
// a given file extension and this function will panic as a result.
//
// Example:
//
//    import "github.com/alexcrichton/go-paste"
//
//    func init() {
//      paste.RegisterProcessor(paste.ProcessorFunc(process), ".sass")
//    }
//
//    func process(infile string, outfile string) error {
//      // ... convert the sass in 'infile' to css in 'outfile'
//    }
func RegisterProcessor(p Processor, ext string) {
  _, ok := processors[ext]
  if ok {
    panic("Processor already registered for " + ext)
  }
  processors[ext] = p
}

// Registers a compressor to run for the given extension whenever files are
// compressed. It is considered an error to register more than one compressor
// for a given file extension and this function will panic as a result.
//
// Compressors are not always run, they can be configured to not run if desired
//
// Example:
//
//    import "github.com/alexcrichton/go-paste"
//
//    func init() {
//      paste.RegisterCompressor(paste.ProcessorFunc(minify), ".js")
//    }
//
//    func minify(infile string, outfile string) error {
//      // ... do something like invoke the closure compiler
//    }
func RegisterCompressor(p Processor, ext string) {
  _, ok := compressors[ext]
  if ok {
    panic("Compressor already registered for " + ext)
  }
  compressors[ext] = p
}

// Registers an alias from one extension to another. This means that any files
// which end with the extension 'alias' will also be understood to translate to
// the 'extension'
//
// Example:
//
//    import "github.com/alexcrichton/go-paste"
//
//    func init() {
//      // Enable all '.scss' files to be looked for as well whenever a '.css'
//      // file is requested
//      paste.RegisterAlias(".css", ".scss")
//    }
func RegisterAlias(extension, alias string) {
  prev, ok := aliases[extension]
  if !ok {
    prev = make([]string, 0)
  }
  prev = append(prev, alias)
  aliases[extension] = prev
}

// Creates a new file server for assets. This server is meant for development
// and updates all assets on-the-fly as they're requested. It watches for local
// changes and will process assets as they're created and modified.
//
// The path given is the root path to deliver all assets out of. They're all
// interpreted as being relative to this location.
//
// The version argument is some string to prepend to all hashes such that when
// it changes the digests of all assets will change. This is meant for an easy
// form of 'cache busting'
func FileServer(c Config) Server {
  abs, err := filepath.Abs(c.Root)
  if err != nil { panic(err) }
  c.Root = abs

  if c.TempDir == "" {
    c.TempDir = filepath.Join(c.Root, "tmp")
  } else {
    abs, err = filepath.Abs(c.TempDir)
    if err != nil { panic(err) }
    c.TempDir = abs
  }

  return &fileServer{ assets: make(map[string]*assetMeta), config: c }
}

func (p ProcessorFunc) Process(infile, outfile string) error {
  return p(infile, outfile)
}

func (s *fileServer) Config() *Config {
  return &s.config
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  serveHTTP(s, w, r)
}

func serveHTTP(s Server, w http.ResponseWriter, r *http.Request) {
  dir, file := path.Split(r.URL.Path)
  file, digest := findDigest(file)
  asset, err := s.Asset(path.Join(dir, file))
  if err != nil || (digest != "" && digest != asset.Digest()) {
    http.NotFound(w, r)
    return
  }


  headers := w.Header()
  if digest != "" {
    endoftime := time.Now().Add(31536000 * time.Second)
    headers.Set("Cache-Control", "public, max-age=31536000")
    headers.Set("Expires", endoftime.Format(http.TimeFormat))
  } else {
    headers.Set("Cache-Control", "public, must-revalidate")
  }
  if etagMatches(w, r, asset) {
    return
  }
  http.ServeFile(w, r, asset.Pathname())
}

func etagMatches(w http.ResponseWriter, r *http.Request, a Asset) bool {
  tag := r.Header.Get("If-None-Match")
  if tag == etag(a) {
    w.WriteHeader(http.StatusNotModified)
    return true
  }
  w.Header().Set("ETag", etag(a))
  return false
}

func etag(a Asset) string {
  return `"` + a.Digest() + `"`
}

func findDigest(file string) (string, string) {
  matches := hashRegex.FindStringSubmatch(file)
  if len(matches) == 0 {
    return file, ""
  }
  return matches[1] + matches[3], matches[2]
}

func (s *fileServer) Asset(logical string) (Asset, error) {
  logical = path.Clean("/" + logical)
  s.Lock()
  ret, ok := s.assets[logical]
  if !ok {
    ret = &assetMeta{}
    s.assets[logical] = ret
  }
  ret.Lock()
  defer ret.Unlock()
  s.Unlock()
  if ret.err != nil {
    return nil, ret.err
  } else if ret.Asset == nil || ret.Stale() {
    a, err := s.buildAsset(logical)
    if err == nil {
      ret.Asset = a
    } else {
      s.Lock()
      delete(s.assets, logical)
      s.Unlock()
      ret.err = err
      return nil, err
    }
  }
  return ret, nil
}

func (s *fileServer) buildAsset(logical string) (Asset, error) {
  pathname, err := s.resolve(logical)
  if err != nil {
    return nil, err
  }

  /* If we have a processor, or possibly a compressor, or this is js/css which
     could possibly have requires at the top, then we need a processed asset */
  _, ok1 := processors[path.Ext(pathname)]
  _, ok2 := compressors[path.Ext(logical)]
  if ok1 || (ok2 && s.config.Compressed) ||
      path.Ext(logical) == ".js" || path.Ext(logical) == ".css" {
    return newProcessed(s, logical, pathname)
  }
  return newStatic(s, logical, pathname)
}

func (s *fileServer) resolve(logical string) (string, error) {
  try := filepath.Join(s.config.Root, logical)
  _, err := os.Stat(try)
  if err == nil {
    return try, nil
  }
  ext := filepath.Ext(logical)
  candidates, ok := aliases[ext]
  if ok {
    for _, cand := range candidates {
      try = filepath.Join(s.config.Root, logical[:len(logical) - len(ext)] + cand)
      _, err = os.Stat(try)
      if err == nil {
        return try, nil
      }
    }
  }
  return "", err
}

func (s *fileServer) AssetPath(logical string) (string, error) {
  asset, err := s.Asset(logical)
  if err != nil {
    return "", err
  }
  return asset.LogicalName(), nil
}
