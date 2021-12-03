package schema

/*
func TestSQLite(t *testing.T) {
	db := connectDB(t, "sqlite")

	// run this first since other tests might change the expected schema
	t.Run("full migration", func(t *testing.T) {

		migrator := makeTestMigrator(WithDialect(SQLite))
		outOfOrderMigrations := []*Migration{
			{
				ID:     "D",
				Script: "CREATE TABLE t2 (id TEXT);",
			},
			{
				ID:     "C",
				Script: "DROP TABLE t2;",
			},
			{
				ID:     "B",
				Script: "CREATE TABLE t2 (id INTEGER);",
			},
			{
				ID:     "A",
				Script: "CREATE TABLE t1 (id INTEGER);",
			},
		}

		err := migrator.Apply(db, outOfOrderMigrations)
		if err != nil {
			t.Error(err)
		}

		rows, err := db.Query(
			`SELECT name, sql FROM sqlite_master WHERE type='table' ORDER BY name;`)
		if err != nil {
			t.Error(err)
		}

		results := make(map[string]string)
		for rows.Next() {
			var table, schema string
			if err := rows.Scan(&table, &schema); err != nil {
				t.Error(err)
			}
			results[table] = schema
		}

		const dontCare = "only care about table name"
		expected := map[string]string{
			"t1":              "CREATE TABLE t1 (id INTEGER)",
			"t2":              "CREATE TABLE t2 (id TEXT)",
			"test_migrations": dontCare,
			"test_locks":      dontCare,
		}

		for table, expSchema := range expected {
			schema, ok := results[table]
			if !ok {
				t.Errorf("expect to find table %q", table)
				continue
			}
			if expSchema != dontCare && expSchema != schema {
				t.Errorf("schema mismatch. expected %q, got %q", expSchema, schema)
			}
		}

		for table := range results {
			if _, ok := expected[table]; !ok {
				t.Errorf("unexpected extra table %q", table)
			}
		}
	})
}
*/
