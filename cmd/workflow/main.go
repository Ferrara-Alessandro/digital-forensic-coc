// Client per le operazioni di ciclo di vita del reperto su Fabric (senza IPFS).
// Ogni modalità corrisponde a una funzione del chaincode: avanzamento stato,
// lettura reperto, storia transazioni.
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

// Risultato generico per le operazioni che cambiano stato.
type workflowResult struct {
	Mode          string `json:"mode"`
	IDReperto     string `json:"idReperto"`
	Org           string `json:"org"`
	TransactionID string `json:"transactionId,omitempty"`
	TempoFabricMs int64  `json:"tempoFabricMs,omitempty"`
	TempoTotaleMs int64  `json:"tempoTotaleMs"`
}

// Risultato per le query (lettura senza transazione).
type queryResult struct {
	Mode          string          `json:"mode"`
	IDReperto     string          `json:"idReperto,omitempty"`
	IDCaso        string          `json:"idCaso,omitempty"`
	IDDocumento   string          `json:"idDocumento,omitempty"`
	IDEvidenza    string          `json:"idEvidenza,omitempty"`
	Org           string          `json:"org"`
	Payload       json.RawMessage `json:"payload"`
	TempoTotaleMs int64           `json:"tempoTotaleMs"`
}

func writeJSON(v any) error {
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

func run() error {
	startTotal := time.Now()

	mode := flag.String("mode", "", "leggi-reperto | storia-reperto | leggi-documento | leggi-evidenza | richiedi-analisi | avvia-trasporto | ricevi-laboratorio | completa-analisi | prepara-riconsegna | deposita-sede")
	idReperto := flag.String("id-reperto", "", "")
	org := flag.String("org", "", "pg | pm | lab (default: scelto in base al mode)")
	pki := flag.String("pki", filepath.Join("infrastruttura_blockchain", "certificati_pki"), "")
	channel := flag.String("channel", "canale-coc", "")
	chaincode := flag.String("chaincode", "reperto", "")
	submitTimeout := flag.Duration("submit-timeout", 120*time.Second, "")

	// Argomenti per leggi-documento / leggi-evidenza
	idCaso := flag.String("id-caso", "", "")
	idDocumento := flag.String("id-documento", "", "")
	idEvidenza := flag.String("id-evidenza", "", "")

	// Argomenti per richiedi-analisi
	idLab := flag.String("id-lab", "", "")
	tipoAnalisi := flag.String("tipo-analisi", "", "")
	cidDecreto := flag.String("cid-decreto", "", "")
	chiaveDecreto := flag.String("chiave-decreto", "", "")

	// Argomenti per avvia-trasporto
	idAgentePG := flag.String("id-agente-pg", "", "")

	// Argomenti per ricevi-laboratorio
	idLaboratorio := flag.String("id-laboratorio", "", "")

	// Argomenti per completa-analisi
	cidRelazione := flag.String("cid-relazione", "", "")
	chiaveRelazione := flag.String("chiave-relazione", "", "")

	// Argomenti per deposita-sede
	cidVerbaleRiconsegna := flag.String("cid-verbale-riconsegna", "", "")
	chiaveVerbaleRiconsegna := flag.String("chiave-verbale-riconsegna", "", "")

	flag.Parse()

	m := strings.ToLower(strings.TrimSpace(*mode))
	if m == "" {
		return fmt.Errorf("-mode obbligatorio: leggi-reperto | storia-reperto | leggi-documento | leggi-evidenza | richiedi-analisi | avvia-trasporto | ricevi-laboratorio | completa-analisi | prepara-riconsegna | deposita-sede")
	}
	if m != "leggi-documento" && m != "leggi-evidenza" && strings.TrimSpace(*idReperto) == "" {
		return fmt.Errorf("-id-reperto obbligatorio")
	}

	// Scelgo l'org predefinita in base al mode se non specificata.
	orgKey := strings.ToLower(strings.TrimSpace(*org))
	if orgKey == "" {
		switch m {
		case "richiedi-analisi":
			orgKey = "pm"
		case "ricevi-laboratorio", "completa-analisi", "prepara-riconsegna":
			orgKey = "lab"
		default:
			orgKey = "pg"
		}
	}

	pkiAbs, err := resolvePKIDir(*pki)
	if err != nil {
		return err
	}
	contract, gw, conn, err := openOrgContract(defaultFabricEnv(pkiAbs), *channel, *chaincode, orgKey)
	if err != nil {
		return fmt.Errorf("fabric gateway: %w", err)
	}
	defer func() { _ = gw.Close() }()
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), *submitTimeout)
	defer cancel()

	switch m {

	case "leggi-reperto":
		start := time.Now()
		data, err := evaluateQuery(ctx, contract, "ReadReperto", *idReperto)
		if err != nil {
			return err
		}
		return writeJSON(queryResult{
			Mode:          m,
			IDReperto:     *idReperto,
			Org:           orgKey,
			Payload:       json.RawMessage(data),
			TempoTotaleMs: time.Since(start).Milliseconds(),
		})

	case "storia-reperto":
		start := time.Now()
		data, err := evaluateQuery(ctx, contract, "OttieniStoriaReperto", *idReperto)
		if err != nil {
			return err
		}
		return writeJSON(queryResult{
			Mode:          m,
			IDReperto:     *idReperto,
			Org:           orgKey,
			Payload:       json.RawMessage(data),
			TempoTotaleMs: time.Since(start).Milliseconds(),
		})

	case "leggi-documento":
		if strings.TrimSpace(*idCaso) == "" || strings.TrimSpace(*idDocumento) == "" {
			return fmt.Errorf("mode leggi-documento: -id-caso e -id-documento obbligatori")
		}
		start := time.Now()
		data, err := evaluateQuery(ctx, contract, "LeggiDocumento", *idCaso, *idDocumento)
		if err != nil {
			return err
		}
		return writeJSON(queryResult{
			Mode: m, IDCaso: *idCaso, IDDocumento: *idDocumento,
			Org: orgKey, Payload: json.RawMessage(data),
			TempoTotaleMs: time.Since(start).Milliseconds(),
		})

	case "leggi-evidenza":
		if strings.TrimSpace(*idCaso) == "" || strings.TrimSpace(*idEvidenza) == "" {
			return fmt.Errorf("mode leggi-evidenza: -id-caso e -id-evidenza obbligatori")
		}
		start := time.Now()
		data, err := evaluateQuery(ctx, contract, "LeggiEvidenza", *idCaso, *idEvidenza)
		if err != nil {
			return err
		}
		return writeJSON(queryResult{
			Mode: m, IDCaso: *idCaso, IDEvidenza: *idEvidenza,
			Org: orgKey, Payload: json.RawMessage(data),
			TempoTotaleMs: time.Since(start).Milliseconds(),
		})

	case "richiedi-analisi":
		if *idLab == "" || *tipoAnalisi == "" || *cidDecreto == "" || *chiaveDecreto == "" {
			return fmt.Errorf("mode richiedi-analisi: -id-lab, -tipo-analisi, -cid-decreto e -chiave-decreto obbligatori")
		}
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "RichiediAnalisi", *idReperto, *idLab, *tipoAnalisi, *cidDecreto, *chiaveDecreto)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	case "avvia-trasporto":
		if *idAgentePG == "" {
			return fmt.Errorf("mode avvia-trasporto: -id-agente-pg obbligatorio")
		}
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "AvviaTrasporto", *idReperto, *idAgentePG)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	case "ricevi-laboratorio":
		if *idLaboratorio == "" {
			return fmt.Errorf("mode ricevi-laboratorio: -id-laboratorio obbligatorio")
		}
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "RiceviInLaboratorio", *idReperto, *idLaboratorio)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	case "completa-analisi":
		if *cidRelazione == "" || *chiaveRelazione == "" {
			return fmt.Errorf("mode completa-analisi: -cid-relazione e -chiave-relazione obbligatori")
		}
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "CompletaAnalisi", *idReperto, *cidRelazione, *chiaveRelazione)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	case "prepara-riconsegna":
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "PreparaRiconsegna", *idReperto)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	case "deposita-sede":
		if *cidVerbaleRiconsegna == "" || *chiaveVerbaleRiconsegna == "" {
			return fmt.Errorf("mode deposita-sede: -cid-verbale-riconsegna e -chiave-verbale-riconsegna obbligatori")
		}
		start := time.Now()
		txID, err := submitAndWait(ctx, contract, "DepositaInSede", *idReperto, *cidVerbaleRiconsegna, *chiaveVerbaleRiconsegna)
		if err != nil {
			return err
		}
		return writeJSON(workflowResult{
			Mode: m, IDReperto: *idReperto, Org: orgKey,
			TransactionID: txID, TempoFabricMs: time.Since(start).Milliseconds(),
			TempoTotaleMs: time.Since(startTotal).Milliseconds(),
		})

	default:
		return fmt.Errorf("-mode non valido: %q", m)
	}
}
