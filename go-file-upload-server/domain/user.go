package domain

import (
	"errors"
	"time"
	"github.com/google/uuid"
)

type User struct {
	Id            	string    `json:"id"`
	FirstName      	string    `json:"firstName"`
	LastName       	string    `json:"lastName"`
	Email		 	string    `json:"email"`
	PasswordHash   	string    `json:"passwordHash"`
	CreatedAt     	time.Time `json:"createdAt"`
	UpdatedAt     	time.Time `json:"updatedAt"`
}

func NewUser(firstName, lastName, email string, passwordHash string) (*User, error) {
	if firstName == "" {
		return nil, errors.New("first name cannot be empty")
	}
	if lastName == "" {
		return nil, errors.New("last name cannot be empty")
	}
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	if passwordHash == "" {
		return nil, errors.New("password hash cannot be empty")
	}

	return &User{
		Id:          uuid.NewString(),
		FirstName:   firstName,
		LastName:    lastName,
		Email:         email,
		PasswordHash: passwordHash,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}