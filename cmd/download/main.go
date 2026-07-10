// Programma per scaricare e decifrare un documento o evidenza registrati.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	TempoTotaleMs      int64  `json:"tempoTotaleMs"`
}

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

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
	skipChaincode := flag.Bool("skip-chaincode", false, "")
	cidFlag := flag.String("cid", "", "")
	keyB64 := flag.String("key-b64", "", "")
	flag.Parse()

	m := strings.ToLower(strings.TrimSpace(*mode))
	if m == "" {
		m = "documento"
	}

	caso := strings.TrimSpace(*idCaso)
	var fileID string
	var cid string
	var key []byte
	var tempoFabricMs int64

	if *skipChaincode {
		if strings.TrimSpace(*cidFlag) == "" || strings.TrimSpace(*keyB64) == "" {
			return fmt.Errorf("-skip-chaincode: -cid e -key-b64 obbligatori")
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(*keyB64))
		if err != nil {
			return fmt.Errorf("decode key-b64: %w", err)
		}
		cid = strings.TrimSpace(*cidFlag)
		key = decoded
		fileID = strings.TrimSpace(*idEvidenza)
		if fileID == "" {
			fileID = strings.TrimSpace(*idDocumento)
		}
		if fileID == "" {
			fileID = "offline"
		}
	} else {
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

		startFabric := time.Now()

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
		tempoFabricMs = time.Since(startFabric).Milliseconds()
	}
	defer zeroBytes(key)

	outputPath := ""
	var outWriter io.Writer = io.Discard
	if !*noWrite {
		outputPath = *outFile
		if outputPath == "" {
			if err := os.MkdirAll(*outDir, 0o755); err != nil {
				return fmt.Errorf("mkdir out-dir: %w", err)
			}
			outputPath = filepath.Join(*outDir, fileID+".bin")
		}
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("open output: %w", err)
		}
		defer func() { _ = f.Close() }()
		outWriter = f
	}

	ipfs := NewIPFSClient(*ipfsAPI, *ipfsTimeout)
	streamRes, err := streamDownloadDecrypt(context.Background(), ipfs, cid, key, outWriter)
	if err != nil {
		return err
	}

	out := readResult{
		Mode:               m,
		Org:                strings.ToLower(strings.TrimSpace(*org)),
		IDCaso:             caso,
		CID:                cid,
		OutputPath:         outputPath,
		EncryptedSizeBytes: int(streamRes.EncryptedSizeBytes),
		DecryptedSizeBytes: int(streamRes.DecryptedSizeBytes),
		TempoFabricMs:      tempoFabricMs,
		TempoIPFSMs:        streamRes.TempoIPFSMs,
		TempoDecryptMs:     streamRes.TempoDecryptMs,
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
