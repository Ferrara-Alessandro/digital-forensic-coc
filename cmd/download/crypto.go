// Decifro file caricati con il programma di upload.
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	gcmNonceSize = 12
	wireMagic    = "EV2"
)

type streamDecryptStats struct {
	DecryptedBytes int64
}

func DecryptAES256GCMFromReader(key []byte, r io.Reader, w io.Writer) (*streamDecryptStats, error) {
	magic := make([]byte, len(wireMagic))
	if err := readFull(r, magic); err != nil {
		return nil, fmt.Errorf("read wire magic: %w", err)
	}
	if string(magic) != wireMagic {
		return nil, fmt.Errorf("formato cifrato non valido (atteso %s)", wireMagic)
	}

	var chunkSizeBuf [4]byte
	if err := readFull(r, chunkSizeBuf[:]); err != nil {
		return nil, fmt.Errorf("read chunk size: %w", err)
	}
	chunkSize := int(binary.BigEndian.Uint32(chunkSizeBuf[:]))
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

	stats := &streamDecryptStats{}
	lenBuf := make([]byte, 4)
	nonce := make([]byte, gcmNonceSize)
	cipherBuf := make([]byte, chunkSize+overhead)

	for {
		if err := readFull(r, lenBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("read chunk len: %w", err)
		}
		plainLen := int(binary.BigEndian.Uint32(lenBuf))
		if plainLen <= 0 || plainLen > chunkSize {
			return nil, fmt.Errorf("lunghezza chunk non valida: %d", plainLen)
		}

		if err := readFull(r, nonce); err != nil {
			return nil, fmt.Errorf("read chunk nonce: %w", err)
		}

		cipherLen := plainLen + overhead
		if cipherLen > len(cipherBuf) {
			cipherBuf = make([]byte, cipherLen)
		}
		if err := readFull(r, cipherBuf[:cipherLen]); err != nil {
			return nil, fmt.Errorf("read chunk ciphertext: %w", err)
		}

		chunkPlain, err := gcm.Open(nil, nonce, cipherBuf[:cipherLen], nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt chunk: %w", err)
		}
		n, err := w.Write(chunkPlain)
		if err != nil {
			return nil, fmt.Errorf("write plaintext chunk: %w", err)
		}
		if n != len(chunkPlain) {
			return nil, io.ErrShortWrite
		}
		stats.DecryptedBytes += int64(n)
	}

	return stats, nil
}

// Dal blob in memoria ricostruisco il file originale (compatibilita' e test).
func DecryptAES256GCMFromWire(key, payload []byte) ([]byte, error) {
	var out bytes.Buffer
	stats, err := DecryptAES256GCMFromReader(key, bytes.NewReader(payload), &out)
	if err != nil {
		return nil, err
	}
	if int64(out.Len()) != stats.DecryptedBytes {
		return nil, fmt.Errorf("dimensione output incoerente")
	}
	return out.Bytes(), nil
}

func readFull(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return err
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
