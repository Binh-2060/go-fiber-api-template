/*
Package testsupport holds the setup integration tests share: a TestMain body
that opens the database, and Marker for row isolation. It knows nothing about
any one feature, so every feature's route tests (internal/api/tests, and the
examples' tests/ packages) reuse it as is.
*/
package testsupport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Binh-2060/go-application-template/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

/*
Main is the body of TestMain for an integration test package:

	func TestMain(m *testing.M) { testsupport.Main(m) }

It loads the environment, opens the pool, runs the tests and closes the pool.
*/
func Main(m *testing.M) {
	loadEnv()

	if err := db.Init(context.Background(), db.ConfigFromEnv()); err != nil {
		// Not a skip: -tags=integration is an explicit request for these tests
		// to run, so a missing database is a failure rather than something to
		// pass over quietly.
		panic("integration tests need a database: " + err.Error())
	}

	code := m.Run()

	// Not deferred: os.Exit does not run deferred functions.
	db.Close()
	os.Exit(code)
}

/*
Load .env the way cmd/api does, but from the repository root.

config/dotenv calls godotenv.Load(), which resolves ".env" against the working
directory. That is the repository root when the binary runs, but `go test` runs
each package in its own directory, so the file has to be found by walking up.
*/
func loadEnv() {
	if os.Getenv("GO_ENV") != "" {
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding one. Not fatal: the
			// variables may already be exported, and db.Init reports the real
			// problem if they are not.
			return
		}
		dir = parent
	}
}

/*
Marker returns a unique prefix for one test and registers the cleanup that
deletes every row of table whose column starts with it. The test puts the
prefix at the start of that column in every row it creates:

	m := testsupport.Marker(t, "users", "name")    // name = m + "Ada"
	m := testsupport.Marker(t, "products", "sku")  // sku  = m + "A-1"

Pick a text column the test always fills. Isolation is by marker rather than by
truncating the table, so the tests never touch rows they did not create and can
run against a shared database.

The prefix contains no '_' or '%': both are wildcards in LIKE, and a marker that
matched other rows would delete data the test did not create.
*/
func Marker(t *testing.T, table, column string) string {
	t.Helper()

	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	marker := "tst" + hex.EncodeToString(buf)

	t.Cleanup(func() {
		// Background context: the test's own context may already be done, and
		// cleanup still has to run.
		ctx := context.Background()
		// table and column are identifiers, which cannot be bound like values;
		// Sanitize quotes them so they can only ever name a table and a column.
		q := `DELETE FROM ` + pgx.Identifier{table}.Sanitize() +
			` WHERE ` + pgx.Identifier{column}.Sanitize() + ` LIKE $1`
		if _, err := db.Q(ctx).Exec(ctx, q, marker+"%"); err != nil {
			t.Errorf("cleanup %s: %v", marker, err)
		}
	})

	return marker
}
