fuka_exporter
=============

This is a library that wraps useful FUKA C++ functions and builds as a shared objects,
so it they can be called from other languages, such as C, Python, Go.


Installation
----
Install Task first (https://taskfile.dev):
```shell
mamba install go-task
```

Build FUKA, so that it produces a static C++ library called `libkadath.a`
in `$HOME_KADATH/lib`.

Export variable that points to the FUKA build:
```shell
export HOME_KADATH=$HOME/fuka
```

Run the build of `fuka-exporter`:

```shell
task build
```

This will produce `libfuka_exporter.so`, which you can either place into your libraries location
or append `LD_LIBRARY_PATH` with its current location:

```shell
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$PWD
```

After that, your code that is linked against `fuka_exporter` can run.