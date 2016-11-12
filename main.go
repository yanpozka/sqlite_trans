package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	os.Remove("./foo.db")
}

const (
	createTableSQL = `create table foo (id integer not null primary key, name text); delete from foo;`
	insertFooSQL   = `insert into foo(id, name) values($1, $2)`
)

func main() {
	db, err := sqlx.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("%q: %s\n", err, createTableSQL)
	}

	if err := doubleInsert(db); err != nil {
		log.Println("error with double insert", err)
	}

	foos := []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}{}

	if err := db.Select(&foos, "SELECT id, name FROM foo"); err != nil {
		log.Fatal(err)
	}

	for _, f := range foos {
		log.Printf("%+v\n", f)
	}

	var ids = []int{2}
	query, args, err := sqlx.In("SELECT id, name FROM foo WHERE id IN (?);", ids)

	// sqlx.In returns queries with the `?` bindvar, we can rebind it for our backend
	query = db.Rebind(query)

	foos = foos[:0]

	if err := db.Select(&foos, query, args...); err != nil {
		log.Fatal("Error with IN query.", err)
	}

	log.Printf("%+v\n", foos)
}

func doubleInsert(db *sqlx.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	{
		stmt, err := tx.Prepare(insertFooSQL)
		if err != nil {
			return err
		}
		defer stmt.Close()

		if _, err := stmt.Exec(1, "Cha Cha Cha"); err != nil {
			tx.Rollback()
			return err
		}
	}

	{
		stmt, err := tx.Prepare(insertFooSQL)
		if err != nil {
			return err
		}
		defer stmt.Close()

		if _, err := stmt.Exec(2, "Mambo italiano"); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
