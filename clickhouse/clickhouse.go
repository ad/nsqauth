package clickhouse

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go"
)

type DB struct {
	DB    *sql.DB
	Table string
}

// Secret ...
type Secret struct {
	UUID        string
	Secret      string
	Topic       string
	Channels    string
	Permissions string
	Created     time.Time
}

// InitClickhouse ...
func InitClickhouse(dsn, table string) (db *DB, err error) {
	var attempts = 60

	db = &DB{Table: table}

	connect, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return db, err
	}

	for {
		err := connect.Ping()

		if err == nil {
			log.Println("database is ready")
			break
		}

		if exception, ok := err.(*clickhouse.Exception); ok {
			log.Printf("[%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}

		attempts--

		if attempts < 1 {
			break
		}

		log.Println("database isn't ready yet, stand by")

		time.Sleep(time.Second)
	}

	db.DB = connect

	return db, nil
}

// AddSecret ...
func (db *DB) AddSecret(secret Secret) (err error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	stmt, _ := tx.Prepare(
		fmt.Sprintf(
			`INSERT INTO
	%s
(uuid, secret, topic, channels, permissions, created)
	VALUES
(?, ?, ?, ?, ?, ?)`,
			db.Table,
		),
	)
	defer stmt.Close()

	if _, err = stmt.Exec(
		secret.UUID,
		secret.Secret,
		secret.Topic,
		secret.Channels,
		secret.Permissions,
		time.Now(),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetSecretsInfo ...
func (db *DB) GetSecretsInfo(secret Secret) (secrets []Secret, err error) {
	rows, err := db.DB.Query(
		fmt.Sprintf(
			`SELECT
	uuid,
	secret,
	topic,
	channels,
	permissions
FROM
	%s
WHERE
	secret = ?;`,
			db.Table,
		),
		secret.Secret,
	)
	if err != nil {
		return secrets, err
	}

	defer rows.Close()

	for rows.Next() {
		var secret Secret

		if err := rows.Scan(&secret.UUID, &secret.Topic, &secret.Channels, &secret.Permissions); err != nil {
			return secrets, err
		}

		secrets = append(secrets, secret)
	}

	if err := rows.Err(); err != nil {
		return secrets, err
	}

	if len(secrets) == 0 {
		return secrets, fmt.Errorf("%s", "secrets not found")
	}

	return secrets, nil
}
