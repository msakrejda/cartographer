## Cartographer

Cartographer is to psql as augmented reality is to boring, regular old
reality. It provides some visual context, tools, and automatic graphing
for your `psql` session without getting in the way.


### Usage

You must have Go installed and GOPATH configured correctly.

```console
$ go get github.com/uhoh-itsmaciek/cartographer
$ cd $GOPATH/src/github.com/uhoh-itsmaciek/cartographer
$ go build ./...
$ ./cartographer localhost:65432 /var/run/postgresql/.s.PGSQL.5434
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