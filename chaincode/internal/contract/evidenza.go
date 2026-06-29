// File digitali generici (foto, log, ecc.) sempre in collezione PG-PM.
package contract

import (
	"encoding/json"
	"fmt"

	"coc/chaincode/internal/model"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	compositeTipoEvidenza        = "EVI"
	compositeTipoEvidenzaReperto = "EVIREP"
)

// Chiave ledger per la parte pubblica dell'evidenza.
func evidenzaLedgerKey(stub shimChaincodeStub, idCaso, idEvidenza string) (string, error) {
	return stub.CreateCompositeKey(compositeTipoEvidenza, []string{idCaso, idEvidenza})
}

// Chiave per elencare evidenze di un reperto.
func evidenzaRepertoMarkerKey(stub shimChaincodeStub, idReperto, idEvidenza string) (string, error) {
	return stub.CreateCompositeKey(compositeTipoEvidenzaReperto, []string{idReperto, idEvidenza})
}

// Controllo se l'evidenza esiste gia'.
func evidenzaExists(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (bool, error) {
	k, err := evidenzaLedgerKey(ctx.GetStub(), idCaso, idEvidenza)
	if err != nil {
		return false, err
	}
	b, err := ctx.GetStub().GetState(k)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

// Leggo cid dall'evidenza pubblica.
func loadEvidenzaPublic(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (*model.EvidenzaPublic, error) {
	k, err := evidenzaLedgerKey(ctx.GetStub(), idCaso, idEvidenza)
	if err != nil {
		return nil, err
	}
	b, err := ctx.GetStub().GetState(k)
	if err != nil {
		return nil, fmt.Errorf("lettura evidenza pubblica: %w", err)
	}
	if b == nil {
		return nil, fmt.Errorf("evidenza %q per caso %q non trovata", idEvidenza, idCaso)
	}
	var pub model.EvidenzaPublic
	if err := json.Unmarshal(b, &pub); err != nil {
		return nil, fmt.Errorf("deserializzazione evidenza pubblica: %w", err)
	}
	return &pub, nil
}

// Leggo descrizione e chiave dalla collezione privata PG-PM.
func loadEvidenzaPrivate(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (*model.EvidenzaPrivate, error) {
	pk := filePrivDataKey(idCaso, idEvidenza)
	b, err := ctx.GetStub().GetPrivateData(pdcPGPM, pk)
	if err != nil {
		if pdcIgnorabileInLettura(err) {
			return nil, nil
		}
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var pr model.EvidenzaPrivate
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, fmt.Errorf("deserializzazione evidenza privata: %w", err)
	}
	if pr.IDCaso != idCaso {
		return nil, fmt.Errorf("incoerenza idCaso nel privato evidenza")
	}
	return &pr, nil
}

// Unisco pubblico e privato in un'evidenza completa.
func mergeEvidenza(pub *model.EvidenzaPublic, priv *model.EvidenzaPrivate) *model.Evidenza {
	if pub == nil {
		return nil
	}
	out := &model.Evidenza{
		IDEvidenza: pub.IDEvidenza,
		CID:        pub.CID,
	}
	if priv != nil {
		out.IDCaso = priv.IDCaso
		out.IDReperto = priv.IDReperto
		out.Descrizione = priv.Descrizione
		out.Classe = priv.Classe
		out.RegistratoIl = priv.RegistratoIl
		out.Autore = priv.Autore
		out.ChiaveCifrata = priv.ChiaveCifrata
	}
	return out
}

// Leggo evidenza con tutti i campi visibili a questo peer.
func loadEvidenzaMerged(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (*model.Evidenza, error) {
	pub, err := loadEvidenzaPublic(ctx, idCaso, idEvidenza)
	if err != nil {
		return nil, err
	}
	priv, err := loadEvidenzaPrivate(ctx, idCaso, idEvidenza)
	if err != nil {
		return nil, err
	}
	return mergeEvidenza(pub, priv), nil
}

// Scrivo evidenza su ledger e metadati privati.
func putEvidenza(ctx contractapi.TransactionContextInterface, pub *model.EvidenzaPublic, priv *model.EvidenzaPrivate, idRepertoLink string) error {
	if pub == nil || priv == nil {
		return fmt.Errorf("evidenza pubblica/privata obbligatoria")
	}
	if pub.IDEvidenza == "" || pub.CID == "" {
		return fmt.Errorf("idEvidenza e cid obbligatori sul pubblico")
	}
	if priv.IDCaso == "" {
		return fmt.Errorf("idCaso obbligatorio nel privato")
	}
	if priv.Descrizione == "" {
		return fmt.Errorf("descrizione obbligatoria nel metadato evidenza")
	}
	if err := requireChiaveCifrata(priv.ChiaveCifrata); err != nil {
		return err
	}
	k, err := evidenzaLedgerKey(ctx.GetStub(), priv.IDCaso, pub.IDEvidenza)
	if err != nil {
		return err
	}
	pubBytes, err := json.Marshal(pub)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(k, pubBytes); err != nil {
		return fmt.Errorf("PutState evidenza pubblica: %w", err)
	}
	privBytes, err := json.Marshal(priv)
	if err != nil {
		return err
	}
	pk := filePrivDataKey(priv.IDCaso, pub.IDEvidenza)
	if err := ctx.GetStub().PutPrivateData(pdcPGPM, pk, privBytes); err != nil {
		return fmt.Errorf("PutPrivateData evidenza: %w", err)
	}
	if idRepertoLink != "" {
		rk, err := evidenzaRepertoMarkerKey(ctx.GetStub(), idRepertoLink, pub.IDEvidenza)
		if err != nil {
			return err
		}
		marker, err := json.Marshal(map[string]string{"idCaso": priv.IDCaso})
		if err != nil {
			return err
		}
		if err := ctx.GetStub().PutState(rk, marker); err != nil {
			return fmt.Errorf("indice EVIREP: %w", err)
		}
	}
	return nil
}

// Registro evidenza con cid e chiave gia' noti.
func (s *SmartContract) registraEvidenzaConCID(ctx contractapi.TransactionContextInterface, idEvidenza, idCaso, idReperto, classe, cid, chiaveCifrata, descrizione string) error {
	if err := requireMSPID(ctx, mspPG); err != nil {
		return err
	}
	if idEvidenza == "" || idCaso == "" {
		return fmt.Errorf("idEvidenza e idCaso obbligatori")
	}
	exists, err := evidenzaExists(ctx, idCaso, idEvidenza)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("evidenza %q gia' registrata per il caso %q", idEvidenza, idCaso)
	}
	if cid == "" {
		return fmt.Errorf("cid obbligatorio")
	}
	if err := requireChiaveCifrata(chiaveCifrata); err != nil {
		return err
	}
	if descrizione == "" {
		return fmt.Errorf("descrizione obbligatoria")
	}
	if idReperto != "" {
		ok, err := s.RepertoExists(ctx, idReperto)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("reperto %q non trovato", idReperto)
		}
	}
	aut, err := autoreRegistrante(ctx)
	if err != nil {
		return err
	}
	pub := &model.EvidenzaPublic{IDEvidenza: idEvidenza, CID: cid}
	priv := &model.EvidenzaPrivate{
		IDCaso:        idCaso,
		IDReperto:     idReperto,
		Classe:        classe,
		Descrizione:   descrizione,
		RegistratoIl:  registratoIlUTC(ctx),
		Autore:        aut,
		ChiaveCifrata: chiaveCifrata,
	}
	return putEvidenza(ctx, pub, priv, idReperto)
}

// Registro leggendo dal transient del programma di upload.
func (s *SmartContract) RegistraEvidenzaConTransient(ctx contractapi.TransactionContextInterface, idEvidenza, idCaso, idReperto, descrizione, classe string) error {
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("errore lettura transient map: %w", err)
	}
	raw, ok := transientMap["evidenza"]
	if !ok || len(raw) == 0 {
		return fmt.Errorf("chiave transient 'evidenza' mancante")
	}
	payload, err := parseTransientFile(raw, "evidenza")
	if err != nil {
		return err
	}
	return s.registraEvidenzaConCID(ctx, idEvidenza, idCaso, idReperto, classe, payload.Cid, payload.ChiaveCifrata, descrizione)
}

// Leggo evidenza unendo le parti consentite.
func (s *SmartContract) LeggiEvidenza(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (*model.Evidenza, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idCaso == "" || idEvidenza == "" {
		return nil, fmt.Errorf("idCaso e idEvidenza obbligatori")
	}
	e, err := loadEvidenzaMerged(ctx, idCaso, idEvidenza)
	if err != nil {
		return nil, err
	}
	if e.Descrizione == "" {
		return nil, fmt.Errorf("metadati evidenza non disponibili per questo MSP/peer (PDC non leggibile)")
	}
	return e, nil
}

// Elenco evidenze di un caso.
func (s *SmartContract) ListaEvidenzeCaso(ctx contractapi.TransactionContextInterface, idCaso string) ([]*model.Evidenza, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idCaso == "" {
		return nil, fmt.Errorf("idCaso obbligatorio")
	}
	it, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeTipoEvidenza, []string{idCaso})
	if err != nil {
		return nil, fmt.Errorf("iterazione evidenze caso: %w", err)
	}
	defer func() { _ = it.Close() }()

	out := make([]*model.Evidenza, 0)
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return nil, err
		}
		_, attrs, err := ctx.GetStub().SplitCompositeKey(kv.GetKey())
		if err != nil || len(attrs) < 2 {
			return nil, fmt.Errorf("chiave EVI inattesa")
		}
		idCasoKey, idEvi := attrs[0], attrs[1]
		var pub model.EvidenzaPublic
		if err := json.Unmarshal(kv.GetValue(), &pub); err != nil {
			return nil, fmt.Errorf("parse evidenza pubblica: %w", err)
		}
		priv, err := loadEvidenzaPrivate(ctx, idCasoKey, idEvi)
		if err != nil {
			return nil, err
		}
		out = append(out, mergeEvidenza(&pub, priv))
	}
	return out, nil
}

