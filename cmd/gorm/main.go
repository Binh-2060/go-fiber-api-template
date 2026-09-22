package main

import (
	"log"
	"os"

	"github.com/Binh-2060/go-application-template/internal/config/dotenv"
	"github.com/Binh-2060/go-application-template/pkg/db"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func init() {
	if os.Getenv("GO_ENV") == "" {
		dotenv.SetDotenv()
	}
}

/*
Generate one reference struct per table in the live database into
internal/api/models/gen/<table>.gen.go. Run from the repo root — ModelPkgPath is
resolved against the working directory:

	go run ./cmd/gorm

Reference only: the structs show each column's Go type, SQL type and
nullability, to check scans and validators against. Nothing imports them —
repositories declare their own models in internal/api/models.

Models only: nothing calls ApplyBasic, so Execute writes no query code and
OutPath is never created. It is still set because gen derives the query package
name from it. Connects through db.ConfigFromEnv, so it reads the same
DATABASE_URL / DB_* vars as the API.
*/
func main() {
	gormdb, err := gorm.Open(postgres.Open(db.ConfigFromEnv().DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:      "internal/api/models/gen/query",
		ModelPkgPath: "internal/api/models/gen",
		// Nullable column -> pointer: pgx scans NULL as nil and JSON encodes it
		// as null, so "not set" never looks like "" or 0.
		FieldNullable: true,
		// Keep the SQL type (uuid, varchar(200), ...) in the gorm tag.
		FieldWithTypeTag: true,
	})
	g.UseDB(gormdb)
	g.GenerateAllTable()
	g.Execute()
}
