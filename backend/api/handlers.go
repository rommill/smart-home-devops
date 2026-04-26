package api

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
)

// GetTelemetry возвращает список комнат с учетом RLS
func GetTelemetry(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")

		// Активируем RLS для текущей сессии
		_, err := db.Exec(fmt.Sprintf("SET app.current_user_id = '%s'", userID))
		if err != nil {
			c.JSON(500, gin.H{"error": "RLS failed"})
			return
		}

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
			"status":     "online",
			"user_id":    userID,
			"your_rooms": rooms,
		})
	}
}