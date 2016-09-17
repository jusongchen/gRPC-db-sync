package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-oci8"
	"github.com/pkg/errors"

	_ "github.com/lib/pq"
)

type OraConnParams struct {
	OraConnID string
	OraUser   string
	OraPasswd string
}

type PgConnParams struct {
	PgHost   string
	PgUser   string
	PgPasswd string
	PgDB     string
}

func OpenDB() (*sql.DB, error) {
	var db *sql.DB
	var err error

	switch *dbVendor {
	case "Oracle":
		oraConn := OraConnParams{
			OraConnID: *dbServer,
			OraUser:   *dbUser,
			OraPasswd: *dbPasswd,
		}
		connStr := fmt.Sprintf("%s:%s@%s", oraConn.OraUser, oraConn.OraPasswd, oraConn.OraConnID)
		db, err = sql.Open("oci8", connStr)
		// defer db.Close()

		if err != nil {
			return nil, errors.Wrapf(err, "openDB with %s fail", connStr)
		}

		if ver, err := getOracleVersion(db); err != nil {
			return nil, errors.Wrapf(err, "openDB with %s fail", connStr)
		} else {
			log.Printf("Connected to Oracle server %s@%s\n", oraConn.OraUser, oraConn.OraConnID)
			log.Printf("Version: %s\n", ver)
		}
		// log.Printf("Connected to Oracle server %s@%s", oraConn.OraUser, oraConn.OraConnID)
		return db, nil
	case "PostGres":
		connStr := fmt.Sprintf("postgres://%s:%s@%s?sslmode=require", *dbUser, *dbPasswd, *dbServer)
		db, err = sql.Open("postgres", connStr)

		if err != nil {
			return nil, errors.Wrapf(err, "open PostGres DB as %s fail", connStr)
		}

		if ver, err := getPostgresVersion(db); err != nil {
			return nil, errors.Wrapf(err, "open PostGres DB as %s fail", connStr)
		} else {
			log.Printf("Connected to Postgres %s@%s", *dbUser, *dbServer)
			log.Printf("Postgres version %s\n", ver)
		}
		return db, nil
	default:
		log.Fatal("unknown DB verdor:%s, expecting Oracle or PostGres", *dbVendor)
	}
	return nil, nil
}

func getPostgresVersion(db *sql.DB) (string, error) {
	var ver string

	err := db.QueryRow("SELECT version() ver").Scan(&ver)
	switch {
	case err == sql.ErrNoRows:
		log.Fatal("Panic:'select version()' returns no row")

	case err != nil:
		return "", errors.Wrap(err, "getPostgresVersion fail")
	default:
	}
	return ver, nil
}

func getOracleVersion(db *sql.DB) (string, error) {
	// var inst_name, host_name, version string
	var version string
	// err := db.QueryRow("select instance_name,host_name,version from v$instance  ").Scan(&inst_name, &host_name, &version)
	err := db.QueryRow("select banner from v$version where rownum<=1").Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		log.Fatal("Panic:'select banner from v$version where rownum<=1' returns no row")
		return "", nil
	case err != nil:
		return "", errors.Wrap(err, "getOracleVersion:")
	default:
	}
	return version, nil
}
