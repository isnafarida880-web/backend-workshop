package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// resolvePath mencari file dari daftar kandidat path yang ada
func resolvePath(candidates ...string) string {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func main() {
	// Lokasi file schema.sql dan seed.sql
	schemaPath := resolvePath("database/schema.sql", "backend/database/schema.sql")
	seedPath := resolvePath("database/seed.sql", "backend/database/seed.sql")

	// Lokasi database data/app.db
	dbPath := "data/app.db"
	if strings.HasPrefix(schemaPath, "backend") {
		dbPath = filepath.Join("backend", "data", "app.db")
	}

	// Pastikan folder direktori database dibuat
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Gagal membuat direktori database (%s): %v", dbDir, err)
	}

	// Buka koneksi database dengan driver modernc.org/sqlite
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Gagal membuka database %s: %v", dbPath, err)
	}
	defer db.Close()

	// Baca dan jalankan schema.sql
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Gagal membaca file schema (%s): %v", schemaPath, err)
	}

	if _, err := db.Exec(string(schemaContent)); err != nil {
		log.Fatalf("Gagal mengeksekusi schema.sql: %v", err)
	}

	// Pastikan kolom image_url ada pada tabel menu
	_, _ = db.Exec("ALTER TABLE menu ADD COLUMN image_url TEXT DEFAULT ''")

	// Baca dan jalankan seed.sql jika ada
	seedContent, err := os.ReadFile(seedPath)
	if err == nil {
		seedSQL := strings.TrimSpace(string(seedContent))
		if len(seedSQL) > 0 {
			if _, err := db.Exec(seedSQL); err != nil {
				log.Fatalf("Gagal mengeksekusi seed.sql: %v", err)
			}
		}
	}

	// Tampilkan info kolom tabel menu untuk verifikasi
	rows, err := db.Query("PRAGMA table_info(menu)")
	if err == nil {
		fmt.Println("Kolom pada tabel menu:")
		for rows.Next() {
			var cid, notnull, pk int
			var name, ctype string
			var dflt sql.NullString
			_ = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
			fmt.Printf("- %s (%s)\n", name, ctype)
		}
		rows.Close()
	}

	// Hitung jumlah baris pada tabel services / menu
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM services").Scan(&count)
	if err != nil {
		err = db.QueryRow("SELECT COUNT(*) FROM menu").Scan(&count)
	}
	if err != nil {
		log.Fatalf("Gagal menghitung jumlah baris: %v", err)
	}

	// Cetak jumlah baris sesuai instruksi
	fmt.Printf("Jumlah baris di tabel services: %d\n", count)
	fmt.Printf("Jumlah baris di tabel menu: %d\n", count)
}
