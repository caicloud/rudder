<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Rudder](#rudder)
  - [About the project](#about-the-project)
    - [Status](#status)
    - [Design](#design)
  - [Getting Started](#getting-started)
    - [Layout](#layout)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Rudder

## About the project

Application packaging system based on helm Charts.

### Status

The project is still in `alpha` version.

### Design

Learn more about rudder on [design doc](docs/design.md)

## Getting Started
### Layout

We ignored `.git`, `bin`, `vendor` with command:
```
$ tree -d -I 'vendor|bin|.git'
```

```
├── build
│   └── controller
├── cmd
│   └── controller
│       └── app
├── docs
└── pkg
    ├── controller
    │   ├── gc
    │   ├── release
    │   └── status
    ├── kube
    ├── release
    ├── render
    ├── status
    │   └── assistants
    ├── storage
    └── store
```

Explanation for main pkgs:

- `build` contains a docker file for rudder controller.
- `cmd` contains main packages, each subdirecoty of cmd is a main package.
- `docs` for project documentations.
- `pkg`
  - `controller` contains gc/release/status controllerso
  - `kube` contains tools to communicate with kubernetes cluster. You can find:
    - A rest client pool.
    - A codec for converting between resource and object.
    - A resource client.
  - `release` has a manager to manages all release coroutines.
  - `render` can render a template with config.
  - `status` has many assistants to judge the status for specific resources.
  - `storage` contains a tool to manipulate release.
  - `store` contains a integration informer factory.

