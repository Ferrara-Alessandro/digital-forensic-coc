// Programma per scaricare e decifrare un documento o evidenza registrati.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type readResult struct {
	Mode               string `json:"mode"`
	Org                string `json:"org"`
	IDCaso             string `json:"idCaso,omitempty"`
	IDDocumento        string `json:"idDocumento,omitempty"`
	IDEvidenza         string `json:"idEvidenza,omitempty"`
	CID                string `json:"cid"`
	OutputPath         string `json:"outputPath"`
	EncryptedSizeBytes int    `json:"encryptedSizeBytes"`
	DecryptedSizeBytes int    `json:"decryptedSizeBytes"`
	TempoFabricMs      int64  `json:"tempoFabricMs"`
	TempoIPFSMs        int64  `json:"tempoIpfsMs"`
	TempoDecryptMs     int64  `json:"tempoDecryptMs"`
	TempoWriteMs       int64  `json:"tempoWriteMs"`
	TempoTotaleMs      int64  `json:"tempoTotaleMs"`
}

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// Leggo da Fabric, scarico da IPFS e salvo il file in chiaro.
func run() error {
	startTotal := time.Now()
	mode := flag.String("mode", "documento", "")
	idCaso := flag.String("id-caso", "", "")
	idDocumento := flag.String("id-documento", "", "")
	idEvidenza := flag.String("id-evidenza", "", "")
	ipfsAPI := flag.String("ipfs-api", "http://127.0.0.1:5001", "")
	ipfsTimeout := flag.Duration("ipfs-timeout", 45*time.Second, "")
	pki := flag.String("pki", filepath.Join("infrastruttura_blockchain", "certificati_pki"), "")
	channel := flag.String("channel", "canale-coc", "")
	chaincode := flag.String("chaincode", "reperto", "")
	readTimeout := flag.Duration("read-timeout", 120*time.Second, "")
	outDir := flag.String("out-dir", "downloads", "")
	outFile := flag.String("out-file", "", "")
	noWrite := flag.Bool("no-write", false, "")
	org := flag.String("org", "pg", "")
	flag.Parse()

	m := strings.ToLower(strings.TrimSpace(*mode))
	if m == "" {
		m = "documento"
	}

	pkiAbs, err := resolvePKIDir(*pki)
	if err != nil {
		return err
	}
	contract, gw, conn, err := openOrgContract(defaultFabricEnv(pkiAbs), *channel, *chaincode, *org)
	if err != nil {
		return fmt.Errorf("fabric gateway: %w", err)
	}
	defer func() { _ = gw.Close() }()
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), *readTimeout)
	defer cancel()

	caso := strings.TrimSpace(*idCaso)
	var fileID string
	startFabric := time.Now()

	var cid string
	var key []byte

	switch m {
	case "documento":
		fileID = strings.TrimSpace(*idDocumento)
		if caso == "" || fileID == "" {
			return fmt.Errorf("mode documento: -id-caso e -id-documento obbligatori")
		}
		cid, key, err = fetchDocumentoCIDAndKey(ctx, contract, caso, fileID)
	case "evidenza":
		fileID = strings.TrimSpace(*idEvidenza)
		if caso == "" || fileID == "" {
			return fmt.Errorf("mode evidenza: -id-caso e -id-evidenza obbligatori")
		}
		cid, key, err = fetchEvidenzaCIDAndKey(ctx, contract, caso, fileID)
	default:
		return fmt.Errorf("-mode non valido: %q (documento|evidenza)", *mode)
	}
	if err != nil {
		return err
	}
	tempoFabricMs := time.Since(startFabric).Milliseconds()
	defer zeroBytes(key)

	ipfs := NewIPFSClient(*ipfsAPI, *ipfsTimeout)
	startIPFS := time.Now()
	encrypted, err := ipfs.CatBytes(context.Background(), cid)
	if err != nil {
		return err
	}
	tempoIPFSMs := time.Since(startIPFS).Milliseconds()

	startDecrypt := time.Now()
	plaintext, err := DecryptAES256GCMFromWire(key, encrypted)
	if err != nil {
		return err
	}
	tempoDecryptMs := time.Since(startDecrypt).Milliseconds()

	outputPath := ""
	tempoWriteMs := int64(0)
	if !*noWrite {
		outputPath = *outFile
		if outputPath == "" {
			if err := os.MkdirAll(*outDir, 0o755); err != nil {
				return fmt.Errorf("mkdir out-dir: %w", err)
			}
			outputPath = filepath.Join(*outDir, fileID+".bin")
		}
		startWrite := time.Now()
		if err := os.WriteFile(outputPath, plaintext, 0o600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		tempoWriteMs = time.Since(startWrite).Milliseconds()
	}

	out := readResult{
		Mode:               m,
		Org:                strings.ToLower(strings.TrimSpace(*org)),
		IDCaso:             caso,
		CID:                cid,
		OutputPath:         outputPath,
		EncryptedSizeBytes: len(encrypted),
		DecryptedSizeBytes: len(plaintext),
		TempoFabricMs:      tempoFabricMs,
		TempoIPFSMs:        tempoIPFSMs,
		TempoDecryptMs:     tempoDecryptMs,
		TempoWriteMs:       tempoWriteMs,
		TempoTotaleMs:      time.Since(startTotal).Milliseconds(),
	}
	if m == "documento" {
		out.IDDocumento = fileID
	} else {
		out.IDEvidenza = fileID
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
