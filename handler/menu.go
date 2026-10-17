package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/model"

	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	DB        *sql.DB
	UploadDir string
}

func NewMenuHandler(db *sql.DB, uploadDir string) *MenuHandler {
	return &MenuHandler{
		DB:        db,
		UploadDir: uploadDir,
	}
}

// ensureColumn memastikan kolom image_url ada pada tabel menu jika belum ditambahkan
func (h *MenuHandler) ensureColumn() {
	_, _ = h.DB.Exec("ALTER TABLE menu ADD COLUMN image_url TEXT DEFAULT ''")
}

// SanitizeFilename membersihkan nama file agar aman disimpan di filesystem
func SanitizeFilename(name string) string {
	name = filepath.Base(name)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	var cleaned strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			cleaned.WriteRune(r)
		}
	}
	res := cleaned.String()
	if res == "" {
		res = "image"
	}
	if len(res) > 30 {
		res = res[:30]
	}
	return res
}

// MenuPayload menyimpan data hasil ekstraksi request
type MenuPayload struct {
	Name        string
	Description string
	Icon        string
	ImageURL    string
	HasImageURL bool
}

// extractPayload mengekstrak data dari request baik berupa JSON maupun multipart/form-data (file upload/link URL)
func (h *MenuHandler) extractPayload(c *gin.Context) (*MenuPayload, error) {
	contentType := c.ContentType()
	p := &MenuPayload{}

	// 1. Jika request berupa multipart/form-data (upload file atau form)
	if strings.Contains(contentType, "multipart/form-data") {
		p.Name = strings.TrimSpace(c.PostForm("name"))
		p.Description = strings.TrimSpace(c.PostForm("description"))
		p.Icon = strings.TrimSpace(c.PostForm("icon"))

		// Cek apakah field text image_url atau image disediakan di form
		if val, exists := c.GetPostForm("image_url"); exists {
			p.ImageURL = strings.TrimSpace(val)
			p.HasImageURL = true
		} else if val, exists := c.GetPostForm("image"); exists {
			p.ImageURL = strings.TrimSpace(val)
			p.HasImageURL = true
		}

		// Cek apakah ada file gambar yang diunggah
		file, fileErr := c.FormFile("image")
		if fileErr != nil {
			file, fileErr = c.FormFile("image_file")
		}
		if fileErr != nil {
			file, fileErr = c.FormFile("image_url") // kemungkinan form field file bernama image_url
		}
		if fileErr != nil {
			file, fileErr = c.FormFile("file")
		}

		// Jika file ditemukan, simpan ke folder uploads
		if file != nil {
			ext := strings.ToLower(filepath.Ext(file.Filename))
			if ext == "" {
				ext = ".png"
			}
			filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), SanitizeFilename(file.Filename), ext)
			dst := filepath.Join(h.UploadDir, filename)
			if saveErr := c.SaveUploadedFile(file, dst); saveErr != nil {
				return nil, fmt.Errorf("gagal menyimpan file gambar: %w", saveErr)
			}
			p.ImageURL = "/uploads/" + filename
			p.HasImageURL = true
		}

		return p, nil
	}

	// 2. Jika request berupa application/json
	var jsonInput struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Icon        string  `json:"icon"`
		ImageURL    *string `json:"image_url"`
		Image       *string `json:"image"`
	}

	if bindErr := c.ShouldBindJSON(&jsonInput); bindErr == nil {
		p.Name = strings.TrimSpace(jsonInput.Name)
		p.Description = strings.TrimSpace(jsonInput.Description)
		p.Icon = strings.TrimSpace(jsonInput.Icon)
		if jsonInput.ImageURL != nil {
			p.ImageURL = strings.TrimSpace(*jsonInput.ImageURL)
			p.HasImageURL = true
		} else if jsonInput.Image != nil {
			p.ImageURL = strings.TrimSpace(*jsonInput.Image)
			p.HasImageURL = true
		}
		return p, nil
	}

	// 3. Fallback jika x-www-form-urlencoded
	p.Name = strings.TrimSpace(c.PostForm("name"))
	p.Description = strings.TrimSpace(c.PostForm("description"))
	p.Icon = strings.TrimSpace(c.PostForm("icon"))
	if val, exists := c.GetPostForm("image_url"); exists {
		p.ImageURL = strings.TrimSpace(val)
		p.HasImageURL = true
	} else if val, exists := c.GetPostForm("image"); exists {
		p.ImageURL = strings.TrimSpace(val)
		p.HasImageURL = true
	}

	return p, nil
}

