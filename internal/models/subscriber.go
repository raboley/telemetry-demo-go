package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrSubscriberNotFound = errors.New("subscriber not found")

type Subscriber struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateSubscriberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

func NewSubscriber(email, name string) *Subscriber {
	now := time.Now()
	return &Subscriber{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}