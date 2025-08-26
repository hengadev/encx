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
	Email             string    `encx:"encrypt,hash_secure" json:"email"`
	EmailHash         string    `json:"email_hash"`
	EmailEncrypted    []byte    `json:"email_encrypted"`
	Password          string    `encx:"encrypt,hash_secure"`
	PasswordHash      string    `json:"password_hash"`
	PasswordEncrypted []byte    `json:"password_encrypted"`
	Age               int       `encx:"encrypt"`
	AgeEncrypted      []byte    `json:"age"`
	OneField          int       `encx:"hash_basic"`
	OneFieldHash      string    `json:"one_field_hash"`
	SomeTime          time.Time `encx:"encrypt"`
	SomeTimeEncrypted []byte    `json:"some_time"`
	DEK               []byte    `json:"dek" encx:"encrypt"`
	DEKEncrypted      []byte    `json:"dek_encrypted" encx:"encrypt"`
	KeyVersion        int       `json:"version"`
}

func NewUser() *User {
	return &User{
		Email:             "john.doe@example.com",
		EmailHash:         "",
		EmailEncrypted:    nil,
		Password:          "thisisaweakpassword",
		PasswordHash:      "",
		PasswordEncrypted: nil,
		Age:               27,
		AgeEncrypted:      nil,
		OneField:          12399343,
		OneFieldHash:      "",
		SomeTime:          time.Now(),
		SomeTimeEncrypted: nil,
		DEK:               nil,
		// DEK:          []byte("c278cdbd59cb1bf79c52491295b94bb2"),
		DEKEncrypted: nil,
		KeyVersion:   1,
	}
}

// something to make that thing work brother
func (u *User) copyUser() *User {
	return &User{
		Email:             u.Email,
		EmailHash:         u.EmailHash,
		EmailEncrypted:    u.EmailEncrypted,
		Password:          u.Password,
		PasswordHash:      u.PasswordHash,
		PasswordEncrypted: u.PasswordEncrypted,
		Age:               u.Age,
		AgeEncrypted:      u.AgeEncrypted,
		OneField:          u.OneField,
		OneFieldHash:      u.OneFieldHash,
		SomeTime:          u.SomeTime,
		SomeTimeEncrypted: u.SomeTimeEncrypted,
		DEK:               u.DEK,
		DEKEncrypted:      u.DEKEncrypted,
		KeyVersion:        u.KeyVersion,
	}
}

type OTP struct {
	Email        string  `json:"email" encx:"hash_basic"`
	EmailHash    string  `json:"-"`
	Data         OTPData `json:"-"`
	DEK          []byte  `json:"-"`
	DEKEncrypted []byte  `json:"-"`
	KeyVersion   int     `json:"-"`
	// Data         struct {
	// 	Code          string    `json:"code" validate:"len=6" encx:"encrypt"`
	// 	CodeEncrypted []byte    `json:"-" validate:"len=6"`
	// 	Attempts      int       `json:"attempts"`
	// 	ExpiresAt     time.Time `json:"expires_at"`
	// 	CreatedAt     time.Time `json:"created_at"`
	// }
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
