package db

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitPostgres подключает к PG
func InitPostgres(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка PG:", err)
	}
	return db
}

// InitMongo подключает к MongoDB
func InitMongo(uri string) *mongo.Client {
	ctx := context.TODO()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Ошибка Mongo:", err)
	}
	return client
}