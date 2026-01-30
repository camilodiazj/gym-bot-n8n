package entity

import "time"

// User represents a user in the system
type User struct {
	ID              string    `json:"id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	CelNumber       string    `json:"cel_number"`
	FullPhoneNumber string    `json:"full_phone_number"`
	Timezone        string    `json:"timezone"`
	CreatedAt       time.Time `json:"created_at"`
}

// NewUser creates a new User entity
func NewUser(id, fullName, email, celNumber, fullPhoneNumber, timezone string) *User {
	return &User{
		ID:              id,
		FullName:        fullName,
		Email:           email,
		CelNumber:       celNumber,
		FullPhoneNumber: fullPhoneNumber,
		Timezone:        timezone,
		CreatedAt:       time.Now(),
	}
}
