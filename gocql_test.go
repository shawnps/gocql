// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.

package gocql

import (
	"bytes"
	"database/sql"
	"github.com/tux21b/gocql/uuid"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	db, err := sql.Open("gocql", "localhost:8000 keyspace=system")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("SELECT keyspace_name FROM schema_keyspaces")
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var keyspace string
		if err := rows.Scan(&keyspace); err != nil {
			t.Fatal(err)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

type Page struct {
	Title      string
	RevID      uuid.UUID
	Body       string
	Hits       int
	Protected  bool
	Modified   time.Time
	Attachment []byte
}

var pages = []*Page{
	&Page{"Frontpage", uuid.TimeUUID(), "Hello world!", 0, false,
		time.Date(2012, 8, 20, 10, 0, 0, 0, time.UTC), nil},
	&Page{"Frontpage", uuid.TimeUUID(), "Hello modified world!", 0, false,
		time.Date(2012, 8, 22, 10, 0, 0, 0, time.UTC), []byte("img data\x00")},
	&Page{"LoremIpsum", uuid.TimeUUID(), "Lorem ipsum dolor sit amet", 12,
		true, time.Date(2012, 8, 22, 10, 0, 8, 0, time.UTC), nil},
}

func TestWiki(t *testing.T) {
	db, err := sql.Open("gocql", "localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec("DROP KEYSPACE gocql_wiki")
	if _, err := db.Exec(`CREATE KEYSPACE gocql_wiki
        WITH strategy_class = 'SimpleStrategy'
        AND strategy_options:replication_factor = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("USE gocql_wiki"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`CREATE TABLE page (
        title varchar,
        revid timeuuid,
        body varchar,
        hits int,
        protected boolean,
        modified timestamp,
        attachment blob,
        PRIMARY KEY (title, revid)
        )`); err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if _, err := db.Exec(`INSERT INTO page (title, revid, body, hits,
            protected, modified, attachment) VALUES (?, ?, ?, ?, ?, ?, ?);`,
			p.Title, p.RevID, p.Body, p.Hits, p.Protected, p.Modified,
			p.Attachment); err != nil {
			t.Fatal(err)
		}
	}

	rowCount := 0
	rows, err := db.Query(`SELECT revid FROM page`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var revid uuid.UUID
		if err := rows.Scan(&revid); err != nil {
			t.Fatal(err)
		}
		rowCount++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if rowCount != len(pages) {
		t.Fatalf("expected %d rows, got %d", len(pages), rowCount)
	}

	for _, page := range pages {
		row := db.QueryRow(`SELECT title, revid, body, hits, protected,
            modified, attachment
            FROM page WHERE title = ? AND revid = ?`, page.Title, page.RevID)
		var p Page
		err := row.Scan(&p.Title, &p.RevID, &p.Body, &p.Hits, &p.Protected,
			&p.Modified, &p.Attachment)
		if err != nil {
			t.Fatal(err)
		}
		p.Modified = p.Modified.In(time.UTC)
		if page.Title != p.Title || page.RevID != p.RevID ||
			page.Body != p.Body || page.Modified != p.Modified ||
			page.Hits != p.Hits || page.Protected != p.Protected ||
			!bytes.Equal(page.Attachment, p.Attachment) {
			t.Errorf("expected %#v got %#v", *page, p)
		}
	}

	row := db.QueryRow(`SELECT title, revid, body, hits, protected,
        modified, attachment
        FROM page WHERE title = ? ORDER BY revid DESC`, "Frontpage")
	var p Page
	if err := row.Scan(&p.Title, &p.RevID, &p.Body, &p.Hits, &p.Protected,
		&p.Modified, &p.Attachment); err != nil {
		t.Error(err)
	}
	p.Modified = p.Modified.In(time.UTC)
	page := pages[1]
	if page.Title != p.Title || page.RevID != p.RevID ||
		page.Body != p.Body || page.Modified != p.Modified ||
		page.Hits != p.Hits || page.Protected != p.Protected ||
		!bytes.Equal(page.Attachment, p.Attachment) {
		t.Errorf("expected %#v got %#v", *page, p)
	}
}