package utils

import (
	"log"
	"time"
)

func LogUserAction(action string, userID int) {
	// Имитация асинхронного логирования
	time.Sleep(10 * time.Millisecond) // Имитация задержки
	log.Printf("User action: %s, UserID: %d, Timestamp: %s", action, userID, time.Now().Format(time.RFC3339))
}
