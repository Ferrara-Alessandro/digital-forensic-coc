package contract

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type shimChaincodeStub interface {
	CreateCompositeKey(objectType string, attributes []string) (string, error)
}

// Nomi delle collezioni private: devono essere uguali a collections_config.json.
const (
	pdcPGPM  = "collezione_PG_PM"
	pdcPMLAB = "collezione_PM_LAB"
)

// Chiave PDC metadati privati file (caso, id file).
func filePrivDataKey(idCaso, id string) string {
	return idCaso + "/" + id
}

// Data e ora della transazione in formato standard.
func registratoIlUTC(ctx contractapi.TransactionContextInterface) string {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil || ts == nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	t, err := ptypes.Timestamp(ts)
	if err != nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
}

// Chi ha inviato la transazione (organizzazione e certificato).
func autoreRegistrante(ctx contractapi.TransactionContextInterface) (string, error) {
	ci := ctx.GetClientIdentity()
	msp, err := ci.GetMSPID()
	if err != nil {
		return "", err
	}
	id, err := ci.GetID()
	if err == nil && id != "" {
		return msp + ":" + id, nil
	}
	return msp, nil
}

// Se il peer non puo' leggere la collezione privata, ignoro l'errore in lettura.
func pdcIgnorabileInLettura(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "private data matching public hash") ||
		strings.Contains(msg, "does not have read access permission")
}

// Dati che passo nel transient quando carico un file: puntatore IPFS e chiave AES.
type transientFilePayload struct {
	Cid           string `json:"cid"`
	ChiaveCifrata string `json:"chiaveCifrata"`
}

func parseTransientFile(raw []byte, label string) (transientFilePayload, error) {
	var p transientFilePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("parse transient %s: %w", label, err)
	}
	if p.Cid == "" {
		return p, fmt.Errorf("cid obbligatorio nel transient %s", label)
	}
	if strings.TrimSpace(p.ChiaveCifrata) == "" {
		return p, fmt.Errorf("chiaveCifrata obbligatoria nel transient %s: il file su IPFS deve essere cifrato", label)
	}
	return p, nil
}

// Chiave di cifratura obbligatoria nel transient.
func requireChiaveCifrata(chiave string) error {
	if strings.TrimSpace(chiave) == "" {
		return fmt.Errorf("chiaveCifrata obbligatoria")
	}
	return nil
}
