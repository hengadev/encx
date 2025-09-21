package main

import (
	"fmt"
	"time"
)

func reset(user *User) {
	user.Email = ""
	user.Password = ""
	user.Age = 0
}

type User struct {
	Email    string    `encx:"encrypt,hash_secure" json:"email"`
	Password string    `encx:"encrypt,hash_secure"`
	Age      int       `encx:"encrypt"`
	OneField int       `encx:"hash_basic"`
	SomeTime time.Time `encx:"encrypt"`

	// No companion fields needed! Code generation creates separate UserEncx struct
}

func NewUser() *User {
	return &User{
		Email:    "john.doe@example.com",
		Password: "thisisaweakpassword",
		Age:      27,
		OneField: 12399343,
		SomeTime: time.Now(),
	}
}

// copyUser creates a copy of the user
func (u *User) copyUser() *User {
	return &User{
		Email:    u.Email,
		Password: u.Password,
		Age:      u.Age,
		OneField: u.OneField,
		SomeTime: u.SomeTime,
	}
}

type OTP struct {
	Email string  `json:"email" encx:"hash_basic"`
	Data  OTPData `json:"-"`

	// No companion fields needed! Code generation creates separate OTPEncx struct
}

// func (o *OTP) somefunc() any { return o.Value
// }

// How does my encrypt thing works with encapsulated structs
type OTPData struct {
	Code          string    `json:"code" validate:"len=6" encx:"encrypt"`
	CodeEncrypted []byte    `json:"-" validate:"len=6"`
	Attempts      int       `json:"attempts"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (o *OTP) reset() {
	o.Email = ""
	o.Data.Code = ""
	o.DEK = nil
}

func (o *OTP) print() {
	fmt.Printf("Email: %s\nEmailHash: %s\nCode: %s\nCodeEncrypted: %#+v\nDEK: %#+v\nDEKEncrypted: %#+v\n", o.Email, o.EmailHash, o.Data.Code, o.Data.CodeEncrypted, o.DEK, o.DEKEncrypted)
}
