package contract

import (
	"encoding/json"
	"fmt"

	"coc/chaincode/internal/model"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	mspPG  = "PGMSP"
	mspPM  = "PMMSP"
	mspLAB = "LABMSP"
)

// Collezione privata dove salvo i dati reperto tra PG e PM.
const repertoPrivateCollection = "collezione_PG_PM"

// Leggo quale organizzazione ha firmato la transazione.
func getCallerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	return ctx.GetClientIdentity().GetMSPID()
}

// Controllo che la transazione arrivi dall'organizzazione che mi aspetto.
func requireMSPID(ctx contractapi.TransactionContextInterface, want string) error {
	got, err := getCallerMSPID(ctx)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("accesso negato: richiesto MSP %q, chiamante %q", want, got)
	}
	return nil
}

// Accetto solo chiamate da PG, PM o LAB.
func requireCocMember(ctx contractapi.TransactionContextInterface) error {
	got, err := getCallerMSPID(ctx)
	if err != nil {
		return err
	}
	if got == mspPG || got == mspPM || got == mspLAB {
		return nil
	}
	return fmt.Errorf("accesso negato: MSP %q non e' un attore forense previsto (PG, PM, LAB)", got)
}

// Restituisce lo stato del reperto; se sul ledger e' vuoto usa SEQUESTRATO.
func statoEffettivoReperto(p *model.RepertoPublic) string {
	if p == nil || p.Stato == "" {
		return model.StatoSequestrato
	}
	return p.Stato
}

// Leggo la parte pubblica del reperto dal ledger.
func loadRepertoPublic(ctx contractapi.TransactionContextInterface, idReperto string) (*model.RepertoPublic, error) {
	b, err := ctx.GetStub().GetState(idReperto)
	if err != nil {
		return nil, fmt.Errorf("lettura state: %w", err)
	}
	if b == nil {
		return nil, fmt.Errorf("reperto con id %q non trovato", idReperto)
	}
	var p model.RepertoPublic
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("deserializzazione pubblico: %w", err)
	}
	return &p, nil
}

// Scrivo la parte pubblica del reperto sul ledger.
func saveRepertoPublic(ctx contractapi.TransactionContextInterface, p *model.RepertoPublic) error {
	if p.ID_Reperto == "" {
		return fmt.Errorf("idReperto mancante")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("serializzazione pubblico: %w", err)
	}
	return ctx.GetStub().PutState(p.ID_Reperto, b)
}

// Leggo i dati privati del reperto; se non ci sono blocco.
func loadRepertoPrivateMandatory(ctx contractapi.TransactionContextInterface, idReperto string) (*model.RepertoPrivate, error) {
	b, err := ctx.GetStub().GetPrivateData(repertoPrivateCollection, idReperto)
	if err != nil {
		return nil, fmt.Errorf("lettura PDC %s: %w", repertoPrivateCollection, err)
	}
	if b == nil {
		return nil, fmt.Errorf("dati privati non trovati per reperto %s", idReperto)
	}
	var pr model.RepertoPrivate
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, fmt.Errorf("deserializzazione privato: %w", err)
	}
	return &pr, nil
}

const repertoPrivatePMLABKeyPrefix = "reperto_privato_pm_lab/"

// Chiave che uso per la copia dei dati reperto visibili al laboratorio.
func repertoPrivatePMLABKey(idReperto string) string {
	return repertoPrivatePMLABKeyPrefix + idReperto
}

// Leggo la copia dei dati reperto che il laboratorio puo' vedere.
func loadRepertoPrivatePMLab(ctx contractapi.TransactionContextInterface, idReperto string) (*model.RepertoPrivate, error) {
	k := repertoPrivatePMLABKey(idReperto)
	b, err := ctx.GetStub().GetPrivateData(pdcPMLAB, k)
	if err != nil {
		if pdcIgnorabileInLettura(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lettura reperto privato PM_LAB: %w", err)
	}
	if b == nil {
		return nil, nil
	}
	var pr model.RepertoPrivate
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, fmt.Errorf("deserializzazione reperto privato PM_LAB: %w", err)
	}
	return &pr, nil
}

// Scrivo i dati reperto nella collezione condivisa con il laboratorio.
func putRepertoPrivateInPMLab(ctx contractapi.TransactionContextInterface, idReperto string, priv *model.RepertoPrivate) error {
	if priv == nil {
		return fmt.Errorf("dati privati reperto nil")
	}
	b, err := json.Marshal(priv)
	if err != nil {
		return fmt.Errorf("serializzazione reperto privato: %w", err)
	}
	k := repertoPrivatePMLABKey(idReperto)
	if err := ctx.GetStub().PutPrivateData(pdcPMLAB, k, b); err != nil {
		return fmt.Errorf("scrittura reperto privato in PM_LAB: %w", err)
	}
	return nil
}

// Copio il verbale di sequestro nella collezione che legge anche il laboratorio.
func copyVerbaleSequestroVersoPMLab(ctx contractapi.TransactionContextInterface, idCaso, idVerbale string) error {
	if idCaso == "" || idVerbale == "" {
		return fmt.Errorf("idCaso e id verbale sequestro obbligatori per la copia in PM_LAB")
	}
	pk := filePrivDataKey(idCaso, idVerbale)
	b, err := ctx.GetStub().GetPrivateData(pdcPGPM, pk)
	if err != nil {
		return fmt.Errorf("lettura verbale sequestro PG_PM: %w", err)
	}
	if b == nil {
		return fmt.Errorf("verbale di sequestro %q non trovato in collezione_PG_PM", idVerbale)
	}
	if err := ctx.GetStub().PutPrivateData(pdcPMLAB, pk, b); err != nil {
		return fmt.Errorf("scrittura copia sequestro in PM_LAB: %w", err)
	}
	return nil
}

// Unisco pubblico e privato in un solo oggetto da restituire.
func mergeReperto(pub *model.RepertoPublic, priv *model.RepertoPrivate) *model.Reperto {
	if pub == nil {
		return nil
	}
	out := &model.Reperto{
		ID_Caso:                 pub.ID_Caso,
		ID_Reperto:              pub.ID_Reperto,
		Tipo_Analisi:            pub.Tipo_Analisi,
		Stato:                   statoEffettivoReperto(pub),
		CustodeAttuale:          pub.CustodeAttuale,
		LaboratorioDestinazione: pub.LaboratorioDestinazione,
		IDVerbaleSequestro:      pub.IDVerbaleSequestro,
		IDDecretoAccertamento:   pub.IDDecretoAccertamento,
		IDRelazioneTecnica:      pub.IDRelazioneTecnica,
		IDVerbaleRiconsegna:     pub.IDVerbaleRiconsegna,
	}
	if out.CustodeAttuale == "" {
		out.CustodeAttuale = "PG"
	}
	if priv != nil {
		out.ID_Caso = priv.ID_Caso
		out.ID_Agente = priv.ID_Agente
		out.ID_Distretto = priv.ID_Distretto
		out.Data_Ora_Prelievo = priv.Data_Ora_Prelievo
		out.Descrizione_Bene = priv.Descrizione_Bene
	}
	return out
}
