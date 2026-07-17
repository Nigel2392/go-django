package encryption

var encryptionRegistry = make(map[string]func(key []byte, fallbackKeys [][]byte) (Encryption, error))

func GetCrypto(which string, key []byte, fallbackKeys [][]byte) (Encryption, error) {
	var f, ok = encryptionRegistry[which]
	if !ok {
		return nil, nil
	}

	return f(key, fallbackKeys)
}

func RegisterCrypto(which string, newEncryption func(key []byte, fallbackKeys [][]byte) (Encryption, error)) (overwritten bool) {
	_, overwritten = encryptionRegistry[which]
	encryptionRegistry[which] = newEncryption
	return overwritten
}
