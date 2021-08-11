package clickhouse

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go"
)

// Secret ...
type Secret struct {
	UUID        string
	Topic       string
	Channels    string
	Permissions string
	Created     time.Time
}

// InitClickhouse ...
func InitClickhouse(dsn string) (db *sql.DB, err error) {
	var attempts = 60

	connect, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
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

	return connect, nil
}

// AddSecret ...
func AddSecret(db *sql.DB, secret Secret) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, _ := tx.Prepare("INSERT INTO ipmn.secrets (uuid, topic, channels, permissions, created) VALUES (?, ?, ?, ?, ?)")
	defer stmt.Close()

	if _, err = stmt.Exec(
		secret.UUID,
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
func GetSecretsInfo(db *sql.DB, secret Secret) (secrets []Secret, err error) {
	rows, err := db.Query(`
SELECT
	uuid,
	topic,
	channels,
	permissions
FROM
	ipmn.secrets
WHERE
	uuid = ?;`, secret.UUID)
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
