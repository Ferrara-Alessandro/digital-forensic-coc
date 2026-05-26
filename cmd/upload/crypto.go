// Qui cifro i file a blocchi con AES; il formato e' lo stesso che usa il download.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	aesKeySize   = 32
	gcmNonceSize = 12
	wireMagic    = "EV2"

	defaultEncryptChunkSize = 4 * 1024 * 1024
)

type StreamEncryptResult struct {
	Key             []byte
	PlainBytes      int64
	EncryptedBytes  int64
	EncryptOnlyTime time.Duration
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Leggo il file originale e scrivo la versione cifrata.
func EncryptFileToWriter(inputPath string, writer io.Writer, chunkSize int) (*StreamEncryptResult, error) {
	if chunkSize <= 0 {
		chunkSize = defaultEncryptChunkSize
	}
	if chunkSize > int(^uint32(0)) {
		return nil, fmt.Errorf("chunk size troppo grande: %d", chunkSize)
	}

	in, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("open input: %w", err)
	}
	defer func() { _ = in.Close() }()

	key := make([]byte, aesKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("rand key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	var encryptedBytes int64
	writeCount := func(b []byte) error {
		n, err := writer.Write(b)
		encryptedBytes += int64(n)
		if err != nil {
			return err
		}
		if n != len(b) {
			return io.ErrShortWrite
		}
		return nil
	}

	startEncrypt := time.Now()
	if err := writeCount([]byte(wireMagic)); err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("write wire magic: %w", err)
	}
	var chunkSizeBuf [4]byte
	binary.BigEndian.PutUint32(chunkSizeBuf[:], uint32(chunkSize))
	if err := writeCount(chunkSizeBuf[:]); err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("write chunk size: %w", err)
	}

	var plainBytes int64
	buf := make([]byte, chunkSize)
	for {
		n, readErr := io.ReadFull(in, buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			zeroBytes(key)
			return nil, fmt.Errorf("read chunk: %w", readErr)
		}
		if n == 0 {
			break
		}
		plainChunk := buf[:n]
		plainBytes += int64(n)

		nonce := make([]byte, gcmNonceSize)
		if _, err := rand.Read(nonce); err != nil {
			zeroBytes(key)
			return nil, fmt.Errorf("rand nonce: %w", err)
		}
		cipherChunk := gcm.Seal(nil, nonce, plainChunk, nil)

		var nbuf [4]byte
		binary.BigEndian.PutUint32(nbuf[:], uint32(n))
		if err := writeCount(nbuf[:]); err != nil {
			zeroBytes(key)
			return nil, fmt.Errorf("write chunk len: %w", err)
		}
		if err := writeCount(nonce); err != nil {
			zeroBytes(key)
			return nil, fmt.Errorf("write chunk nonce: %w", err)
		}
		if err := writeCount(cipherChunk); err != nil {
			zeroBytes(key)
			return nil, fmt.Errorf("write chunk ciphertext: %w", err)
		}
		zeroBytes(plainChunk)

		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return &StreamEncryptResult{
		Key:             key,
		PlainBytes:      plainBytes,
		EncryptedBytes:  encryptedBytes,
		EncryptOnlyTime: time.Since(startEncrypt),
	}, nil
}

func decryptEncryptedStream(key, payload []byte) ([]byte, error) {
	const headerLen = len(wireMagic) + 4
	if len(payload) < headerLen {
		return nil, fmt.Errorf("payload troppo corto")
	}
	if string(payload[:len(wireMagic)]) != wireMagic {
		return nil, fmt.Errorf("formato cifrato non valido")
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
