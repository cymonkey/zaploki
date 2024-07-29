<h1 align="center">
  zaploki
</h1>

<p align="center">
  <a href="https://cymonkey.mit-license.org/"><img src="https://img.shields.io/badge/License-MIT-blue.svg"></a>
</p>

This is a generic zap core implementation for loki, independent with loki client, there're some implementation of loki client out there that you can choose to work with.

## Installation
```
$ go get github.com/cymonkey/zaploki
```

## Usage
```go
import (
    "github.com/cymonkey/zaploki"
)

func main() {
    zaploki.NewCore(NewWithDefaultConfig(&loki.Config{}))
}
```

## License

MIT License, check [LICENSE](./LICENSE).