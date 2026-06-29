// Decifro file caricati con il programma di upload.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
)

const (
	gcmNonceSize = 12
	wireMagic    = "EV2"
)

// Dal blob scaricato ricostruisco il file originale.
func DecryptAES256GCMFromWire(key, payload []byte) ([]byte, error) {
	const headerLen = len(wireMagic) + 4
	if len(payload) < headerLen {
		return nil, fmt.Errorf("payload troppo corto")
	}
	if string(payload[:len(wireMagic)]) != wireMagic {
		return nil, fmt.Errorf("formato cifrato non valido (atteso %s)", wireMagic)
	}

	chunkSize := int(binary.BigEndian.Uint32(payload[len(wireMagic):headerLen]))
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size non valido: %d", chunkSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	overhead := gcm.Overhead()

	out := make([]byte, 0, len(payload))
	i := headerLen
	for i < len(payload) {
		if i+4 > len(payload) {
			return nil, fmt.Errorf("stream tronco: lunghezza chunk assente")
		}
		plainLen := int(binary.BigEndian.Uint32(payload[i : i+4]))
		i += 4
		if plainLen <= 0 || plainLen > chunkSize {
			return nil, fmt.Errorf("lunghezza chunk non valida: %d", plainLen)
		}
		if i+gcmNonceSize > len(payload) {
			return nil, fmt.Errorf("stream tronco: nonce assente")
		}
		nonce := payload[i : i+gcmNonceSize]
		i += gcmNonceSize

		cipherLen := plainLen + overhead
		if i+cipherLen > len(payload) {
			return nil, fmt.Errorf("stream tronco: ciphertext incompleto")
		}
		chunkCipher := payload[i : i+cipherLen]
		i += cipherLen

		chunkPlain, err := gcm.Open(nil, nonce, chunkCipher, nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt chunk: %w", err)
		}
		out = append(out, chunkPlain...)
	}

	return out, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
