package dsn

import (
	"fmt"
	"net/url"
)

// AmpqDsnBuilder builds an AMQP URL (for passsword special characters encoding)
func AmqpDsnBuilder(username, password, host, port, vhost string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, encodedPassword, host, port, vhost)
}

// RedisDsnBuilder builds a Redis URL (for passsword special characters encoding)
func RedisDsnBuilder(password, host, port, db string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("redis://:%s@%s:%s", encodedPassword, host, port)
}

// RedisDsnBuilderWithDB builds a Redis URL with a DB (for passsword special characters encoding)
func RedisDsnBuilderWithDB(password, host, port, db string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("redis://:%s@%s:%s/%s", encodedPassword, host, port, db)
}

// PostgresDsnBuilder builds a Postgres URL (for passsword special characters encoding)
func PostgresDsnBuilder(username, password, host, port, db string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, encodedPassword, host, port, db)
}

// MysqlDsnBuilder builds a MySQL URL (for passsword special characters encoding)
func MysqlDsnBuilder(username, password, host, port, db string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, encodedPassword, host, port, db)
}

// ElasticsearchDsnBuilder builds an Elasticsearch URL (for passsword special characters encoding)
func ElasticsearchDsnBuilder(username, password, host, port, scheme string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("http://%s:%s@%s:%s", username, encodedPassword, host, port)
}

// MongodbDsnBuilder builds a MongoDB URL (for passsword special characters encoding)
func MongodbDsnBuilder(username, password, host, port, db string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", username, encodedPassword, host, port, db)
}

// KafkaDsnBuilder builds a Kafka URL (for passsword special characters encoding)
func KafkaDsnBuilder(username, password, host, port string) string {
	encodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("%s:%s@%s:%s", username, encodedPassword, host, port)
}
