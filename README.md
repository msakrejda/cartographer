## Cartographer

Cartographer is to psql as augmented reality is to boring, regular old
reality. It provides some visual context, tools, and automatic graphing
for your `psql` session without getting in the way.


### Usage

At this time, you'll just see the query traffic logged. Cartographer
is in progress.

Right now, you must have a locally installed version of
[FEMEBE](https://github.com/deafbybeheading/cartographer.git) to run
Cartographer (it will be `go get`-able eventually).

 In $GOROOT/src

```console
$ git clone git@github.com:deafbybeheading/cartographer.git
$ cd cartographer
$ go run cartographer.go localhost:65432 /var/run/postgresql/.s.PGSQL.5434
```

This will block. The first argument is the address Cartographer will
listen on (and what your psql client will connect to), the second is
the address Postgres is listening on (either the full path or a Unix
socket--the latter is typically in `/var/run/postgresql` or in
`/tmp`).

Then in another session, run the following:

```console
PGSSLMODE=disable psql -p 65432 -h localhost
```

And then run queries as usual. You'll see the queries logged in a
simple JSON format in the cartographer console.