package schema

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSQLite(t *testing.T) {
	db := connectDB(t, "sqlite")

	// run this first since other tests might change the expected schema
	t.Run("full migration", func(t *testing.T) {
		tableName := "test_migrations"
		dialect := NewSQLite(WithSQLiteLockTable("test_locks"))

		migrator := NewMigrator(WithDialect(dialect), WithTableName(tableName))
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

	t.Run("locking", func(t *testing.T) {
		var wg sync.WaitGroup
		var inflight int32
		tableName := "test_migrations"

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				s := NewSQLite()
				if err := s.Lock(db, tableName); err != nil {
					t.Error(err)
				}
				atomic.AddInt32(&inflight, 1)
				if !atomic.CompareAndSwapInt32(&inflight, 1, 1) {
					t.Error("expected 1 concurrent sqlite migration")
				}

				time.Sleep(500 * time.Millisecond)

				atomic.AddInt32(&inflight, -1)
				if err := s.Unlock(db, tableName); err != nil {
					t.Error(err)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	})

	t.Run("lock timeout", func(t *testing.T) {
		s := NewSQLite(WithSQLiteLockDuration(3 * time.Second))
		tableName := "test_migrations"

		_, err := db.Exec(
			fmt.Sprintf(`INSERT INTO %s (id, code, expiration) VALUES (?,?,?)`, s.lockTable),
			lockMagicNum, 1234, time.Now().Add(10*time.Second))
		if err != nil {
			t.Error(err)
		}

		err = s.Lock(db, tableName)
		if err != ErrSQLiteLockTimeout {
			t.Errorf("expected timeout error, got %v", err)
		}
	})
}
