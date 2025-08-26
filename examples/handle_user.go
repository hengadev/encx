package main

import (
	"context"
	"fmt"

	"github.com/hengadev/encx"
)

func handleUser(ctx context.Context, crypto *encx.Crypto) {

	user := NewUser()

	fmt.Println("the initial user is:", user)
	// TODO: the program panics if I have a pointer to pointer ie if I user a &user when user is of type &User
	if err := crypto.ProcessStruct(ctx, user); err != nil {
		fmt.Println("failing to encrypt user:", err)
		return
	}
	// copyUser := user.copyUser()
	// fmt.Printf("the new user encrypted is: %#+v\n", user)

	encryptUser := user.copyUser()
	reset(user)

	// fmt.Printf("the dek encrypted is: %q\n", user.DEKEncrypted)
	if err := crypto.DecryptStruct(ctx, user); err != nil {
		fmt.Println("failing to decrypt user:", err)
		return
	}
	fmt.Printf("the final user decrypted is: %#+v\n", user)
	// TODO: Now do key rotation thing
	// if err := crypto.RotateKEK(ctx); err != nil {
	// 	fmt.Println("rotation key error:", err)
	// 	return
	// }

	if err := crypto.DecryptStruct(ctx, encryptUser); err != nil {
		fmt.Println("fail to decrypt user:", err)
		return
	}
	// fmt.Println("after changing the key I get : %#+v", encryptUser)

}
