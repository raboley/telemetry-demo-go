package models

import "time"

type Subscriber struct {
	ID       int       `json:"id"`
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Created  time.Time `json:"created"`
}