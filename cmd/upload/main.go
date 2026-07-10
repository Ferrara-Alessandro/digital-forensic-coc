// Programma da terminale: carico reperto, documenti o evidenze (cifro, IPFS, Fabric).
// Output JSON su stdout.
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

type repertoUploadResult struct {
	Mode           string `json:"mode"`
	IDReperto      string `json:"idReperto"`
	TransactionID  string `json:"transactionId,omitempty"`
	TempoFabricMs  int64  `json:"tempoFabricMs,omitempty"`
	TempoTotaleMs  int64  `json:"tempoTotaleMs"`
}

type documentoUploadResult struct {
	Mode               string `json:"mode"`
	IDDocumento        string `json:"idDocumento"`
	IDCaso             string `json:"idCaso"`
	TipoDocumento      string `json:"tipoDocumento"`
	IDReperto          string `json:"idReperto,omitempty"`
	IngestOrg          string `json:"ingestOrg"`
	CID                string `json:"cid"`
	ChiaveB64          string `json:"chiaveB64"`
	TransactionID      string `json:"transactionId"`
	InputSizeBytes     int64  `json:"inputSizeBytes"`
	EncryptedSizeBytes int    `json:"encryptedSizeBytes,omitempty"`
	TempoEncryptOnlyMs int64  `json:"tempoEncryptOnlyMs,omitempty"`
	TempoIPFSMs        int64  `json:"tempoIpfsMs,omitempty"`
	TempoFabricMs      int64  `json:"tempoFabricMs,omitempty"`
}

type evidenzaUploadResult struct {
	Mode               string `json:"mode"`
	IDEvidenza         string `json:"idEvidenza"`
	IDCaso             string `json:"idCaso"`
	IDReperto          string `json:"idReperto,omitempty"`
	Classe             string `json:"classe,omitempty"`
	IngestOrg          string `json:"ingestOrg"`
	CID                string `json:"cid"`
	ChiaveB64          string `json:"chiaveB64"`
	TransactionID      string `json:"transactionId"`
	InputSizeBytes     int64  `json:"inputSizeBytes"`
	EncryptedSizeBytes int    `json:"encryptedSizeBytes,omitempty"`
	TempoEncryptOnlyMs int64  `json:"tempoEncryptOnlyMs,omitempty"`
	TempoIPFSMs        int64  `json:"tempoIpfsMs,omitempty"`
	TempoFabricMs      int64  `json:"tempoFabricMs,omitempty"`
}

func writeResultJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func normalizeMode(mode string) string {
	return strings.ToLower(strings.TrimSpace(mode))
}

func scegliOrgFirmatariaDocumento(tipo string) (string, error) {
	switch strings.TrimSpace(tipo) {
	case "VERBALE_SOPRALLUOGO", "VERBALE_SEQUESTRO", "VERBALE_RICONSEGNA":
		return "pg", nil
	case "DECRETO_ACCERTAMENTO":
		return "pm", nil
	case "RELAZIONE_TECNICA":
		return "lab", nil
	default:
		return "", fmt.Errorf("tipoDocumento %q: nessun MSP predefinito (usa -ingest-org pg|pm|lab)", tipo)
	}
}

