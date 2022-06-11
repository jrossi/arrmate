package server

import (
	"context"
	"flag"
	"fmt"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
)

var test_db_path = flag.String("arrmate.test_db_path", "", "Directory to store test databases, empty will default to temp dir will autoclean up")

type DBCS struct {
	cs string
}

func (self *DBCS) ConnectString() string {
	return self.cs
}

func makeDBConfig(t testing.TB, name string) DBConfig {
	var cs string

	cwd, _ := filepath.Abs("./..")

	if *test_db_path == "" {
		cs = filepath.Join(t.TempDir(), fmt.Sprintf("%s.sqlite", t.Name()))
	} else {
		cs = filepath.Join(cwd, *test_db_path, fmt.Sprintf("%s.sqlite", t.Name()))
	}
	fmt.Println("Path DB", cs)

	dcfg := &DBCS{
		cs: cs,
	}
	return dcfg
}

func TestDB_ConnPoolFunctions(t *testing.T) {
	onErrorFuncRan := 0
	db := &DB{}

	onReadyFuncRan := 0
	fakeOnReadyFunc := func() sqlitemigration.SignalFunc {
		f := db.OnReadyFunc()
		return func() {
			onReadyFuncRan += 1
			f()
		}
	}

	prepareConnRan := 0
	fakePrepareConn := func() sqlitemigration.ConnPrepareFunc {
		f := db.ConnPrepareFunc()
		return func(conn *sqlite.Conn) error {
			prepareConnRan += 1
			return f(conn)
		}
	}
	pool := sqlitemigration.NewPool("", Schema, sqlitemigration.Options{
		Flags:          sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenNoMutex | sqlite.OpenWAL,
		OnStartMigrate: db.StartMigrateFunc(),
		OnError:        db.OnErrorFunc(),
		OnReady:        fakeOnReadyFunc(),
		PrepareConn:    fakePrepareConn(),
	})
	db.Pool = pool
	db.Log = log.With().Str("ss", "db").Logger()

	defer db.Pool.Close()

	// Get a connection. This blocks until the migration completes.
	conn, err := db.Pool.Get(context.TODO())
	if err != nil {
		// handle error
	}
	defer db.Pool.Put(conn)

	assert.True(t, prepareConnRan >= 1, "PrepareConn ran before migrations")
	assert.True(t, onErrorFuncRan == 0, "OnErrorFunc should not have benn called")

}

func TestNewDB(t *testing.T) {
	dcfg := makeDBConfig(t, "testing")
	db, _ := NewDB(dcfg)
	defer db.Close()

	// Get a connection. This blocks until the migration completes.
	conn, err := db.Get(context.TODO())
	if err != nil {
		// handle error
	}
	defer db.Put(conn)
	t.Run("Test_PRAGMA_foreign_keys_enabled", func(t *testing.T) {
		s, _, err := conn.PrepareTransient("pragma foreign_keys;")
		if err != nil {
			return
		}
		defer s.Finalize()
		_, err = s.Step()
		assert.Nil(t, err, "Get pragma foreign_keys should always return someting")
		state := s.GetInt64("foreign_keys")
		assert.Equal(t, int64(1), state, "1 confirms foreign_keys are enabled for the connection")
	})
	t.Run("Test_PRAGMA_journal_mode_wal", func(t *testing.T) {
		s, _, err := conn.PrepareTransient("pragma journal_mode;")
		if err != nil {
			return
		}
		defer s.Finalize()
		_, err = s.Step()
		assert.Nil(t, err, "Get pragma journal_mode should always return someting")
		state := s.GetText("journal_mode")
		assert.Equal(t, "wal", state, "wal confirms journal_mode are enabled for the database")
	})

}

