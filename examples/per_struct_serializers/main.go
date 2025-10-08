package main

import (
	"fmt"
	"log"
)

// Code generation methods:
//   1. Direct commands (recommended): encx-gen generate .
//   2. Go generate (optional): go generate . (requires //go:generate directive below)
//
//go:generate go run ../../cmd/encx-gen generate .

// Example demonstrating per-struct serializer options using comments

// Default JSON serializer (from encx.yaml configuration)
type User struct {
	Email    string `encx:"encrypt,hash_basic"`
	Password string `encx:"hash_secure"`
}

//encx:options serializer=gob
type HighPerformanceData struct {
	Data        []byte `encx:"encrypt"`
	Metadata    string `encx:"encrypt"`
	SearchIndex string `encx:"hash_basic"`
}

//encx:options serializer=basic
type SimpleConfiguration struct {
	HostPort   string `encx:"encrypt"`
	ApiKey     string `encx:"hash_secure"`
	MaxRetries int    `encx:"encrypt"`
	Enabled    bool   `encx:"encrypt"`
}

//encx:options serializer=json,future_option=value
type ComplexAPIResponse struct {
	UserData   map[string]interface{} `encx:"encrypt"`
	Timestamps []int64                `encx:"encrypt"`
	Checksum   string                 `encx:"hash_basic"`
}

func main() {
	fmt.Println("Per-Struct Serializer Options Example")
	fmt.Println("=====================================")
	fmt.Println()

	fmt.Println("This example demonstrates how to specify different serializers")
	fmt.Println("for different structs using comment-based configuration:")
	fmt.Println()

	fmt.Println("1. User struct:")
	fmt.Println("   - Uses default JSON serializer (from encx.yaml)")
	fmt.Println("   - Good for general-purpose data with web compatibility")
	fmt.Println()

	fmt.Println("2. HighPerformanceData struct:")
	fmt.Println("   - Uses GOB serializer (//encx:options serializer=gob)")
	fmt.Println("   - Better performance and smaller size for Go-to-Go communication")
	fmt.Println()

	fmt.Println("3. SimpleConfiguration struct:")
	fmt.Println("   - Uses Basic serializer (//encx:options serializer=basic)")
	fmt.Println("   - Optimized for primitive types with minimal overhead")
	fmt.Println()

	fmt.Println("4. ComplexAPIResponse struct:")
	fmt.Println("   - Uses JSON serializer with additional future options")
	fmt.Println("   - Shows extensibility for future configuration options")
	fmt.Println()

	fmt.Println("To generate code:")
	fmt.Println("  go generate")
	fmt.Println()
	fmt.Println("Generated files will contain serializer-specific implementations")
	fmt.Println("optimized for each struct's declared serializer type.")

	// Note: In a real application, you would use the generated functions
	// Example usage would be:
	//
	// user := User{Email: "test@example.com", Password: "secret"}
	// processedUser, err := ProcessUser(ctx, crypto, user)
	// if err != nil {
	//     log.Fatal(err)
	// }
	//
	// data := HighPerformanceData{Data: []byte("important"), Metadata: "urgent"}
	// processedData, err := ProcessHighPerformanceData(ctx, crypto, data)
	// if err != nil {
	//     log.Fatal(err)
	// }

	log.Println("Example completed successfully")
}