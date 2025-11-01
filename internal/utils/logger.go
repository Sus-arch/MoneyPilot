package utils

import (
	"log"
)

func Info(msg string) {
	log.Println("ℹ️ ", msg)
}

func Error(err error) {
	log.Println("❌ ", err)
}
