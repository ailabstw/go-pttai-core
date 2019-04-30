go-pttai-core
==========

Official golang implementation of the PTT.ai Core Framework.

[![API Reference](https://godoc.org/github.com/ailabstw/go-pttai-core?status.png)](https://godoc.org/github.com/ailabstw/go-pttai-core)
[![Travis](https://travis-ci.org/ailabstw/go-pttai-core.svg?branch=master)](https://travis-ci.org/ailabstw/go-pttai-core)

The architecture of PTT.ai can be found in the [link](https://docs.google.com/presentation/d/1q44LYz0i-iMxXMD9zfV9kqwah9UJGFOaQZxs0GvM5E4/edit#slide=id.p) [(中文版)](https://docs.google.com/presentation/d/1X6fGAElPtvsMK8Fys8VwSj9UPfNRkRRHDE0lQcUyK4Y/edit#slide=id.p)

More documents can be found in [PIPs](https://github.com/ailabstw/PIPs)

## E2E test

1. build test binary

```
$ cd e2e/bin
$ go build
```

2. run test

```
$ cd e2e
$ rm -rf tmp
$ go test -run TestFriendBasic
```
