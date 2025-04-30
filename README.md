# rhc insights

Data collection library for Go application [`rhc`](https://github.com/RedHatInsights/rhc).

## Developing

For now, this repository contains mock binary that acts as a playground for rhc subcommand.

```shell
$ make build
$ sudo ./rhc-insights --help
$ sudo _STAGE=1 HTTP_PROXY=... ./rhc-insights --debug run org.example.mock
```

## License

GNU GPL v3
