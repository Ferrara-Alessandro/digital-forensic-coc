// Interrogo Fabric come Admin PG per ottenere cid e chiave.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const aesKeySize = 32

type orgMSP string

const mspPG orgMSP = "PGMSP"

type peerDial struct {
	Address   string
	TLSHost   string
	TLSCACert string
}

type fabricEnv struct {
	PKIRoot string
	PeerPG  peerDial
}

func defaultFabricEnv(pkiRoot string) fabricEnv {
	return fabricEnv{
		PKIRoot: pkiRoot,
		PeerPG: peerDial{
			Address:   "localhost:7051",
			TLSHost:   "peer0.pg.it",
			TLSCACert: filepath.Join(pkiRoot, "peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"),
		},
	}
}

func resolvePKIDir(p string) (string, error) {
	if filepath.IsAbs(p) {
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			return "", fmt.Errorf("pki non trovata o non directory: %s", p)
		}
		return filepath.Clean(p), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for range 12 {
		cand := filepath.Join(dir, p)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return filepath.Abs(cand)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	fallback, _ := filepath.Abs(filepath.Join(wd, p))
	return "", fmt.Errorf("pki non trovata o non directory: %s (cwd=%s; prova -pki assoluto)", fallback, wd)
}

func adminMSPPath(pkiRoot, orgHost string) string {
	return filepath.Join(pkiRoot, "peerOrganizations", orgHost, "users", "Admin@"+orgHost, "msp")
}

func loadCertAndKey(mspDir string) ([]byte, []byte, error) {
	signDir := filepath.Join(mspDir, "signcerts")
	entries, err := os.ReadDir(signDir)
	if err != nil {
		return nil, nil, fmt.Errorf("signcerts: %w", err)
	}
	var certFile string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pem") {
			certFile = filepath.Join(signDir, e.Name())
			break
		}
	}
	if certFile == "" {
		return nil, nil, fmt.Errorf("nessun .pem in %s", signDir)
	}
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(filepath.Join(mspDir, "keystore", "priv_sk"))
	if err != nil {
		return nil, nil, fmt.Errorf("keystore/priv_sk: %w", err)
	}
	return certPEM, keyPEM, nil
}

func grpcConn(peer peerDial) (*grpc.ClientConn, error) {
	caPEM, err := os.ReadFile(peer.TLSCACert)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("CA peer non valida: %s", peer.TLSCACert)
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
		ServerName: peer.TLSHost,
	}
	return grpc.NewClient(peer.Address, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
}

// Mi collego al peer PG per le letture.
func openPGContract(env fabricEnv, channel, chaincode string) (*client.Contract, *client.Gateway, *grpc.ClientConn, error) {
	certPEM, keyPEM, err := loadCertAndKey(adminMSPPath(env.PKIRoot, "pg.it"))
	if err != nil {
		return nil, nil, nil, err
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, nil, err
	}
	id, err := identity.NewX509Identity(string(mspPG), cert)
	if err != nil {
		return nil, nil, nil, err
	}
	pk, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, nil, nil, err
	}
	sign, err := identity.NewPrivateKeySign(pk)
	if err != nil {
		return nil, nil, nil, err
	}

	conn, err := grpcConn(env.PeerPG)
	if err != nil {
		return nil, nil, nil, err
	}
	gw, err := client.Connect(id,
		client.WithSign(sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(45*time.Second),
		client.WithEndorseTimeout(120*time.Second),
		client.WithSubmitTimeout(60*time.Second),
		client.WithCommitStatusTimeout(120*time.Second),
	)
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, err
	}
	contract := gw.GetNetwork(channel).GetContract(chaincode)
	return contract, gw, conn, nil
}

// Chiedo il documento al chaincode e ricavo cid e chiave.
func fetchDocumentoCIDAndKey(ctx context.Context, contract *client.Contract, idCaso, idDocumento string) (string, []byte, error) {
	readOut, err := contract.EvaluateWithContext(ctx, "LeggiDocumento", client.WithArguments(idCaso, idDocumento))
	if err != nil {
		return "", nil, fmt.Errorf("LeggiDocumento: %w", err)
	}
	var doc struct {
		CID           string `json:"cid"`
		ChiaveCifrata string `json:"chiaveCifrata"`
	}
	if err := json.Unmarshal(readOut, &doc); err != nil {
		return "", nil, fmt.Errorf("parse LeggiDocumento: %w", err)
	}
	return decodeCIDAndKey(doc.CID, doc.ChiaveCifrata, "documento", idCaso, idDocumento)
}

// Come sopra ma per un'evidenza.
func fetchEvidenzaCIDAndKey(ctx context.Context, contract *client.Contract, idCaso, idEvidenza string) (string, []byte, error) {
	readOut, err := contract.EvaluateWithContext(ctx, "LeggiEvidenza", client.WithArguments(idCaso, idEvidenza))
	if err != nil {
		return "", nil, fmt.Errorf("LeggiEvidenza: %w", err)
	}
	var ev struct {
		CID           string `json:"cid"`
		ChiaveCifrata string `json:"chiaveCifrata"`
	}
	if err := json.Unmarshal(readOut, &ev); err != nil {
		return "", nil, fmt.Errorf("parse LeggiEvidenza: %w", err)
	}
	return decodeCIDAndKey(ev.CID, ev.ChiaveCifrata, "evidenza", idCaso, idEvidenza)
}

func decodeCIDAndKey(cid, chiaveB64, kind, idCaso, id string) (string, []byte, error) {
	if cid == "" {
		return "", nil, fmt.Errorf("cid assente per %s %s/%s", kind, idCaso, id)
	}
	if chiaveB64 == "" {
		return "", nil, fmt.Errorf("chiaveCifrata assente (PDC non leggibile o %s non cifrata)", kind)
	}
	key, err := base64.StdEncoding.DecodeString(chiaveB64)
	if err != nil {
		return "", nil, fmt.Errorf("decode chiaveCifrata: %w", err)
	}
	if len(key) != aesKeySize {
		return "", nil, fmt.Errorf("chiave AES inattesa: %d byte", len(key))
	}
	return cid, key, nil
}
