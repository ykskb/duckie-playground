# Duckie Playground

Minimal (single-file) web service in Go for SQL (`SELECT`) playground with DuckDB 

### Background

DuckDB WASM is awesome. But sometimes it's not easy to make data files accessible to your local (like S3 in enterprise).

So I made a very simple web service in Go that hosts data and runs SQL through DuckDB.

### Minimal usage by trusted users only

* Not perfectly secure:

    The service validates the names of data sources and checks only `SELECT` statement can run, but it does not intend to be perfectly secure. 

* Scalability unknown:

    Each request creates DuckDB instance within goroutine, so scalability is unknown. I'd imagine heavy queries can easily use up RAM and CPUs depending on workloads.

* No Web API:

    For the reasons above, only UI-based usage is supported for human usage.

* No CSS:

    Only raw HTML tags are used. Sorry.

### How to use

1. Multiple data source can be configured. Basically it's a list of strings after `FROM` clause, so CSV file, Parquet file or functions like `read_parquet` can be used, including partitioned data. Just update `dataSources` string array in `duck-server.go` and make sure it's accessible from root.

2. Run `go run duckie.go` to start the service.

3. Visit the UI at `127.0.0.1:8081`

4. Selecting data source automatically runs `DESCRIBE` and shows data source schema.

5. Run any `SELECT` query you want to.
