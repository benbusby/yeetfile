package crypto

import (
	"errors"
	"log"
	"os"

	"yeetfile/lang"
)

var CLIKeyEnvVar = "YEETFILE_CLI_KEY"
var KeysNotIngestedError = errors.New(lang.I18n.T("cli.crypto.error.notingested"))

type CryptoCtx struct {
	EncryptionKey []byte
	DecryptionKey []byte
	EncryptFunc   CryptFunc
	DecryptFunc   CryptFunc
}

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func ReadCLIKey() []byte {
	value, exists := os.LookupEnv(CLIKeyEnvVar)
	if !exists || len(value) == 0 {
		return nil
	}

	return []byte(value)
}

// IngestKeys takes the private and public keys and stores them as pubKey
// and privKey
func IngestKeys(privateKey, publicKey []byte) KeyPair {
	return KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}

// DeriveVaultCryptoContext decrypts a vault item's specific key using the key
// sequence returned from the server and returns the key alongside the proper
// functions for encrypting and decrypting content
func (kp KeyPair) DeriveVaultCryptoContext(keySequence [][]byte) (CryptoCtx, error) {
	if kp.PublicKey == nil || kp.PrivateKey == nil {
		return CryptoCtx{}, KeysNotIngestedError
	}

	var decryptedFolderKey []byte
	var encryptKey []byte
	var err error
	var decryptFunc CryptFunc
	var encryptFunc CryptFunc
	if len(keySequence) > 0 {
		decryptedFolderKey, err = kp.UnwindKeySequence(keySequence)
		encryptKey = decryptedFolderKey
		decryptFunc = DecryptChunk
		encryptFunc = EncryptChunk
	} else {
		decryptedFolderKey = kp.PrivateKey
		encryptKey = kp.PublicKey
		decryptFunc = DecryptRSA
		encryptFunc = EncryptRSA
	}

	return CryptoCtx{
		EncryptionKey: encryptKey,
		DecryptionKey: decryptedFolderKey,
		EncryptFunc:   encryptFunc,
		DecryptFunc:   decryptFunc,
	}, err
}

// UnwindKeySequence takes the key sequence for a vault folder and decrypts
// each one in order, returning the final key which can be used to decrypt the
// current folder key
func (kp KeyPair) UnwindKeySequence(keySequence [][]byte) ([]byte, error) {
	var parentKey []byte
	var err error
	for _, key := range keySequence {
		if parentKey == nil {
			parentKey, err = DecryptRSA(kp.PrivateKey, key)
			if err != nil {
				log.Println("Error decrypting root folder key")
				return nil, err
			}

			continue
		}

		parentKey, err = DecryptChunk(parentKey, key)
		if err != nil {
			log.Println("Error decrypting folder key")
			return nil, err
		}
	}

	return parentKey, nil
}
