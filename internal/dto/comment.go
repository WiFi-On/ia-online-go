package dto

import "time"

type AddCommentDTO struct {
	IdLead  int64  `json:"lead_id" validate:"omitempty"`
	Comment string `json:"comment" validate:"omitempty"`
}

type CommentDTO struct {
	ID        int64      `json:"id"`
	Manager   bool       `json:"is_manager"`
	Text      string     `json:"text"`
	CreatedAt *time.Time `json:"created_at"`
}
