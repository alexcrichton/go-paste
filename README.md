# go-paste

[![Build Status](https://travis-ci.org/alexcrichton/go-paste.png?branch=master)](https://travis-ci.org/alexcrichton/go-paste)

go-paste is a package for Go which is aimed at providing a
[sprockets](https://github.com/sstephenson/sprockets)-like experience for
managing assets for Go web applications. The api documentation can be found at
http://godoc.org/github.com/alexcrichton/go-paste

## Usage

Currently, go-paste doesn't provide a set of fancy wrappers which makes it so
this can just plug into your application. This does require some form of
configuration. To use it, you'll have to go through steps similar to this:

1. Create a `FileServer` instance and install it at some path:

```go
import "github.com/alexcrichton/go-paste

func main() {
  // ...

  // Assumes that there's a directory called 'assets' at the root of your
  // repository containing all the assets
  srv := paste.FileServer("./assets")
  http.Handle("/assets/", http.StripPrefix("/assets", srv))

  // ...
}
```

2. Modify HTML templates to use paste's paths instead of custom ones

```
# Some HTML template
<html>
  ...
  <script type='text/javascript' src='{{ AssetPath "foo.png" }}></script>
  <img src='{{ AssetPath "foo.png" }}
  ...
</html>

# Elsewhere in Go
var srv paste.Server

func AssetPath(path string) string {
  path, err := srv.AssetPath(path, false /* no digest in the filename */)
  // deal with err if non-nil
  return path
}
```

3. If processors are desired, be sure to import them somewhere in your project
   like:

```
import _ "github.com/alexcrichton/go-paste/jsmin"
import _ "github.com/alexcrichton/go-paste/sass"
```

## Asset Processors

For all assets, it's possible to specify dependencies of the asset to bundle
assets together. If `foo.js` required `bar.js`, then whenever `foo.js` is
generated the contents of `bar.js` will be inserted at the top of the generated
file.

To require other dependencies, the top of the file must contain a comment like:

```
//= require bar
//= require foo/bar
```

Currently only one directive, `require` is supported which means "insert the
source contents of this file here."

### JSMin

This is available via the `github.com/alexcrichton/go-paste/jsmin` package. When
imported, all javascript will be passed through `jsmin`, a minification of js
[implemented by Douglas Crockford](http://www.crockford.com/javascript/jsmin.html).

### Sass

This is available via the `github.com/alexcrichton/go-paste/sass` package. When
imported, all css will be passed through this processor for minification. Using
this package requires that you have
[libsass](https://github.com/hcatlin/libsass) installed on your system. As of
this writing, you must also have the HEAD version of `libsass` installed.

You can create files like `foo.scss` which are then served up via the logical
path of `foo.css`. The contents served up are also processed with sass.

## Deployment

When deploying an application, you probably don't want to slow down startup of
the app by spending a lot of time compiling assets and things like that. For
this reason paste has a method of pre-compiling assets into a directory which
can then be shipped with the deployment

The `Server` returned has a `Compile` method which takes a destination of where
things are supposed to go. For example, your `main` function may look like:

```go
srv := paste.FileServer(...)
if os.Args()[1] == "precompile" {
  srv.Compile(...)
}
```

In production, instead of using a `FileServer` you would want to use a
`CompiledFileServer`. This version has far fewer filesystem accesses and
contains all the precomputed digests to be placed in urls. For example, you
might have the following setup:

```go
# prod.go
// +build prod
package main

import "github.com/alexcrichton/go-paste"

var PasteServer paste.Server
func init() {
  srv, err := paste.CompiledFileServer("./precompiled")
  if err != nil { panic(err) }
  PasteServer = srv
}

# dev.go
// +build !prod
package main

import "github.com/alexcrichton/go-paste"

var PasteServer = paste.FileServer("./assets")
```

And then in development you'd just use `go build` whereas to build a production
version of the server you'd use `go build -tags prod`. Additionally the prod and
dev files could have a global constant `AssetDigest` which is passed to the
`AssetPath` function so that in production all urls would have the digest in
them (to have long expiry dates) whereas in development they would not (for
better debugging)
