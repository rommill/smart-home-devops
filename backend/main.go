package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.TODO()

	// --- 1. ПОДКЛЮЧЕНИЯ К БАЗАМ ---
	connStr := "postgresql://admin:smart_password@localhost:5432/smart_home_db?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	mongoColl := mongoClient.Database("smart_home").Collection("telemetry")

	// --- 2. НАСТРОЙКА MQTT (Collector) ---
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetClientID("smart_backend_main")
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		// (Тут остается твоя логика проверки MAC и сохранения в Mongo, которую мы писали)
		mac := "A1:B2:C3:D4:E5:F6"
		var deviceID int
		err := db.QueryRow("SELECT id FROM devices WHERE mac_address = $1", mac).Scan(&deviceID)
		if err == nil {
			mongoColl.InsertOne(ctx, bson.M{
				"device_id": deviceID,
				"value":     string(msg.Payload()),
				"timestamp": time.Now(),
			})
		}
	})
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() == nil {
		mqttClient.Subscribe("home/#", 0, nil)
	}

	// --- 3. HTTP API СЕРВЕР ---
	r := gin.Default()

	r.GET("/telemetry", func(c *gin.Context) {
		userID := c.Query("user_id") // Имитируем авторизацию через параметр

		// АКТИВАЦИЯ RLS: говорим Postgres, какой пользователь делает запрос
		_, err := db.Exec(fmt.Sprintf("SET app.current_user_id = '%s'", userID))
		if err != nil {
			c.JSON(500, gin.H{"error": "RLS activation failed"})
			return
		}

		// Теперь запрашиваем список комнат. Благодаря RLS, 
		// Postgres сам отфильтрует только те комнаты, которые принадлежат userID
		rows, err := db.Query("SELECT name FROM rooms")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var rooms []string
		for rows.Next() {
			var name string
			rows.Scan(&name)
			rooms = append(rooms, name)
		}

		c.JSON(200, gin.H{
			"status": "online",
			"user":   userID,
			"your_rooms": rooms,
		})
	})

	// --- 4. ЗАПУСК ВСЕГО ---
	go func() {
		for {
			payload := fmt.Sprintf("%.2f", 20+rand.Float64()*5)
			mqttClient.Publish("home/kitchen/temperature", 0, false, payload)
			time.Sleep(10 * time.Second)
		}
	}()

	fmt.Println("🚀 API запущен на http://localhost:8080")
	r.Run(":8080") 
}