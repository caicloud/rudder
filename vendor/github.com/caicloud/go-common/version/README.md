# version

## 使用

### 1. 在 Makefile 中设置真实值

在 `go build` 中的 `-ldflags` 参数后面逐个为变量赋真实值。

```makefile
GOCOMMON     := github.com/caicloud/go-common
VERSION      ?= $(shell git describe --tags --always --dirty)
GITREMOTE    ?= $(shell git remote get-url origin)
GITCOMMIT    ?= $(shell git rev-parse HEAD)
GITTREESTATE ?= $(if $(shell git status --porcelain),dirty,clean)
BUILDDATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build:
	go build -i -v -o ./bin/example                            \
	-ldflags "-s -w -X $(GOCOMMON)/version.version=$(VERSION)  \
	  -X $(GOCOMMON)/version.gitRemote=$(GITREMOTE)            \
	  -X $(GOCOMMON)/version.gitCommit=$(GITCOMMIT)            \
	  -X $(GOCOMMON)/version.gitTreeState=$(GITTREESTATE)      \
	  -X $(GOCOMMON)/version.buildDate=$(BUILDDATE)"           \
	.
```

### 2. 在代码中使用 version

在代码中加入如下行，并确保它会被执行到：

```go
fmt.Printf("go-common build information: %v\n", version.Get().Pretty())
```

### 3. 查看输出

然后通过 Makefile 构建可执行文件并执行，会在终端打印出如下信息：

```text
go-common build information: {
    "version": "v0.1.0-33-g164779c-dirty",
    "gitRemote": "https://github.com/hezhizhen/go-common.git",
    "gitCommit": "164779c155b909d3d18c9b81eeca99df8fcdffc8",
    "gitTreeState": "dirty",
    "buildDate": "2019-09-18T08:05:09Z",
    "goVersion": "go1.13",
    "compiler": "gc",
    "platform": "darwin/amd64"
}
```
