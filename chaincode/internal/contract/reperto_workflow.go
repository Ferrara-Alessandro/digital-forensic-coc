// Passaggi del ciclo di vita del reperto (trasporto, laboratorio, riconsegna).
package contract

import (
	"encoding/json"
	"fmt"

	"coc/chaincode/internal/model"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Richiesta analisi PM: decreto e dati per il laboratorio.
func (s *SmartContract) RichiediAnalisi(ctx contractapi.TransactionContextInterface, idReperto, idLab, tipoAnalisi, cidDecreto, chiaveDecreto string) error {
	if err := requireMSPID(ctx, mspPM); err != nil {
		return err
	}
	if idLab == "" || cidDecreto == "" || tipoAnalisi == "" {
		return fmt.Errorf("idLab, tipoAnalisi e cidDecreto obbligatori")
	}
	if err := requireChiaveCifrata(chiaveDecreto); err != nil {
		return fmt.Errorf("chiave decreto: %w", err)
	}

	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	if statoEffettivoReperto(p) != model.StatoSequestrato {
		return fmt.Errorf("stato attuale %q: atteso %q", statoEffettivoReperto(p), model.StatoSequestrato)
	}
	priv, err := loadRepertoPrivateMandatory(ctx, idReperto)
	if err != nil {
		return err
	}
	idDoc, err := registraDocumentoAllegato(ctx, priv.ID_Caso, idReperto, model.TipoDecretoAccertamento, cidDecreto, chiaveDecreto,
		idLab, "Decreto di accertamento e incarico di analisi tecnica presso laboratorio")
	if err != nil {
		return err
	}
	if err := putRepertoPrivateInPMLab(ctx, idReperto, priv); err != nil {
		return err
	}
	if err := copyVerbaleSequestroVersoPMLab(ctx, priv.ID_Caso, p.IDVerbaleSequestro); err != nil {
		return err
	}
	p.IDDecretoAccertamento = idDoc
	p.ID_Caso = priv.ID_Caso
	p.Tipo_Analisi = tipoAnalisi
	p.LaboratorioDestinazione = idLab
	p.Stato = model.StatoAttesaTrasporto
	return saveRepertoPublic(ctx, p)
}

// La PG segna che il reperto e' in viaggio.
func (s *SmartContract) AvviaTrasporto(ctx contractapi.TransactionContextInterface, idReperto, idAgentePG string) error {
	if err := requireMSPID(ctx, mspPG); err != nil {
		return err
	}
	if idAgentePG == "" {
		return fmt.Errorf("idAgentePG obbligatorio")
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	cur := statoEffettivoReperto(p)
	if cur != model.StatoAttesaTrasporto && cur != model.StatoAttesaRitiro {
		return fmt.Errorf("stato attuale %q: atteso %q o %q", cur, model.StatoAttesaTrasporto, model.StatoAttesaRitiro)
	}
	p.Stato = model.StatoInTransito
	p.CustodeAttuale = idAgentePG
	return saveRepertoPublic(ctx, p)
}

// Il laboratorio conferma che il reperto e' arrivato.
func (s *SmartContract) RiceviInLaboratorio(ctx contractapi.TransactionContextInterface, idReperto, idLaboratorio string) error {
	if err := requireMSPID(ctx, mspLAB); err != nil {
		return err
	}
	if idLaboratorio == "" {
		return fmt.Errorf("idLaboratorio obbligatorio")
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	if statoEffettivoReperto(p) != model.StatoInTransito {
		return fmt.Errorf("stato attuale %q: atteso %q", statoEffettivoReperto(p), model.StatoInTransito)
	}
	if p.ID_Caso == "" {
		return fmt.Errorf("idCaso mancante sul reperto pubblico")
	}
	if p.IDDecretoAccertamento == "" {
		return fmt.Errorf("decreto di accertamento non collegato al reperto")
	}
	if p.LaboratorioDestinazione == "" {
		return fmt.Errorf("laboratorio destinazione assente: eseguire prima RichiediAnalisi (PM)")
	}
	if p.LaboratorioDestinazione != idLaboratorio {
		return fmt.Errorf("idLaboratorio %q non coincide con la destinazione (%q)", idLaboratorio, p.LaboratorioDestinazione)
	}
	p.Stato = model.StatoInAnalisi
	p.CustodeAttuale = idLaboratorio
	return saveRepertoPublic(ctx, p)
}

// Il laboratorio carica la relazione e segna attesa ritiro.
func (s *SmartContract) CompletaAnalisi(ctx contractapi.TransactionContextInterface, idReperto, cidRelazione, chiaveRelazione string) error {
	if err := requireMSPID(ctx, mspLAB); err != nil {
		return err
	}
	if cidRelazione == "" {
		return fmt.Errorf("cidRelazione obbligatorio")
	}
	if err := requireChiaveCifrata(chiaveRelazione); err != nil {
		return fmt.Errorf("chiave relazione: %w", err)
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	if statoEffettivoReperto(p) != model.StatoInAnalisi {
		return fmt.Errorf("stato attuale %q: atteso %q", statoEffettivoReperto(p), model.StatoInAnalisi)
	}
	labRef := p.CustodeAttuale
	if labRef == "" {
		return fmt.Errorf("custode laboratorio assente")
	}
	if p.ID_Caso == "" {
		return fmt.Errorf("idCaso mancante sul reperto pubblico")
	}
	idDoc, err := registraDocumentoAllegato(ctx, p.ID_Caso, idReperto, model.TipoRelazioneTecnica, cidRelazione, chiaveRelazione,
		labRef, "Relazione tecnica di laboratorio sulle analisi richieste")
	if err != nil {
		return err
	}
	p.IDRelazioneTecnica = idDoc
	p.Stato = model.StatoAttesaRitiro
	return saveRepertoPublic(ctx, p)
}

// Il laboratorio conferma che il reperto e' pronto per essere ritirato.
func (s *SmartContract) PreparaRiconsegna(ctx contractapi.TransactionContextInterface, idReperto string) error {
	if err := requireMSPID(ctx, mspLAB); err != nil {
		return err
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	if statoEffettivoReperto(p) != model.StatoAttesaRitiro {
		return fmt.Errorf("stato attuale %q: atteso %q", statoEffettivoReperto(p), model.StatoAttesaRitiro)
	}
	return nil
}

// La PG registra il verbale di riconsegna e chiude il rientro in sede.
func (s *SmartContract) DepositaInSede(ctx contractapi.TransactionContextInterface, idReperto, cidVerbaleRiconsegna, chiaveVerbaleRiconsegna string) error {
	if err := requireMSPID(ctx, mspPG); err != nil {
		return err
	}
	if cidVerbaleRiconsegna == "" {
		return fmt.Errorf("cidVerbaleRiconsegna obbligatorio")
	}
	if err := requireChiaveCifrata(chiaveVerbaleRiconsegna); err != nil {
		return fmt.Errorf("chiave verbale riconsegna: %w", err)
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	if statoEffettivoReperto(p) != model.StatoInTransito {
		return fmt.Errorf("stato attuale %q: atteso %q", statoEffettivoReperto(p), model.StatoInTransito)
	}
	priv, err := loadRepertoPrivateMandatory(ctx, idReperto)
	if err != nil {
		return err
	}
	if priv.ID_Distretto == "" {
		return fmt.Errorf("idDistretto mancante nella PDC")
	}
	idCaso := p.ID_Caso
	if idCaso == "" {
		idCaso = priv.ID_Caso
	}
	if idCaso == "" {
		return fmt.Errorf("idCaso assente")
	}
	idDoc, err := registraDocumentoAllegato(ctx, idCaso, idReperto, model.TipoVerbaleRiconsegna, cidVerbaleRiconsegna, chiaveVerbaleRiconsegna,
		priv.ID_Distretto, "Verbale di riconsegna del reperto in sede")
	if err != nil {
		return err
	}
	p.IDVerbaleRiconsegna = idDoc
	p.Stato = model.StatoSequestrato
	p.CustodeAttuale = priv.ID_Distretto
	return saveRepertoPublic(ctx, p)
}

type voceStoriaReperto struct {
	TxId      string          `json:"txId"`
	Timestamp string          `json:"timestamp"`
	IsDelete  bool            `json:"isDelete,omitempty"`
	Reperto   json.RawMessage `json:"reperto"`
}

func (s *SmartContract) OttieniStoriaReperto(ctx contractapi.TransactionContextInterface, idReperto string) (string, error) {
	if err := requireCocMember(ctx); err != nil {
		return "", err
	}
	hist, err := ctx.GetStub().GetHistoryForKey(idReperto)
	if err != nil {
		return "", fmt.Errorf("GetHistoryForKey: %w", err)
	}
	defer func() { _ = hist.Close() }()

	var out []voceStoriaReperto
	for hist.HasNext() {
		km, err := hist.Next()
		if err != nil {
			return "", err
		}
		entry := voceStoriaReperto{TxId: km.GetTxId(), IsDelete: km.GetIsDelete()}
		if ts := km.GetTimestamp(); ts != nil {
			entry.Timestamp = formatProtoTime(ts)
		}
		if km.GetIsDelete() {
			entry.Reperto = nil
		} else if len(km.GetValue()) == 0 {
			entry.Reperto = json.RawMessage("null")
		} else {
			v := make([]byte, len(km.GetValue()))
			copy(v, km.GetValue())
			entry.Reperto = v
		}
		out = append(out, entry)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func formatProtoTime(t *timestamppb.Timestamp) string {
	if t == nil {
		return ""
	}
	if err := t.CheckValid(); err != nil {
		return ""
	}
	return t.AsTime().UTC().Format("2006-01-02T15:04:05.000Z07:00")
}
