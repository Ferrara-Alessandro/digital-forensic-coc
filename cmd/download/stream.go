// Pipeline IPFS -> pipe -> decifratura, simmetrica a cifratura -> pipe -> IPFS in upload.
package main

import (
	"context"
	"fmt"
	"io"
	"time"
)

type streamDownloadResult struct {
	EncryptedSizeBytes int64
	DecryptedSizeBytes int64
	TempoIPFSMs        int64
	TempoDecryptMs     int64
}

func streamDownloadDecrypt(
	ctx context.Context,
	ipfs *IPFSClient,
	cid string,
	key []byte,
	out io.Writer,
) (*streamDownloadResult, error) {
	body, err := ipfs.OpenCat(ctx, cid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	pr, pw := io.Pipe()
	ipfsDone := make(chan error, 1)
	var encryptedBytes int64
	var tempoIPFSMs int64

	go func() {
		startIPFS := time.Now()
		n, copyErr := io.Copy(pw, body)
		encryptedBytes = n
		tempoIPFSMs = time.Since(startIPFS).Milliseconds()
		ipfsDone <- pw.CloseWithError(copyErr)
	}()

	startDecrypt := time.Now()
	stats, decErr := DecryptAES256GCMFromReader(key, pr, out)
	tempoDecryptMs := time.Since(startDecrypt).Milliseconds()
	_ = pr.Close()

	ipfsErr := <-ipfsDone
	if decErr == nil && ipfsErr != nil {
		decErr = fmt.Errorf("ipfs copy: %w", ipfsErr)
	}
	if decErr != nil {
		return nil, decErr
	}

	return &streamDownloadResult{
		EncryptedSizeBytes: encryptedBytes,
		DecryptedSizeBytes: stats.DecryptedBytes,
		TempoIPFSMs:        tempoIPFSMs,
		TempoDecryptMs:     tempoDecryptMs,
	}, nil
}
