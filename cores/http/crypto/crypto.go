package crypto

type ICrypto interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}
