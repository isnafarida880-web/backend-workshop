package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/handler"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

// corsMiddleware membatasi akses CORS hanya dari alamat dev server Vue
func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			if allowedOrigins[origin] {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		c.Next()
	}
}

// ensureSchema memastikan tabel dan kolom image_url tersedia pada database
func ensureSchema(db *sql.DB) {
	// Buat tabel jika belum ada
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS menu (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		icon TEXT,
		image_url TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Printf("Catatan schema: %v", err)
	}

	// Migrasi otomatis kolom image_url jika database sebelumnya dibuat tanpa kolom ini
	_, _ = db.Exec("ALTER TABLE menu ADD COLUMN image_url TEXT DEFAULT ''")
}

func main() {
	// Muat file .env jika ada
	_ = godotenv.Load(".env", "backend/.env")

	port := os.Getenv("PORT")
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if strings.TrimSpace(dbPath) == "" {
		dbPath = "data/app.db"
	}

	// Buka koneksi database SQLite dengan driver modernc.org/sqlite
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Gagal membuka database %s: %v", dbPath, err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Gagal menghubungkan ke database: %v", err)
	}

	// Pastikan skema tabel dan kolom image_url sudah ada
	ensureSchema(db)

	// Tentukan dan buat direktori penyimpanan file gambar upload
	uploadDir := "uploads"
	if _, err := os.Stat("backend"); err == nil {
		uploadDir = filepath.Join("backend", "uploads")
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Gagal membuat direktori uploads (%s): %v", uploadDir, err)
	}

	// Inisialisasi handler
	menuHandler := handler.NewMenuHandler(db, uploadDir)

	// Buat router Gin
	r := gin.Default()

	// Pasang middleware CORS (hanya dev server Vue)
	r.Use(corsMiddleware())

	// Aktifkan static file serving agar gambar bisa diakses publik melalui browser
	r.Static("/uploads", uploadDir)
	r.Static("/static", uploadDir)

	// Endpoint CRUD persis seperti alamat di blueprint (/api/menu)
	menuRoutes := r.Group("/api/menu")
	{
		menuRoutes.GET("", menuHandler.GetAll)
		menuRoutes.GET("/:id", menuHandler.GetByID)
		menuRoutes.POST("", menuHandler.Create)
		menuRoutes.PUT("/:id", menuHandler.Update)
		menuRoutes.DELETE("/:id", menuHandler.Delete)
	}

	// Endpoint alias /api/services untuk kompatibilitas
	servicesRoutes := r.Group("/api/services")
	{
		servicesRoutes.GET("", menuHandler.GetAll)
		servicesRoutes.GET("/:id", menuHandler.GetByID)
		servicesRoutes.POST("", menuHandler.Create)
		servicesRoutes.PUT("/:id", menuHandler.Update)
		servicesRoutes.DELETE("/:id", menuHandler.Delete)
	}

	// Endpoint opsional upload file mandiri
	r.POST("/api/upload", func(c *gin.Context) {
		file, err := c.FormFile("image")
		if err != nil {
			file, err = c.FormFile("file")
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file gambar wajib diunggah"})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == "" {
			ext = ".png"
		}
		filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), handler.SanitizeFilename(file.Filename), ext)
		dst := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan gambar"})
			return
		}

		imageURL := "/uploads/" + filename
		c.JSON(http.StatusOK, gin.H{
			"url":       imageURL,
			"image_url": imageURL,
		})
	})

	addr := ":" + port
	if strings.HasPrefix(port, ":") {
		addr = port
	}

	fmt.Printf("Server berjalan di port %s dengan database %s dan static uploads di %s\n", port, dbPath, uploadDir)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
