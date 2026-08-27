package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables dari file .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Inisialisasi Router Gin
	r := gin.Default()

	// Contoh route sederhana untuk cek server
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Server ProMaP Backend Berjalan!",
		})
	})

	// Jalankan server di port 8080
	r.Run(":8080")
}