func run() error {
	startTotal := time.Now()
	mode := flag.String("mode", "reperto", "")
	input := flag.String("file", "", "")
	idReperto := flag.String("id-reperto", "", "")
	idCaso := flag.String("id-caso", "", "")
	idAgente := flag.String("id-agente", "", "")
	idDistretto := flag.String("id-distretto", "", "")
	descrizione := flag.String("descrizione-bene", "", "")
	dataOra := flag.String("data-ora-prelievo", "", "")
	ipfsAPI := flag.String("ipfs-api", "http://127.0.0.1:5001", "")
	ipfsTimeout := flag.Duration("ipfs-timeout", 45*time.Second, "")
	encryptChunkMB := flag.Int("encrypt-chunk-mb", 4, "")
	pki := flag.String("pki", filepath.Join("infrastruttura_blockchain", "certificati_pki"), "")
	channel := flag.String("channel", "canale-coc", "")
	chaincode := flag.String("chaincode", "reperto", "")
	submitTimeout := flag.Duration("submit-timeout", 120*time.Second, "")
	skipChaincode := flag.Bool("skip-chaincode", false, "")

	idDocumento := flag.String("id-documento", "", "")
	tipoDocumento := flag.String("tipo-documento", "", "")
	descrizioneDoc := flag.String("descrizione-documento", "", "")
	idRepertoDoc := flag.String("id-reperto-documento", "", "")
	riferimentoEnte := flag.String("riferimento-ente", "", "")

	idEvidenza := flag.String("id-evidenza", "", "")
	descrizioneEvi := flag.String("descrizione-evidenza", "", "")
	idRepertoEvi := flag.String("id-reperto-evidenza", "", "")
	classeEvi := flag.String("classe-evidenza", "", "")

	ingestOrg := flag.String("ingest-org", "", "")

	flag.Parse()

	m := normalizeMode(*mode)
	switch m {
	case "reperto":
		return runReperto(startTotal, runRepertoFlags{
			IDReperto:       *idReperto,
			IDCaso:          *idCaso,
			IDAgente:        *idAgente,
			IDDistretto:     *idDistretto,
			DescrizioneBene: *descrizione,
			DataOra:         *dataOra,
			PKI:             *pki,
			Channel:         *channel,
			Chaincode:       *chaincode,
			SubmitTimeout:   *submitTimeout,
			SkipChaincode:   *skipChaincode,
		})
	case "documento":
		return runDocumento(runDocumentoFlags{
			Input:           *input,
			IDCaso:          *idCaso,
			IDDocumento:     *idDocumento,
			TipoDocumento:   *tipoDocumento,
			IDReperto:       *idRepertoDoc,
			Descrizione:     *descrizioneDoc,
			RiferimentoEnte: *riferimentoEnte,
			IngestOrg:       *ingestOrg,
			IPFSAPI:         *ipfsAPI,
			IPFSTimeout:     *ipfsTimeout,
			EncryptChunkMB:  *encryptChunkMB,
			PKI:             *pki,
			Channel:         *channel,
			Chaincode:       *chaincode,
			SubmitTimeout:   *submitTimeout,
			SkipChaincode:   *skipChaincode,
		})
	case "evidenza":
		return runEvidenza(runEvidenzaFlags{
			Input:          *input,
			IDCaso:         *idCaso,
			IDEvidenza:     *idEvidenza,
			IDReperto:      *idRepertoEvi,
			Descrizione:    *descrizioneEvi,
			Classe:         *classeEvi,
			IngestOrg:      *ingestOrg,
			IPFSAPI:        *ipfsAPI,
			IPFSTimeout:    *ipfsTimeout,
			EncryptChunkMB: *encryptChunkMB,
			PKI:            *pki,
			Channel:        *channel,
			Chaincode:      *chaincode,
			SubmitTimeout:  *submitTimeout,
			SkipChaincode:  *skipChaincode,
		})
	default:
		return fmt.Errorf("-mode non valido: %q (reperto|documento|evidenza)", *mode)
	}
}

type runRepertoFlags struct {
	IDReperto       string
	IDCaso          string
	IDAgente        string
	IDDistretto     string
	DescrizioneBene string
	DataOra         string
	PKI             string
	Channel         string
	Chaincode       string
	SubmitTimeout   time.Duration
	SkipChaincode   bool
}