// Elenco evidenze collegate a un reperto.
func (s *SmartContract) ListaEvidenzeReperto(ctx contractapi.TransactionContextInterface, idReperto string) ([]*model.Evidenza, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idReperto == "" {
		return nil, fmt.Errorf("idReperto obbligatorio")
	}
	it, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeTipoEvidenzaReperto, []string{idReperto})
	if err != nil {
		return nil, fmt.Errorf("iterazione EVIREP: %w", err)
	}
	defer func() { _ = it.Close() }()

	out := make([]*model.Evidenza, 0)
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return nil, err
		}
		var marker struct {
			IDCaso string `json:"idCaso"`
		}
		if err := json.Unmarshal(kv.GetValue(), &marker); err != nil {
			return nil, fmt.Errorf("parse marker EVIREP: %w", err)
		}
		_, attrs, err := ctx.GetStub().SplitCompositeKey(kv.GetKey())
		if err != nil || len(attrs) < 2 {
			return nil, fmt.Errorf("chiave EVIREP inattesa")
		}
		idEvi := attrs[1]
		e, err := loadEvidenzaMerged(ctx, marker.IDCaso, idEvi)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// Storico modifiche alla parte pubblica dell'evidenza.
func (s *SmartContract) OttieniStoriaEvidenza(ctx contractapi.TransactionContextInterface, idCaso, idEvidenza string) (string, error) {
	if err := requireCocMember(ctx); err != nil {
		return "", err
	}
	k, err := evidenzaLedgerKey(ctx.GetStub(), idCaso, idEvidenza)
	if err != nil {
		return "", err
	}
	hist, err := ctx.GetStub().GetHistoryForKey(k)
	if err != nil {
		return "", fmt.Errorf("GetHistoryForKey evidenza: %w", err)
	}
	defer func() { _ = hist.Close() }()

	type voce struct {
		TxId      string          `json:"txId"`
		Timestamp string          `json:"timestamp"`
		IsDelete  bool            `json:"isDelete,omitempty"`
		Evidenza  json.RawMessage `json:"evidenza"`
	}
	var out []voce
	for hist.HasNext() {
		km, err := hist.Next()
		if err != nil {
			return "", err
		}
		entry := voce{TxId: km.GetTxId(), IsDelete: km.GetIsDelete()}
		if ts := km.GetTimestamp(); ts != nil {
			entry.Timestamp = formatProtoTime(ts)
		}
		if km.GetIsDelete() {
			entry.Evidenza = nil
		} else if len(km.GetValue()) == 0 {
			entry.Evidenza = json.RawMessage("null")
		} else {
			v := make([]byte, len(km.GetValue()))
			copy(v, km.GetValue())
			entry.Evidenza = v
		}
		out = append(out, entry)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
