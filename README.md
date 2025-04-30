# encx

`encx` is a Go package for handling field-level encryption and hashing in structs. It supports:

- **AES-GCM encryption** for securing sensitive data.
- **Argon2id hashing** for secure password storage.
- **SHA-256 hashing** for fast, non-reversible identifiers.

## 🚀 Usage

Simply tag struct fields with `encx` to specify how they should be processed.

```go
type User struct {
    Email    string `encx:"hash_basic"`
    Password string `encx:"hash_secure"`
    Address  string `encx:"encrypt"`
}
``````

## Important: Version Control (.gitignore)

When using the `encx` package, it's **highly recommended** to add the following line to your project's `.gitignore` file:

```gitignore
.encx/
```

## 🚧 TODOs

- [ ] implement example for different key management services: 
    - [X] HashiCorp Vault
    - [ ] AWS KMS
    - [ ] Azure Key Vault
    - [ ] Google Cloud KMS
    - [ ] Thales CipherTrust (formerly Vormetric)
    - [ ] AWS CloudHSM

## 📚 Documentation

