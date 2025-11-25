package utils

import (
	"log"
	"time"
)

func LogUserAction(action string, userID int) {
	log.Printf("User action: %s, UserID: %d, Timestamp: %s", action, userID, time.Now().Format(time.RFC3339))
}
