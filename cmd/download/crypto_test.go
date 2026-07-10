package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func encryptWireFormat(t *testing.T, plain []byte, chunkSize int) ([]byte, []byte) {
	t.Helper()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}

	var wire bytes.Buffer
	if _, err := wire.Write([]byte(wireMagic)); err != nil {
		t.Fatal(err)
	}
	var chunkSizeBuf [4]byte
	binary.BigEndian.PutUint32(chunkSizeBuf[:], uint32(chunkSize))
	if _, err := wire.Write(chunkSizeBuf[:]); err != nil {
		t.Fatal(err)
	}

	for offset := 0; offset < len(plain); {
		end := offset + chunkSize
		if end > len(plain) {
			end = len(plain)
		}
		chunk := plain[offset:end]
		offset = end

		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(chunk)))
		if _, err := wire.Write(lenBuf[:]); err != nil {
			t.Fatal(err)
		}

		nonce := make([]byte, gcmNonceSize)
		if _, err := rand.Read(nonce); err != nil {
			t.Fatal(err)
		}
		if _, err := wire.Write(nonce); err != nil {
			t.Fatal(err)
		}
		cipherChunk := gcm.Seal(nil, nonce, chunk, nil)
		if _, err := wire.Write(cipherChunk); err != nil {
			t.Fatal(err)
		}
	}

	return key, wire.Bytes()
}

func TestDecryptAES256GCMFromReaderMatchesWire(t *testing.T) {
	plain := bytes.Repeat([]byte{0xab}, (3*1024*1024)+123)
	key, payload := encryptWireFormat(t, plain, 1024*1024)

	gotBlob, err := DecryptAES256GCMFromWire(key, payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBlob, plain) {
		t.Fatalf("blob decrypt: got %d bytes, want %d", len(gotBlob), len(plain))
	}

	var gotStream bytes.Buffer
	stats, err := DecryptAES256GCMFromReader(key, bytes.NewReader(payload), &gotStream)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotStream.Bytes(), plain) {
		t.Fatalf("stream decrypt: got %d bytes, want %d", gotStream.Len(), len(plain))
	}
	if stats.DecryptedBytes != int64(len(plain)) {
		t.Fatalf("stats decrypted bytes = %d, want %d", stats.DecryptedBytes, len(plain))
	}
}

func TestStreamDownloadDecryptWithPipe(t *testing.T) {
	plain, err := os.ReadFile(filepath.Join("..", "upload", "fabric_test.go"))
	if err != nil {
		t.Skip("fabric_test.go non disponibile:", err)
	}

	key, payload := encryptWireFormat(t, plain, 64*1024)
	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(payload)
		_ = pw.Close()
	}()

	var out bytes.Buffer
	stats, err := DecryptAES256GCMFromReader(key, pr, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatal("output mismatch")
	}
	if stats.DecryptedBytes != int64(len(plain)) {
		t.Fatalf("decrypted bytes = %d, want %d", stats.DecryptedBytes, len(plain))
	}
}
