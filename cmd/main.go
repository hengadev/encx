package main

import (
	"fmt"
	"time"

	"github.com/hengadev/encx"
)

type User struct {
	Email             string    `encx:"encrypt,hash"`
	Password          string    `encx:"encrypt,hashSecure"`
	Age               int       `encx:"encrypt"`
	EncryptedAge      string    `json:"age"`
	SomeTime          time.Time `encx:"encrypt"`
	EncryptedSomeTime string    `json:"some_time"`
	DataEncryptionKey string    `json:"dek" encx:"encrypt"`
	// TODO: Use that field definition (typo to have a better error message)
	// DataEcnryptionKey string `json:"some_time"`
}

func main() {
	user := User{
		Email:             "john.doe@example.com",
		Password:          "rtg9sg73r9hE)90rwa34DS",
		Age:               23,
		SomeTime:          time.Now(),
		DataEncryptionKey: "48545a565e72b10f15a1ea5cf35d5aeabf314eaf2ae1d3ebd4fc16daf8a942c5",
	}
	// fmt.Printf("the person is initially: %#+v\n", user)
	KEK := "fb259f76a783dc70d8e7c73a4c056b496dcb52942248d467e81c931609aef4f7"
	// TODO: use the option pattern for the argon2params thing ?
	encryptor, err := encx.New(KEK, nil)
	if err != nil {
		fmt.Println("error in encryptor configuration:", err)
	}

	if err = encryptor.Encrypt(&user); err != nil {
		fmt.Println("the bool:", err != nil)
		fmt.Printf("fail to encrypt user: %s\n", err)
	}
	// fmt.Printf("the encrypted user: %#+v\n", user)
}
