package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/providers/hashicorpvault"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	kms, err := hashicorpvault.New()
	if err != nil {
		fmt.Println("creating vault:", err)
		return
	}
	ctx := context.Background()
	crypto, err := encx.New(
		ctx,
		kms,
		"leviosa-app-key",
		"secret/data/pepper",
	)
	if err != nil {
		fmt.Println("creating crypto:", err)
		return
	}

	// handleUser(ctx, crypto)
	otp := &OTP{
		Email: "jean.dupont@gmail.com",
		Data: OTPData{
			Code:      "593901",
			Attempts:  1,
			ExpiresAt: time.Now().Add(15 * time.Minute),
			CreatedAt: time.Now(),
		},
	}

	// The code is not encrypted
	if err := crypto.ProcessStruct(ctx, otp); err != nil {
		fmt.Println("failing to encrypt OTP:", err)
		return
	}
	otp.print()

	// reset the fields that I need to set after the decryption
	otp.reset()
	fmt.Println("AFTER THE RESET:")
	otp.print()

	if err := crypto.DecryptStruct(ctx, otp); err != nil {
		fmt.Println("failing to decrypt otp:", err)
		return
	}
	otp.print()
}