func runReperto(startTotal time.Time, f runRepertoFlags) error {
	if !f.SkipChaincode {
		for _, req := range []struct {
			val, flag string
		}{
			{f.IDReperto, "-id-reperto"},
			{f.IDCaso, "-id-caso"},
			{f.IDAgente, "-id-agente"},
			{f.IDDistretto, "-id-distretto"},
			{f.DescrizioneBene, "-descrizione-bene"},
		} {
			if strings.TrimSpace(req.val) == "" {
				return fmt.Errorf("%s obbligatorio (mode reperto)", req.flag)
			}
		}
	}
	ts := f.DataOra
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}

	if f.SkipChaincode {
		return writeResultJSON(repertoUploadResult{
			Mode:          "reperto",
			IDReperto:     f.IDReperto,
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})
	}

	pkiAbs, err := resolvePKIDir(f.PKI)
	if err != nil {
		return err
	}
	contract, gw, conn, err := openOrgContract(defaultFabricEnv(pkiAbs), f.Channel, f.Chaincode, "pg")
	if err != nil {
		return fmt.Errorf("fabric gateway: %w", err)
	}
	defer func() { _ = gw.Close() }()
	defer func() { _ = conn.Close() }()

	transient, err := BuildTransientReperto(RepertoPrivatoInput{
		IDCaso:          f.IDCaso,
		IDAgente:        f.IDAgente,
		IDDistretto:     f.IDDistretto,
		DataOraPrelievo: ts,
		DescrizioneBene: f.DescrizioneBene,
	})
	if err != nil {
		return fmt.Errorf("build transient: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.SubmitTimeout)
	defer cancel()

	startFabric := time.Now()
	txID, err := submitCreaReperto(ctx, contract, f.IDReperto, transient)
	if err != nil {
		return fmt.Errorf("submit CreaReperto: %w", err)
	}

	return writeResultJSON(repertoUploadResult{
		Mode:          "reperto",
		IDReperto:     f.IDReperto,
		TransactionID: txID,
		TempoFabricMs: time.Since(startFabric).Milliseconds(),
		TempoTotaleMs: time.Since(startTotal).Milliseconds(),
	})
}

type runDocumentoFlags struct {
	Input           string
	IDCaso          string
	IDDocumento     string
	TipoDocumento   string
	IDReperto       string
	Descrizione     string
	RiferimentoEnte string
	IngestOrg       string
	IPFSAPI         string
	IPFSTimeout     time.Duration
	EncryptChunkMB  int
	PKI             string
	Channel         string
	Chaincode       string
	SubmitTimeout   time.Duration
	SkipChaincode   bool
}

func runDocumento(f runDocumentoFlags) error {
	if strings.TrimSpace(f.Input) == "" {
		return fmt.Errorf("-file obbligatorio (mode documento)")
	}
	if strings.TrimSpace(f.IDCaso) == "" || strings.TrimSpace(f.IDDocumento) == "" || strings.TrimSpace(f.TipoDocumento) == "" {
		return fmt.Errorf("mode documento: -id-caso, -id-documento e -tipo-documento obbligatori")
	}
	if strings.TrimSpace(f.Descrizione) == "" {
		return fmt.Errorf("mode documento: -descrizione-documento obbligatoria")
	}

	orgKey := strings.ToLower(strings.TrimSpace(f.IngestOrg))
	if orgKey == "" {
		var err error
		orgKey, err = scegliOrgFirmatariaDocumento(f.TipoDocumento)
		if err != nil {
			return err
		}
	}

	cid, encRes, fi, tempoIPFSMs, err := uploadEncryptedToIPFS(f.Input, f.IPFSAPI, f.IPFSTimeout, f.EncryptChunkMB)
	if err != nil {
		return err
	}
	defer zeroBytes(encRes.Key)

	out := documentoUploadResult{
		Mode:               "documento",
		IDDocumento:        f.IDDocumento,
		IDCaso:             f.IDCaso,
		TipoDocumento:      f.TipoDocumento,
		IDReperto:          strings.TrimSpace(f.IDReperto),
		IngestOrg:          orgKey,
		CID:                cid,
		ChiaveB64:          base64.StdEncoding.EncodeToString(encRes.Key),
		InputSizeBytes:     fi.Size(),
		EncryptedSizeBytes: int(encRes.EncryptedBytes),
		TempoEncryptOnlyMs: encRes.EncryptOnlyTime.Milliseconds(),
		TempoIPFSMs:        tempoIPFSMs,
	}

	if f.SkipChaincode {
		return writeResultJSON(out)
	}

	transient, err := BuildTransientDocumento(cid, encRes.Key)
	if err != nil {
		return err
	}

	pkiAbs, err := resolvePKIDir(f.PKI)
	if err != nil {
		return err
	}
	contract, gw, conn, err := openOrgContract(defaultFabricEnv(pkiAbs), f.Channel, f.Chaincode, orgKey)
	if err != nil {
		return fmt.Errorf("fabric gateway: %w", err)
	}
	defer func() { _ = gw.Close() }()
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), f.SubmitTimeout)
	defer cancel()

	startFabric := time.Now()
	txID, err := submitRegistraDocumento(ctx, contract, f.IDDocumento, f.IDCaso, f.TipoDocumento,
		out.IDReperto, f.Descrizione, f.RiferimentoEnte, transient)
	if err != nil {
		return fmt.Errorf("submit RegistraDocumentoConTransient: %w", err)
	}
	out.TransactionID = txID
	out.TempoFabricMs = time.Since(startFabric).Milliseconds()
	return writeResultJSON(out)
}

type runEvidenzaFlags struct {
	Input          string
	IDCaso         string
	IDEvidenza     string
	IDReperto      string
	Descrizione    string
	Classe         string
	IngestOrg      string
	IPFSAPI        string
	IPFSTimeout    time.Duration
	EncryptChunkMB int
	PKI            string
	Channel        string
	Chaincode      string
	SubmitTimeout  time.Duration
	SkipChaincode  bool
}