// GetAll menangani GET /api/menu (200, daftar menu)
func (h *MenuHandler) GetAll(c *gin.Context) {
	query := "SELECT id, name, description, COALESCE(icon, ''), COALESCE(image_url, ''), created_at FROM menu ORDER BY id ASC"
	rows, err := h.DB.Query(query)
	if err != nil && strings.Contains(err.Error(), "no such column: image_url") {
		h.ensureColumn()
		rows, err = h.DB.Query(query)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	menus := make([]model.Menu, 0)
	for rows.Next() {
		var m model.Menu
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Icon, &m.ImageURL, &m.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		m.Image = m.ImageURL
		menus = append(menus, m)
	}

	c.JSON(http.StatusOK, menus)
}

// GetByID menangani GET /api/menu/:id (200, satu menu; 404)
func (h *MenuHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	}

	query := "SELECT id, name, description, COALESCE(icon, ''), COALESCE(image_url, ''), created_at FROM menu WHERE id = ?"
	var m model.Menu
	err = h.DB.QueryRow(query, id).
		Scan(&m.ID, &m.Name, &m.Description, &m.Icon, &m.ImageURL, &m.CreatedAt)

	if err != nil && strings.Contains(err.Error(), "no such column: image_url") {
		h.ensureColumn()
		err = h.DB.QueryRow(query, id).
			Scan(&m.ID, &m.Name, &m.Description, &m.Icon, &m.ImageURL, &m.CreatedAt)
	}

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	m.Image = m.ImageURL
	c.JSON(http.StatusOK, m)
}

// Create menangani POST /api/menu (201, menu baru; 400)
// Mendukung image_url berupa link URL maupun upload file gambar
func (h *MenuHandler) Create(c *gin.Context) {
	payload, err := h.extractPayload(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if payload.Name == "" || payload.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name dan description wajib diisi"})
		return
	}

	insertQuery := "INSERT INTO menu (name, description, icon, image_url) VALUES (?, ?, ?, ?)"
	res, err := h.DB.Exec(insertQuery, payload.Name, payload.Description, payload.Icon, payload.ImageURL)
	if err != nil && strings.Contains(err.Error(), "no such column: image_url") {
		h.ensureColumn()
		res, err = h.DB.Exec(insertQuery, payload.Name, payload.Description, payload.Icon, payload.ImageURL)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newID, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var created model.Menu
	err = h.DB.QueryRow("SELECT id, name, description, COALESCE(icon, ''), COALESCE(image_url, ''), created_at FROM menu WHERE id = ?", newID).
		Scan(&created.ID, &created.Name, &created.Description, &created.Icon, &created.ImageURL, &created.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	created.Image = created.ImageURL
	c.JSON(http.StatusCreated, created)
}

// Update menangani PUT /api/menu/:id (200, menu diubah; 404)
// Mendukung edit image_url berupa link URL baru maupun upload file gambar baru
func (h *MenuHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	}

	// Cek keberadaan data dan ambil image_url saat ini
	var existingImageURL string
	checkQuery := "SELECT COALESCE(image_url, '') FROM menu WHERE id = ?"
	err = h.DB.QueryRow(checkQuery, id).Scan(&existingImageURL)
	if err != nil && strings.Contains(err.Error(), "no such column: image_url") {
		h.ensureColumn()
		err = h.DB.QueryRow(checkQuery, id).Scan(&existingImageURL)
	}

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	payload, err := h.extractPayload(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if payload.Name == "" || payload.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name dan description wajib diisi"})
		return
	}

	// Jika payload tidak menyertakan image_url baru, pertahankan yang sudah ada
	finalImageURL := payload.ImageURL
	if !payload.HasImageURL {
		finalImageURL = existingImageURL
	}

	updateQuery := "UPDATE menu SET name = ?, description = ?, icon = ?, image_url = ? WHERE id = ?"
	_, err = h.DB.Exec(updateQuery, payload.Name, payload.Description, payload.Icon, finalImageURL, id)
	if err != nil && strings.Contains(err.Error(), "no such column: image_url") {
		h.ensureColumn()
		_, err = h.DB.Exec(updateQuery, payload.Name, payload.Description, payload.Icon, finalImageURL, id)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var updated model.Menu
	err = h.DB.QueryRow("SELECT id, name, description, COALESCE(icon, ''), COALESCE(image_url, ''), created_at FROM menu WHERE id = ?", id).
		Scan(&updated.ID, &updated.Name, &updated.Description, &updated.Icon, &updated.ImageURL, &updated.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated.Image = updated.ImageURL
	c.JSON(http.StatusOK, updated)
}

// Delete menangani DELETE /api/menu/:id (204; 404)
func (h *MenuHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	}

	var exists int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM menu WHERE id = ?", id).Scan(&exists)
	if err != nil || exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu tidak ditemukan"})
		return
	}

	_, err = h.DB.Exec("DELETE FROM menu WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
