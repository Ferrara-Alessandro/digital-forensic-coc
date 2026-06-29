// Connessione a Fabric Gateway: stessa logica di cmd/upload, senza IPFS.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

type orgMSP string

const (
	mspPG  orgMSP = "PGMSP"
	mspPM  orgMSP = "PMMSP"
	mspLAB orgMSP = "LABMSP"
)

type peerDial struct {
	Address   string
	TLSHost   string
	TLSCACert string
}

type fabricEnv struct {
	PKIRoot string
	Peers   map[string]peerDial
}

func defaultFabricEnv(pkiRoot string) fabricEnv {
	return fabricEnv{
		PKIRoot: pkiRoot,
		Peers: map[string]peerDial{
			"pg": {
				Address:   "localhost:7051",
				TLSHost:   "peer0.pg.it",
				TLSCACert: filepath.Join(pkiRoot, "peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"),
			},
			"pm": {
				Address:   "localhost:8051",
				TLSHost:   "peer0.pm.it",
				TLSCACert: filepath.Join(pkiRoot, "peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt"),
			},
			"lab": {
				Address:   "localhost:9051",
				TLSHost:   "peer0.lab.it",
				TLSCACert: filepath.Join(pkiRoot, "peerOrganizations/lab.it/peers/peer0.lab.it/tls/ca.crt"),
			},
		},
	}
}

// Cerco la cartella certificati generata da cryptogen, risalendo l'albero.
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
	return "", fmt.Errorf("pki non trovata: %s (cwd=%s; usa -pki assoluto)", fallback, wd)
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

var orgProfile = map[string]struct {
	MSP     orgMSP
	OrgHost string
}{
	"pg":  {mspPG, "pg.it"},
	"pm":  {mspPM, "pm.it"},
	"lab": {mspLAB, "lab.it"},
}

// Apro il contratto Fabric firmando come Admin dell'organizzazione indicata.
// Routing del peer fisico:
//   - pg  → peer PG (ha collezione_PG_PM)
//   - pm  → peer PG (gossip asincrono: PM non ha ancora collezione_PG_PM)
//   - lab → peer LAB (ha appena scritto collezione_PM_LAB, i dati ci sono)
func openOrgContract(env fabricEnv, channel, chaincode, org string) (*client.Contract, *client.Gateway, *grpc.ClientConn, error) {
	orgKey := strings.ToLower(strings.TrimSpace(org))
	if orgKey == "" {
		orgKey = "pg"
	}
	prof, ok := orgProfile[orgKey]
	if !ok {
		return nil, nil, nil, fmt.Errorf("org non supportata: %q (pg, pm o lab)", org)
	}
	peerKey := orgKey
	if orgKey == "pm" {
		peerKey = "pg"
	}
	peer, ok := env.Peers[peerKey]
	if !ok {
		return nil, nil, nil, fmt.Errorf("peer non configurato per org %q", peerKey)
	}
	certPEM, keyPEM, err := loadCertAndKey(adminMSPPath(env.PKIRoot, prof.OrgHost))
	if err != nil {
		return nil, nil, nil, err
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, nil, err
	}
	id, err := identity.NewX509Identity(string(prof.MSP), cert)
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
	conn, err := grpcConn(peer)
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

// Invio una transazione e attendo il commit; restituisco il txID.
func submitAndWait(ctx context.Context, contract *client.Contract, fn string, args ...string) (string, error) {
	_, commit, err := contract.SubmitAsync(fn, client.WithArguments(args...))
	if err != nil {
		return "", fmt.Errorf("%s: %w", fn, err)
	}
	status, err := commit.StatusWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("%s commit: %w", fn, err)
	}
	if !status.Successful {
		return status.TransactionID, fmt.Errorf("%s commit non valido: codice %d", fn, status.Code)
	}
	return status.TransactionID, nil
}

// Leggo dati dal ledger senza creare transazione (evaluate).
func evaluateQuery(ctx context.Context, contract *client.Contract, fn string, args ...string) ([]byte, error) {
	result, err := contract.EvaluateWithContext(ctx, fn, client.WithArguments(args...))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	return result, nil
}