func TestNewDB_Migrations(t *testing.T) {
	dcfg := makeDBConfig(t, "testing")
	db, _ := NewDB(dcfg)
	defer db.Close()

	// Get a connection. This blocks until the migration completes.
	conn, err := db.Get(context.TODO())
	if err != nil {
		// handle error
	}
	defer db.Put(conn)

	t.Run("Migrations_pragm_user_version", func(t *testing.T) {
		s := conn.Prep("pragma user_version")
		defer s.Finalize()
		hasRow, err := s.Step()
		assert.Nil(t, err, "pragma should never return error")
		assert.Equal(t, true, hasRow, "pragma always returns a single results")
		userVersion := s.GetInt64("user_version")
		assert.Equal(t, int64(len(Migrations)), userVersion,
			"count of Migration/migration_*.sql  needs to be equal to user_version",
		)
	})
	// Check for all the expect tables that should be setup in the database.
	t.Run("Expected_Tables", func(t *testing.T) {
		// List of all the tables that are expect to be with in the database after migrations
		expectedTables := []string{"config", "sonarr"}

		for _, tName := range expectedTables {
			s := conn.Prep(" SELECT * FROM sqlite_master where type='table' and name=$name")
			s.SetText("$name", tName)
			if hasRow, err := s.Step(); err != nil {
				t.Errorf("Error in s.Step: %v", err)
			} else if !hasRow {
				t.Errorf("tName: %s was not found in database after migration", tName)
			}
			s.Finalize()
		}

		var tableFound []string

		s := conn.Prep("SELECT name from sqlite_master where type='table'")
		for {
			if hasRow, err := s.Step(); err != nil {
				t.Errorf("Error in s.Step: %v", err)
			} else if !hasRow {
				break
			}
			tableFound = append(tableFound, s.GetText("name"))
		}
		s.Finalize()

		var found int
		for _, tName := range tableFound {
			found = 0
			for _, expectedName := range expectedTables {
				if tName == expectedName {
					found = 1
				}
			}
			if found == 0 {
				t.Errorf("Extra table found in database name:%s", tName)
			}
		}

	})

}

func TestNewDB_Migrations_Indexes(t *testing.T) {

}

func TestDB_ConfigGet_ConfigSet(t *testing.T) {
	dcfg := makeDBConfig(t, "testing")
	db, _ := NewDB(dcfg)
	defer db.Close()

	t.Run("ConfigSet_basic_usage", func(t *testing.T) {
		var err error
		var found bool
		var result string
		err = db.ConfigSet("jeremy", "rossi")
		assert.NoError(t, err, "ConfigSet should have worked as expected")

		found, result, err = db.ConfigGet("jeremy")
		assert.Equal(t, result, "rossi", "ConfigSet then ConfigGet should result in rossi")
		assert.True(t, found)
		assert.NoError(t, err, "ConfigGet should not error when key is found")

		err = db.ConfigSet("jeremy", "rossi-rossi")
		assert.NoError(t, err, "ConfigSet should have worked as expected even with duplicate key")

		found, result, err = db.ConfigGet("jeremy")
		assert.Equal(t, result, "rossi-rossi", "ConfigSet then ConfigGet should result in rossi")
		assert.True(t, found)
		assert.NoError(t, err, "ConfigGet should not error when key is found")
	})
	t.Run("ConfigGet_no_key", func(t *testing.T) {
		var err error
		var found bool
		var result string
		found, result, err = db.ConfigGet("jeremy-no-key")
		assert.Equal(t, result, "", "ConfigSet then ConfigGet should result in rossi")
		assert.False(t, found)
		assert.NoError(t, err, "ConfigGet should not error when key is found")
	})
	t.Run("ConfigGet_Delete_key", func(t *testing.T) {
		var err error
		var found bool
		var result string
		err = db.ConfigSet("remove", "me")
		assert.NoError(t, err, "ConfigSet should have worked as expected")

		// Verify was inserted
		found, result, err = db.ConfigGet("remove")
		assert.Equal(t, result, "me", "ConfigSet then ConfigGet should result in rossi")
		assert.True(t, found)
		assert.NoError(t, err, "ConfigGet should not error when key is found")

		// Remove entry
		err = db.ConfigDelete("remove")
		assert.NoError(t, err, "ConfigDelete should not error when removing items")

		// Verify was removed
		found, result, err = db.ConfigGet("remove")
		assert.Equal(t, result, "", "ConfigSet then ConfigGet should result empty string")
		assert.False(t, found)
		assert.NoError(t, err, "ConfigGet should not error when key is not found")
	})
}

/*
func TestDB_NewDB_With_Config_set(t *testing.T) {
	var err error
	var found bool
	var result string

	s := make(map[string]string)
	s["jeremy"] = "rossi"

	dcfg := makeDBConfig(t, "testing")

	dcfg.Set = s
	dcfg.Get = []string{"jeremy"}

	db, _ := NewDB(dcfg)
	defer db.Close()
	err = db.RunCli(dcfg)
	assert.NoError(t, err, "RunCli should not return errors")

	found, result, err = db.ConfigGet("jeremy")
	assert.Equal(t, "rossi", result, "ConfigSet then ConfigGet should result in rossi")
	assert.True(t, found)
	assert.NoError(t, err, "ConfigGet should not error when key is found")
}
*/
