package server

import (
	"context"
	"embed"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/fs"
	"sort"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

//go:embed migration/*.sql
var migration_files embed.FS

var Migrations []string
var Schema sqlitemigration.Schema

func init() {
	templates, _ := fs.Glob(migration_files, "migration/*.sql")
	for _, a := range templates {
		fmt.Println(a)
		s, err := migration_files.ReadFile(a)
		if err != nil {
			fmt.Println(err)
		}
		Migrations = append(Migrations, string(s))
	}
	sort.Strings(Migrations)
	Schema = sqlitemigration.Schema{
		AppID:               0xbf7294,
		Migrations:          Migrations,
		RepeatableMigration: "",
	}
}

/*
type DBConfig struct {
	ConnectString string `long:"connect" default:"" description:"SQLlite connect String"`
}
*/

type DBConfig interface {
	ConnectString() string
}

func NewDB(cfg DBConfig) (*DB, error) {

	db := &DB{}
	pool := sqlitemigration.NewPool(cfg.ConnectString(), Schema, sqlitemigration.Options{
		Flags:          sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenNoMutex | sqlite.OpenWAL,
		OnStartMigrate: db.StartMigrateFunc(),
		OnError:        db.OnErrorFunc(),
		OnReady:        db.OnReadyFunc(),
		PrepareConn:    db.ConnPrepareFunc(),
	})
	db.Pool = pool
	db.Log = log.With().Str("ss", "db").Logger()

	return db, nil
}

type DB struct {
	Pool *sqlitemigration.Pool
	Log  zerolog.Logger
}

func (d *DB) Get(ctx context.Context) (*sqlite.Conn, error) {
	return d.Pool.Get(ctx)
}

func (d *DB) Put(c *sqlite.Conn) {
	//c.Close()
	d.Pool.Put(c)
}

func (d *DB) Close() error {
	d.Log.Debug().Msg("Closing Database")
	return d.Pool.Close()
}

func (d *DB) StartMigrateFunc() sqlitemigration.SignalFunc {
	return func() {
		d.Log.Debug().Msg("Starting SQL Migrations")
	}
}

func (d *DB) OnReadyFunc() sqlitemigration.SignalFunc {
	return func() {
		d.Log.Debug().Msg("SQL Is Ready")
	}
}

func (d *DB) OnErrorFunc() sqlitemigration.ReportFunc {
	return func(err error) {
		d.Log.Warn().Err(err).Msg("Error in sqlite")
	}
}

func (d *DB) ConnPrepareFunc() sqlitemigration.ConnPrepareFunc {
	return func(conn *sqlite.Conn) error {
		d.Log.Debug().Msg("db.ConnPrepareFunc start")
		err := sqlitex.ExecuteTransient(conn, "PRAGMA foreign_keys = ON;", nil)
		if err != nil {
			return err
		}
		return nil
	}
}

func (d *DB) RawConfigGet(s *sqlite.Stmt, conn *sqlite.Conn, k string) (bool, string, error) {
	var err error
	// Create Statement
	s, err = conn.Prepare("SELECT value FROM config WHERE key = $key")
	if err != nil {
		return false, "", fmt.Errorf("execution of Select errored: %w", err)
	}
	s.SetText("$key", k)
	defer s.Reset()

	// Get results
	found, err := s.Step()
	if err != nil {
		return false, "", fmt.Errorf("execution s.step() of select errored: %w", err)
	}
	if found {
		val := s.GetText("value")
		return true, val, nil
	}
	return false, "", nil

}

func (d *DB) ConfigGet(k string) (bool, string, error) {
	var err error
	var s *sqlite.Stmt

	// Get Conn
	conn, err := d.Pool.Get(context.TODO())
	if err != nil {
		return false, "", err
	}
	defer d.Pool.Put(conn)
	return d.RawConfigGet(s, conn, k)
}

func (d *DB) RawConfigSet(s *sqlite.Stmt, conn *sqlite.Conn, k string, v string) error {
	var err error
	// Create Statement
	s, err = conn.Prepare("INSERT INTO config (key, value) VALUES ($key, $value) ON CONFLICT(key) DO UPDATE SET value = EXCLUDED.value")
	if err != nil {
		return fmt.Errorf("execution of Select errored: %w", err)
	}
	s.SetText("$key", k)
	s.SetText("$value", v)
	defer s.Reset()

	// Get results
	_, err = s.Step()
	if err != nil {
		return fmt.Errorf("execution s.step() of select errored: %w", err)
	}
	return nil

}

func (d *DB) ConfigSet(k, v string) error {
	var err error
	var s *sqlite.Stmt

	// Get Conn
	conn, err := d.Pool.Get(context.TODO())
	if err != nil {
		return err
	}
	defer d.Pool.Put(conn)
	return d.RawConfigSet(s, conn, k, v)
}

func (d *DB) RawConfigDelete(s *sqlite.Stmt, conn *sqlite.Conn, k string) error {
	var err error
	// Create Statement
	s, err = conn.Prepare("DELETE FROM config WHERE key = $key")
	if err != nil {
		return fmt.Errorf("execution of Select errored: %w", err)
	}
	s.SetText("$key", k)
	defer s.Reset()

	// Get results
	_, err = s.Step()
	if err != nil {
		return fmt.Errorf("execution s.step() of select errored: %w", err)
	}
	return nil

}
func (d *DB) ConfigDelete(k string) error {
	var err error
	var s *sqlite.Stmt

	// Get Conn
	conn, err := d.Pool.Get(context.TODO())
	if err != nil {
		return err
	}
	defer d.Pool.Put(conn)
	return d.RawConfigDelete(s, conn, k)

}

/*
func (d *DB) CreateHost(ctx context.Context, conn *sqlite.Conn, h *Host) (int64, error) {
	var err error
	var s *sqlite.Stmt
	if h.ID == 0 {
		s, err = conn.Prepare("INSERT into hosts(hostname, details) VALUES ($hostname, $details) RETURNING id;")
		if err != nil {
			//d.Log.Info().Err(err).Msg("Error Preparing statement without ID")
			return 0, fmt.Errorf("conn.Prepare errored: %w", err)
		}
	} else {
		s, err = conn.Prepare("INSERT INTO hosts(id, hostname, details) VALUES($id, $hostname, $details) RETURNING id;")
		if err != nil {
			//d.Log.Warn().Err(err).Msg("Error Preparing statement with ID")
			return 0, fmt.Errorf("conn.Prepare errored: %w", err)
		}
		s.SetInt64("$id", h.ID)
	}
	s.SetText("$hostname", h.Hostname)
	s.SetBytes("$details", h.Details)
	defer s.Reset()
	rowR, err := s.Step()
	if err != nil {
		return 0, fmt.Errorf("execution of INSERT errored: %w", err)
	}
	if rowR {
		id := s.GetInt64("id")
		return id, nil
	}
	return 0, nil
}


func (d *DB) GetHostByID(ctx context.Context, conn *sqlite.Conn, id int64) (bool, *Host, error) {
	var err error
	var s *sqlite.Stmt
	s, err = conn.Prepare("SELECT id, hostname, details FROM hosts WHERE id = $id")
	if err != nil {
		fmt.Printf("1.      \n")
		return false, nil, fmt.Errorf("execution of Select errored: %w", err)
	}
	fmt.Printf("2.         \n")
	s.SetInt64("$id", id)
	found, err := s.Step()
	if err != nil {
		return false, nil, fmt.Errorf("execution s.step() of select errored: %w", err)
	}
	if found {
		h := &Host{
			Hostname: s.GetText("hostname"),
			ID:       s.GetInt64("id"),
			Details:  make([]byte, s.GetLen("details")),
		}
		s.GetBytes("details", h.Details)
		return true, h, nil
	}
	return false, nil, nil
}

func (d *DB) GetHostsByHostname(ctx context.Context, conn *sqlite.Conn, hostname string) (bool, []*Host, error) {
	const query string = `SELECT id, hostname, details FROM hosts WHERE hostname = ?;`
	results := []*Host{}
	f := func(s *sqlite.Stmt) error {
		entry := &Host{
			ID: s.GetInt64("id"),
			Hostname: s.GetText("hostname"),
			Details: make([]byte, s.GetLen("details")),
		}
		s.GetBytes("details", entry.Details)
		results = append(results, entry)
		return nil
	}
	err := sqlitex.Exec(conn, query, f, hostname)
	if err != nil {
		return false, nil, err
	}
	if len(results) == 0 {
		return false, results, nil
	}
	return true, results, nil
}

func (d *DB) UpdateHostDetailsById(ctx context.Context, conn *sqlite.Conn, id, int64, details []byte) ([]byte, error) {




	return nil, nil
}

type Secret struct {
	ID        int64
	HostId    int64
	Secret    []byte
	StartTime int64
	EndTime   int64
}

// CreateSecret will add
func (d *DB) CreateSecret(ctx context.Context, conn *sqlite.Conn, host_id int64) (*Secret, error) {
	var s *sqlite.Stmt
	var err error
	result := &Secret{}
	s, err = conn.Prepare(
		`INSERT INTO secrets(host_id, secret, start_time, end_time)
                           VALUES ($host_id, $secret, $start_time, $end_time);`,
    )
	if err != nil {
		d.Log.Error().Err(err).Msg("Prepare statemented errored")
		return nil, err
	}
	// Secret Bytes
	secret, err := GenerateRandomBytes(64)
	if err != nil {
		d.Log.Error().Err(err).Msg("GenerateRandonBytes errored")
		return nil, err
	}
	result.Secret = secret
	s.SetBytes("$secret", secret)

	// Start and End times
	now := time.Now()
	result.StartTime = now.Unix()
	result.EndTime = now.AddDate(1, 1, 0).Unix()
	s.SetInt64("$start_time", result.StartTime)
	s.SetInt64("$end_time", result.EndTime)
	// Host_id
	s.SetInt64("$host_id", host_id)
	result.HostId = host_id

	defer s.ClearBindings()
	defer s.Reset()
	_, err = s.Step()
	if err != nil {
		d.Log.Error().Err(err).Int64("host_id", host_id).Msg("Execution s.Step() of insert errored")
		return nil, fmt.Errorf("execution s.step() of select errored: %w", err)
	} else {
		result.ID = conn.LastInsertRowID()
		//fmt.Printf("Found id:%d\n", result.ID)
		d.Log.Debug().
			Int64("last_id", conn.LastInsertRowID()).
			Int64("result.ID", result.ID).
			Int64("result.start", result.StartTime).
			Int64("result.end", result.EndTime).
			Msgf("found id:%d", result.ID)

		return result, nil
	}
	return nil, err
}

func (d *DB) DeleteSecret(ctx context.Context, conn *sqlite.Conn, secret_id int64) (error) {
	var s *sqlite.Stmt
	var err error
	s, err = conn.Prepare(
		`DELETE FROM secrets WHERE id = $secret_id;`,
	)
	if err != nil {
		d.Log.Error().Err(err).Msg("Prepare statemented errored")
		return err
	}
	s.SetInt64("$secret_id", secret_id)
	defer s.ClearBindings()
	defer s.Reset()
	_, err = s.Step()
	if err != nil {
		d.Log.Error().Err(err).Int64("secret_id", secret_id).Msg("Execution s.Step() of delete errored")
		return fmt.Errorf("execution s.step() of select errored: %w", err)
	} else {
		i := conn.Changes()
		if i == 1 {
			return nil
		} else {
			return fmt.Errorf("Number of changes to database was not equal to 1.  Total changes was %d", i)
		}
	}
	return nil
}

func (d *DB) GetSecretById(ctx context.Context, conn *sqlite.Conn, secret_id int64) (bool, *Secret, error) {
	var s *sqlite.Stmt
	var err error
	s, err = conn.Prepare(`SELECT host_id, secret, start_time, end_time
                                    FROM secrets WHERE id = $secret_id LIMIT 1;`,
    )
	if err != nil {
		d.Log.Error().Err(err).Msg("Prepare statemented errored")
		return false, nil, err
	}
	s.SetInt64("$secret_id", secret_id)
	defer s.ClearBindings()
	defer s.Reset()
	found, err := s.Step()
	if err != nil {
		return false, nil, err
	}

	if found {
		result := &Secret{
			ID: secret_id,
			StartTime: s.GetInt64("start_time"),
			EndTime: s.GetInt64("end_time"),
			HostId: s.GetInt64("host_id"),
			Secret: make([]byte, s.GetLen("secret")),
		}
		s.GetBytes("secret", result.Secret)
		return true, result, nil
	} else {
		return false, nil, nil
	}
}

func (d *DB) GetSecretsByHostname(ctx context.Context, conn *sqlite.Conn, hostname string) (bool, []*Secret, error) {
	const query string = `SELECT s.id as id,
                                 s.host_id as host_id,
                                 s.secret as secret,
                                 s.start_time as start_time,
                                 s.end_time as end_time
                            FROM secrets s
                            JOIN hosts h on s.host_id = h.id
                           WHERE h.hostname = $hostname;`
	results := []*Secret{}
	f := func(s *sqlite.Stmt) error {
		entry := &Secret{
			ID: s.GetInt64("id"),
			StartTime: s.GetInt64("start_time"),
			EndTime: s.GetInt64("end_time"),
			HostId: s.GetInt64("host_id"),
			Secret: make([]byte, s.GetLen("secret")),
		}
		s.GetBytes("secret", entry.Secret)
		results = append(results, entry)
		return nil
	}
	err := sqlitex.Exec(conn, query, f, hostname)
	if err != nil {
		return false, nil, err
	}
	return true, results, nil
}





*/
