# Eirinix
[![godoc](https://godoc.org/github.com/SUSE/eirinix?status.svg)](https://godoc.org/github.com/SUSE/eirinix)
[![Build Status](https://travis-ci.org/SUSE/eirinix.svg?branch=master)](https://travis-ci.org/SUSE/eirinix)
[![go report card](https://goreportcard.com/badge/github.com/SUSE/eirinix)](https://goreportcard.com/report/github.com/SUSE/eirinix)
[![codecov](https://codecov.io/gh/SUSE/eirinix/branch/master/graph/badge.svg)](https://codecov.io/gh/SUSE/eirinix)

Extensions Library for Cloud Foundry Eirini

## How to use


### Install
    go get -u github.com/SUSE/eirinix

### Write your extension

An Eirini extension is a structure which satisfies the ```eirinix.Extension``` interface.

The interface is defined as follows:

```golang
type Extension interface {
	Handle(context.Context, Manager, *corev1.Pod, types.Request) types.Response
}
```

For example, a dummy extension (which does nothing) would be:

```golang

type MyExtension struct {}

func (e *MyExtension) Handle(context.Context, eirinix.Manager, *corev1.Pod, types.Request) types.Response {
	return types.Response{}
}
```


### Start the extension with eirinix

```golang

import "github.com/SUSE/eirinix"

func main() {
    x := eirinix.NewManager(
            eirinix.ManagerOptions{
                Namespace:  "kubernetes-namespace",
                Host:       "listening.eirini-x.org",
                Port:       8889,
                // KubeConfig can be ommitted for in-cluster connections
                KubeConfig: kubeConfig, 
        })

    x.AddExtension(&MyExtension{})
    log.Fatal(x.Start())
}

```
