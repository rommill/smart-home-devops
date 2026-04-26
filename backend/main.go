package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"smart-home/api" 
	"smart-home/db"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	ctx := context.TODO()

	// 1. Инициализация баз
	pgDB := db.InitPostgres("postgresql://admin:smart_password@localhost:5432/smart_home_db?sslmode=disable")
	mongoClient := db.InitMongo("mongodb://localhost:27017")
	mongoColl := mongoClient.Database("smart_home").Collection("telemetry")

	// 2. MQTT 
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("smart_backend_main")
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		var deviceID int
		err := pgDB.QueryRow("SELECT id FROM devices WHERE mac_address = $1", "A1:B2:C3:D4:E5:F6").Scan(&deviceID)
		if err == nil {
			mongoColl.InsertOne(ctx, bson.M{"device_id": deviceID, "value": string(msg.Payload()), "timestamp": time.Now()})
		}
	})
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() == nil {
		mqttClient.Subscribe("home/#", 0, nil)
	}

	// 3. API 
	r := gin.Default()
	r.GET("/telemetry", api.GetTelemetry(pgDB)) // Просто вызываем функцию из пакета

	// 4. Запуск эмулятора
	go func() {
		for {
			mqttClient.Publish("home/kitchen/temperature", 0, false, fmt.Sprintf("%.2f", 20+rand.Float64()*5))
			time.Sleep(10 * time.Second)
		}
	}()

	r.Run(":8080")
}