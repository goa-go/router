# goa-router
Router middleware for goa.

[![Build Status](https://travis-ci.org/goa-go/router.svg?branch=master)](https://travis-ci.org/goa-go/router)
[![Codecov](https://codecov.io/gh/goa-go/router/branch/master/graph/badge.svg)](https://codecov.io/github/goa-go/router?branch=master)
[![Go Doc](https://godoc.org/github.com/goa-go/router?status.svg)](http://godoc.org/github.com/goa-go/router)
[![Go Report](https://goreportcard.com/badge/github.com/goa-go/router)](https://goreportcard.com/report/github.com/goa-go/router)

- Based on [httprouter](https://github.com/julienschmidt/httprouter)
- Multiple route middleware
- Named URL parameters
- Support for 405 Method Not Allowed
- Responds to OPTIONS requests with matching methods

## Installation
```bash
go get -u github.com/goa-go/goa 
```

## Example
```go
package main

import (
  ...

  "github.com/goa-go/goa"
  "github.com/goa-go/router"
)

func main() {
  app := goa.New()
  router := router.New()
  router.GET("/", func(c *goa.Context) {
    c.String("Hello Goa!")
  })
  ...

  app.Use(router.Routes())
  ...
}
```

## License

[MIT](https://github.com/goa-go/router/blob/master/LICENSE)