func runEvidenza(f runEvidenzaFlags) error {
	if strings.TrimSpace(f.Input) == "" {
		return fmt.Errorf("-file obbligatorio (mode evidenza)")
	}
	if strings.TrimSpace(f.IDCaso) == "" || strings.TrimSpace(f.IDEvidenza) == "" {
		return fmt.Errorf("mode evidenza: -id-caso e -id-evidenza obbligatori")
	}
	if strings.TrimSpace(f.Descrizione) == "" {
		return fmt.Errorf("mode evidenza: -descrizione-evidenza obbligatoria")
	}

	orgKey := strings.ToLower(strings.TrimSpace(f.IngestOrg))
	if orgKey == "" {
		orgKey = "pg"
	}

	cid, encRes, fi, tempoIPFSMs, err := uploadEncryptedToIPFS(f.Input, f.IPFSAPI, f.IPFSTimeout, f.EncryptChunkMB)
	if err != nil {
		return err
	}
	defer zeroBytes(encRes.Key)

	out := evidenzaUploadResult{
		Mode:               "evidenza",
		IDEvidenza:         f.IDEvidenza,
		IDCaso:             f.IDCaso,
		IDReperto:          strings.TrimSpace(f.IDReperto),
		Classe:             strings.TrimSpace(f.Classe),
		IngestOrg:          orgKey,
		CID:                cid,
		ChiaveB64:          base64.StdEncoding.EncodeToString(encRes.Key),
		InputSizeBytes:     fi.Size(),
		EncryptedSizeBytes: int(encRes.EncryptedBytes),
		TempoEncryptOnlyMs: encRes.EncryptOnlyTime.Milliseconds(),
		TempoIPFSMs:        tempoIPFSMs,
	}

	if f.SkipChaincode {
		return writeResultJSON(out)
	}

	transient, err := BuildTransientEvidenza(cid, encRes.Key)
	if err != nil {
		return err
	}

	pkiAbs, err := resolvePKIDir(f.PKI)
	if err != nil {
		return err
	}
	contract, gw, conn, err := openOrgContract(defaultFabricEnv(pkiAbs), f.Channel, f.Chaincode, orgKey)
	if err != nil {
		return fmt.Errorf("fabric gateway: %w", err)
	}
	defer func() { _ = gw.Close() }()
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), f.SubmitTimeout)
	defer cancel()

	startFabric := time.Now()
	txID, err := submitRegistraEvidenza(ctx, contract, f.IDEvidenza, f.IDCaso, out.IDReperto, f.Descrizione, out.Classe, transient)
	if err != nil {
		return fmt.Errorf("submit RegistraEvidenzaConTransient: %w", err)
	}
	out.TransactionID = txID
	out.TempoFabricMs = time.Since(startFabric).Milliseconds()
	return writeResultJSON(out)
}

// Cifro il file a pezzi e lo carico su IPFS in streaming.
func uploadEncryptedToIPFS(input, ipfsAPI string, ipfsTimeout time.Duration, encryptChunkMB int) (cid string, encRes *StreamEncryptResult, fi os.FileInfo, tempoIPFSMs int64, err error) {
	fi, err = os.Stat(input)
	if err != nil {
		return "", nil, nil, 0, fmt.Errorf("stat file: %w", err)
	}
	chunkSize := encryptChunkMB * 1024 * 1024
	if chunkSize <= 0 {
		return "", nil, nil, 0, fmt.Errorf("-encrypt-chunk-mb deve essere > 0")
	}

	ipfs := NewIPFSClient(ipfsAPI, ipfsTimeout)
	pipeReader, pipeWriter := io.Pipe()
	encResCh := make(chan *StreamEncryptResult, 1)
	encErrCh := make(chan error, 1)
	go func() {
		res, encErr := EncryptFileToWriter(input, pipeWriter, chunkSize)
		if encErr != nil {
			_ = pipeWriter.CloseWithError(encErr)
			encErrCh <- encErr
			return
		}
		encResCh <- res
		_ = pipeWriter.Close()
	}()

	startIPFS := time.Now()
	cid, err = ipfs.AddReader(context.Background(), filepath.Base(input)+".enc", pipeReader)
	if err != nil {
		return "", nil, fi, 0, fmt.Errorf("ipfs add: %w", err)
	}
	tempoIPFSMs = time.Since(startIPFS).Milliseconds()

	select {
	case encRes = <-encResCh:
	case encErr := <-encErrCh:
		return "", nil, fi, tempoIPFSMs, fmt.Errorf("encrypt stream: %w", encErr)
	}
	return cid, encRes, fi, tempoIPFSMs, nil
}
