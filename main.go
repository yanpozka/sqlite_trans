package main

import (
	"context"
	"log"
	"os"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

const (
	createTableSQL = `create table foo (id integer not null primary key, name text); delete from foo;`
	insertFooSQL   = `insert into foo(id, name) values($1, $2)`
)

func init() {
	os.Remove("./foo.db")
}

func main() {

	if err := enableOpenCensusTracingAndExporting(); err != nil {
		log.Fatalf("Failed to enable OpenCensus tracing and exporting: %v", err)
	}

	driverName, err := ocsql.Register("sqlite3", ocsql.WithAllTraceOptions())
	if err != nil {
		log.Fatalf("Failed to register the ocsql driver: %v", err)
	}

	log.Printf("Opening db with driver name: %q", driverName)
	db, err := sqlx.Open(driverName, "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		db.Close()
		// Wait to 4 seconds so that the traces can be exported
		waitTime := 4 * time.Second
		log.Printf("Waiting for %s seconds to ensure all traces are exported before exiting", waitTime)
		<-time.After(waitTime)

		os.Remove("./foo.db")
	}()

	ctx, span := trace.StartSpan(context.Background(), "SQLite Go Tx examples")
	defer span.End()

	if err = spanErr(ctx, span, "Create foo table", func(ctx context.Context) error {
		_, errEx := db.ExecContext(ctx, createTableSQL)
		return errEx
	}); err != nil {
		log.Fatalf("%q: %s", err, createTableSQL)
	}

	if err := doubleInsert(ctx, span, db); err != nil {
		log.Println("error with double insert", err)
	}

	foos := []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}{}

	if err = spanErr(ctx, span, "Select all foos", func(ctx context.Context) error {
		return db.SelectContext(ctx, &foos, "SELECT id, name FROM foo")
	}); err != nil {
		log.Fatal(err)
	}

	for _, f := range foos {
		log.Printf("%+v", f)
	}

	foos = foos[:0]

	if err = spanErr(ctx, span, "Select foo with IN()", func(ctx context.Context) error {
		var ids = []int{2}
		query, args, errS := sqlx.In("SELECT id, name FROM foo WHERE id IN (?);", ids)
		if errS != nil {
			return errS
		}

		// sqlx.In returns queries with the `?` bindvar, we can rebind it for our backend
		query = db.Rebind(query)

		return db.SelectContext(ctx, &foos, query, args...)
	}); err != nil {
		log.Fatal("Error with IN query.", err)
	}

	log.Printf("%+v", foos)
}

func spanErr(ctx context.Context, span *trace.Span, name string, fn func(ctx context.Context) error) error {
	if fn == nil {
		panic("no function")
	}

	cCtx, cSpan := trace.StartSpan(ctx, name)
	err := fn(cCtx)
	cSpan.End()
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: err.Error()})
		return err
	}
	return nil
}

func doubleInsert(ctx context.Context, span *trace.Span, db *sqlx.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err = spanErr(ctx, span, "Insert Cha cha cha & Mambo italiano", func(ctx context.Context) error {
		stmt, err := tx.PrepareContext(ctx, insertFooSQL)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		if _, err := stmt.ExecContext(ctx, 1, "Cha Cha Cha"); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := stmt.ExecContext(ctx, 2, "Mambo italiano"); err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func enableOpenCensusTracingAndExporting() error {
	// For demo purposes, we'll always trace
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: "localhost:6831",
		Endpoint:      "http://localhost:14268",
		ServiceName:   "sqlite-ocsql-demo",
	})
	if err == nil {
		// On success, register it as a trace exporter
		trace.RegisterExporter(je)
	}

	return err
}
