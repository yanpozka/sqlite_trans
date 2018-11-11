### Example of database manipulation in Go with [sqlite](https://github.com/mattn/go-sqlite3) and [sqlx](https://github.com/jmoiron/sqlx) packages

### Tracing with [OpenCensus](https://opencensus.io/) and [Jaeger](https://www.jaegertracing.io/)

#### Run:

```
./run_jaeger.sh # starts a new container with Jaeger

go run main.go
```

##### Then visit http://localhost:16686/ and you'll see something like this:

![Jaeger dashboard](https://raw.githubusercontent.com/yanpozka/sqlite_trans/master/traces.png)
