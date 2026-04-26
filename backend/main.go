package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.TODO()

	// --- 1. ПОДКЛЮЧЕНИЕ К POSTGRES (Реляционная база) ---
	connStr := "postgresql://admin:smart_password@localhost:5432/smart_home_db?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Println("🐘 Подключено к PostgreSQL!")

	// --- 2. ПОДКЛЮЧЕНИЕ К MONGO (Логи) ---
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("🍃 Подключено к MongoDB!")
	collection := mongoClient.Database("smart_home").Collection("telemetry")

	// --- 3. НАСТРОЙКА MQTT ---
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetClientID("smart_backend_main")

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		mac := "A1:B2:C3:D4:E5:F6" // В жизни мы бы вытаскивали это из топика или payload

		// ПРОВЕРКА В POSTGRES: Существует ли такое устройство?
		var deviceID int
		err := db.QueryRow("SELECT id FROM devices WHERE mac_address = $1", mac).Scan(&deviceID)
		
		if err != nil {
			fmt.Printf("⚠️  Отказ! Устройство с MAC %s не найдено в базе.\n", mac)
			return
		}

		fmt.Printf("✅ Устройство опознано (ID: %d). Сохраняю данные...\n", deviceID)

		// Сохраняем в MongoDB
		data := bson.M{
			"device_id": deviceID,
			"topic":     msg.Topic(),
			"value":     string(msg.Payload()),
			"timestamp": time.Now(),
		}
		collection.InsertOne(ctx, data)
		fmt.Println("💾 Данные записаны в историю (Mongo).")
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	client.Subscribe("home/#", 0, nil)

	// --- 4. ЭМУЛЯТОР (Шлем данные раз в 5 сек) ---
	go func() {
		for {
			payload := fmt.Sprintf("%.2f", 20+rand.Float64()*5)
			client.Publish("home/kitchen/temperature", 0, false, payload)
			time.Sleep(5 * time.Second)
		}
	}()

	select {}
}