package model

// Menu merepresentasikan entitas menu sesuai spesifikasi blueprint dan dukungan image_url
type Menu struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	ImageURL    string `json:"image_url"`
	Image       string `json:"image"`
	CreatedAt   string `json:"created_at"`
}

// CreateMenuInput payload untuk membuat menu baru via JSON
type CreateMenuInput struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Icon        string  `json:"icon"`
	ImageURL    *string `json:"image_url"`
	Image       *string `json:"image"`
}

// UpdateMenuInput payload untuk memperbarui menu via JSON
type UpdateMenuInput struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Icon        string  `json:"icon"`
	ImageURL    *string `json:"image_url"`
	Image       *string `json:"image"`
}
