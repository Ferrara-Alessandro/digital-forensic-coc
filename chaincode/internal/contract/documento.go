// Gestione degli atti su IPFS collegati a caso o reperto.
package contract

import (
	"encoding/json"
	"fmt"

	"coc/chaincode/internal/model"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	compositeTipoDocumento        = "DOC"
	compositeTipoDocumentoReperto = "DOCREP"
)

// Chiave sul ledger per la parte pubblica del documento (c'e' il cid IPFS).
func documentoLedgerKey(stub shimChaincodeStub, idCaso, idDocumento string) (string, error) {
	return stub.CreateCompositeKey(compositeTipoDocumento, []string{idCaso, idDocumento})
}

// Chiave per trovare tutti i documenti legati a un reperto.
func documentoRepertoMarkerKey(stub shimChaincodeStub, idReperto, idDocumento string) (string, error) {
	return stub.CreateCompositeKey(compositeTipoDocumentoReperto, []string{idReperto, idDocumento})
}

// Scelgo la collezione privata in base al tipo di atto.
func pdcCollectionForTipoDocumento(tipo string) (string, error) {
	switch tipo {
	case model.TipoVerbaleSopralluogo, model.TipoVerbaleSequestro, model.TipoVerbaleRiconsegna:
		return pdcPGPM, nil
	case model.TipoDecretoAccertamento, model.TipoRelazioneTecnica:
		return pdcPMLAB, nil
	default:
		if err := model.ValidateTipoDocumento(tipo); err != nil {
			return "", err
		}
		return "", fmt.Errorf("tipoDocumento %q: collection PDC non definita", tipo)
	}
}

// Controllo se ho gia' registrato questo documento.
func documentoExists(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (bool, error) {
	k, err := documentoLedgerKey(ctx.GetStub(), idCaso, idDocumento)
	if err != nil {
		return false, err
	}
	b, err := ctx.GetStub().GetState(k)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

// Leggo cid e id dal registro pubblico.
func loadDocumentoPublic(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (*model.DocumentoPublic, error) {
	k, err := documentoLedgerKey(ctx.GetStub(), idCaso, idDocumento)
	if err != nil {
		return nil, err
	}
	b, err := ctx.GetStub().GetState(k)
	if err != nil {
		return nil, fmt.Errorf("lettura documento pubblico: %w", err)
	}
	if b == nil {
		return nil, fmt.Errorf("documento %q per caso %q non trovato", idDocumento, idCaso)
	}
	var pub model.DocumentoPublic
	if err := json.Unmarshal(b, &pub); err != nil {
		return nil, fmt.Errorf("deserializzazione documento pubblico: %w", err)
	}
	return &pub, nil
}

// Leggo descrizione, chiave AES e altro dalla collezione privata.
func loadDocumentoPrivate(ctx contractapi.TransactionContextInterface, collection, idCaso, idDocumento string) (*model.DocumentoPrivate, error) {
	pk := filePrivDataKey(idCaso, idDocumento)
	b, err := ctx.GetStub().GetPrivateData(collection, pk)
	if err != nil {
		if pdcIgnorabileInLettura(err) {
			return nil, nil
		}
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var pr model.DocumentoPrivate
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, fmt.Errorf("deserializzazione documento privato: %w", err)
	}
	if pr.IDCaso != idCaso {
		return nil, fmt.Errorf("incoerenza idCaso nel privato documento")
	}
	return &pr, nil
}

// Provo a leggere i metadati prima da una collezione e poi dall'altra.
func tryLoadDocumentoPrivate(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (*model.DocumentoPrivate, error) {
	pr, err := loadDocumentoPrivate(ctx, pdcPGPM, idCaso, idDocumento)
	if err != nil {
		return nil, err
	}
	if pr != nil {
		return pr, nil
	}
	return loadDocumentoPrivate(ctx, pdcPMLAB, idCaso, idDocumento)
}

// Unisco pubblico e privato per rispondere alle query.
func mergeDocumento(pub *model.DocumentoPublic, priv *model.DocumentoPrivate) *model.Documento {
	if pub == nil {
		return nil
	}
	out := &model.Documento{
		IDDocumento: pub.IDDocumento,
		CID:         pub.CID,
	}
	if priv != nil {
		out.IDCaso = priv.IDCaso
		out.IDReperto = priv.IDReperto
		out.TipoDocumento = priv.TipoDocumento
		out.Descrizione = priv.Descrizione
		out.RegistratoIl = priv.RegistratoIl
		out.Autore = priv.Autore
		out.RiferimentoEnte = priv.RiferimentoEnte
		out.ChiaveCifrata = priv.ChiaveCifrata
	}
	return out
}

// Leggo un documento con tutti i campi che il peer puo' vedere.
func loadDocumentoMerged(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (*model.Documento, error) {
	pub, err := loadDocumentoPublic(ctx, idCaso, idDocumento)
	if err != nil {
		return nil, err
	}
	priv, err := tryLoadDocumentoPrivate(ctx, idCaso, idDocumento)
	if err != nil {
		return nil, err
	}
	return mergeDocumento(pub, priv), nil
}

// Scrivo documento sul ledger, metadati privati e eventuale collegamento al reperto.
func putDocumento(ctx contractapi.TransactionContextInterface, pub *model.DocumentoPublic, priv *model.DocumentoPrivate, idRepertoLink string) error {
	if pub == nil || priv == nil {
		return fmt.Errorf("documento pubblico/privato obbligatori")
	}
	if pub.IDDocumento == "" || pub.CID == "" {
		return fmt.Errorf("idDocumento e cid obbligatori sul pubblico")
	}
	if priv.IDCaso == "" {
		return fmt.Errorf("idCaso obbligatorio nel privato")
	}
	if err := model.ValidateTipoDocumento(priv.TipoDocumento); err != nil {
		return err
	}
	if priv.Descrizione == "" {
		return fmt.Errorf("descrizione obbligatoria nel metadato documento")
	}
	if err := requireChiaveCifrata(priv.ChiaveCifrata); err != nil {
		return err
	}
	col, err := pdcCollectionForTipoDocumento(priv.TipoDocumento)
	if err != nil {
		return err
	}
	k, err := documentoLedgerKey(ctx.GetStub(), priv.IDCaso, pub.IDDocumento)
	if err != nil {
		return err
	}
	pubBytes, err := json.Marshal(pub)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(k, pubBytes); err != nil {
		return fmt.Errorf("PutState documento pubblico: %w", err)
	}
	privBytes, err := json.Marshal(priv)
	if err != nil {
		return err
	}
	pk := filePrivDataKey(priv.IDCaso, pub.IDDocumento)
	if err := ctx.GetStub().PutPrivateData(col, pk, privBytes); err != nil {
		return fmt.Errorf("PutPrivateData documento %s: %w", col, err)
	}
	if idRepertoLink != "" {
		rk, err := documentoRepertoMarkerKey(ctx.GetStub(), idRepertoLink, pub.IDDocumento)
		if err != nil {
			return err
		}
		marker, err := json.Marshal(map[string]string{"idCaso": priv.IDCaso})
		if err != nil {
			return err
		}
		if err := ctx.GetStub().PutState(rk, marker); err != nil {
			return fmt.Errorf("indice DOCREP: %w", err)
		}
	}
	return nil
}

// Controllo che sia l'organizzazione giusta per quel tipo di atto.
func requireMSPForTipoDocumento(ctx contractapi.TransactionContextInterface, tipo string) error {
	switch tipo {
	case model.TipoVerbaleSopralluogo, model.TipoVerbaleSequestro, model.TipoVerbaleRiconsegna:
		return requireMSPID(ctx, mspPG)
	case model.TipoDecretoAccertamento:
		return requireMSPID(ctx, mspPM)
	case model.TipoRelazioneTecnica:
		return requireMSPID(ctx, mspLAB)
	default:
		return model.ValidateTipoDocumento(tipo)
	}
}

// Verifico regole su tipo atto e se serve il reperto collegato.
func validateDocumentoShape(tipo, idReperto string) error {
	switch tipo {
	case model.TipoVerbaleSopralluogo:
		if idReperto != "" {
			return fmt.Errorf("VERBALE_SOPRALLUOGO non puo' avere idReperto: e' riferito al caso")
		}
	case model.TipoDecretoAccertamento, model.TipoRelazioneTecnica, model.TipoVerbaleRiconsegna:
		if idReperto == "" {
			return fmt.Errorf("tipo %q richiede idReperto", tipo)
		}
	}
	return nil
}

// Sulla scheda reperto salvo l'id del documento appena registrato.
func aggiornaPuntatoreDocumentoSuReperto(ctx contractapi.TransactionContextInterface, idReperto, idDocumento, tipo string) error {
	if idReperto == "" {
		return nil
	}
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return err
	}
	switch tipo {
	case model.TipoVerbaleSequestro:
		if p.IDVerbaleSequestro == "" {
			p.IDVerbaleSequestro = idDocumento
		}
	default:
		return nil
	}
	return saveRepertoPublic(ctx, p)
}

// Registro un documento quando ho gia' cid e chiave (dal client o da un passo del flusso).
func (s *SmartContract) registraDocumentoConCID(ctx contractapi.TransactionContextInterface, idDocumento, idCaso, tipoDocumento, idReperto, cid, chiaveCifrata, descrizione, riferimentoEnte string) error {
	if idDocumento == "" || idCaso == "" {
		return fmt.Errorf("idDocumento e idCaso obbligatori")
	}
	exists, err := documentoExists(ctx, idCaso, idDocumento)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("documento %q gia' registrato per il caso %q", idDocumento, idCaso)
	}
	if err := requireMSPForTipoDocumento(ctx, tipoDocumento); err != nil {
		return err
	}
	if err := validateDocumentoShape(tipoDocumento, idReperto); err != nil {
		return err
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
	pub := &model.DocumentoPublic{IDDocumento: idDocumento, CID: cid}
	priv := &model.DocumentoPrivate{
		IDCaso:          idCaso,
		IDReperto:       idReperto,
		TipoDocumento:   tipoDocumento,
		Descrizione:     descrizione,
		RegistratoIl:    registratoIlUTC(ctx),
		Autore:          aut,
		RiferimentoEnte: riferimentoEnte,
		ChiaveCifrata:   chiaveCifrata,
	}
	link := idReperto
	if err := putDocumento(ctx, pub, priv, link); err != nil {
		return err
	}
	return aggiornaPuntatoreDocumentoSuReperto(ctx, idReperto, idDocumento, tipoDocumento)
}

// Registro leggendo cid e chiave dal transient inviato dal programma di upload.
func (s *SmartContract) RegistraDocumentoConTransient(ctx contractapi.TransactionContextInterface, idDocumento, idCaso, tipoDocumento, idReperto, descrizione, riferimentoEnte string) error {
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("errore lettura transient map: %w", err)
	}
	raw, ok := transientMap["documento"]
	if !ok || len(raw) == 0 {
		return fmt.Errorf("chiave transient 'documento' mancante")
	}
	payload, err := parseTransientFile(raw, "documento")
	if err != nil {
		return err
	}
	return s.registraDocumentoConCID(ctx, idDocumento, idCaso, tipoDocumento, idReperto, payload.Cid, payload.ChiaveCifrata, descrizione, riferimentoEnte)
}

// Leggo un documento unendo le parti che questo peer puo' vedere.
func (s *SmartContract) LeggiDocumento(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (*model.Documento, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idCaso == "" || idDocumento == "" {
		return nil, fmt.Errorf("idCaso e idDocumento obbligatori")
	}
	d, err := loadDocumentoMerged(ctx, idCaso, idDocumento)
	if err != nil {
		return nil, err
	}
	if d.TipoDocumento == "" {
		return nil, fmt.Errorf("metadati documento non disponibili per questo MSP/peer (PDC non leggibile)")
	}
	return d, nil
}

// Restituisco tutti i documenti registrati per un id caso.
func (s *SmartContract) ListaDocumentiCaso(ctx contractapi.TransactionContextInterface, idCaso string) ([]*model.Documento, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idCaso == "" {
		return nil, fmt.Errorf("idCaso obbligatorio")
	}
	it, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeTipoDocumento, []string{idCaso})
	if err != nil {
		return nil, fmt.Errorf("iterazione documenti caso: %w", err)
	}
	defer func() { _ = it.Close() }()

	out := make([]*model.Documento, 0)
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return nil, err
		}
		_, attrs, err := ctx.GetStub().SplitCompositeKey(kv.GetKey())
		if err != nil || len(attrs) < 2 {
			return nil, fmt.Errorf("chiave DOC inattesa")
		}
		idCasoKey, idDoc := attrs[0], attrs[1]
		var pub model.DocumentoPublic
		if err := json.Unmarshal(kv.GetValue(), &pub); err != nil {
			return nil, fmt.Errorf("parse documento pubblico: %w", err)
		}
		priv, err := tryLoadDocumentoPrivate(ctx, idCasoKey, idDoc)
		if err != nil {
			return nil, err
		}
		out = append(out, mergeDocumento(&pub, priv))
	}
	return out, nil
}

// Restituisco i documenti collegati a un reperto.
func (s *SmartContract) ListaDocumentiReperto(ctx contractapi.TransactionContextInterface, idReperto string) ([]*model.Documento, error) {
	if err := requireCocMember(ctx); err != nil {
		return nil, err
	}
	if idReperto == "" {
		return nil, fmt.Errorf("idReperto obbligatorio")
	}
	it, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeTipoDocumentoReperto, []string{idReperto})
	if err != nil {
		return nil, fmt.Errorf("iterazione DOCREP: %w", err)
	}
	defer func() { _ = it.Close() }()

	out := make([]*model.Documento, 0)
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return nil, err
		}
		var marker struct {
			IDCaso string `json:"idCaso"`
		}
		if err := json.Unmarshal(kv.GetValue(), &marker); err != nil {
			return nil, fmt.Errorf("parse marker DOCREP: %w", err)
		}
		_, attrs, err := ctx.GetStub().SplitCompositeKey(kv.GetKey())
		if err != nil || len(attrs) < 2 {
			return nil, fmt.Errorf("chiave DOCREP inattesa")
		}
		idDoc := attrs[1]
		d, err := loadDocumentoMerged(ctx, marker.IDCaso, idDoc)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// Storico delle modifiche alla parte pubblica del documento.
func (s *SmartContract) OttieniStoriaDocumento(ctx contractapi.TransactionContextInterface, idCaso, idDocumento string) (string, error) {
	if err := requireCocMember(ctx); err != nil {
		return "", err
	}
	k, err := documentoLedgerKey(ctx.GetStub(), idCaso, idDocumento)
	if err != nil {
		return "", err
	}
	hist, err := ctx.GetStub().GetHistoryForKey(k)
	if err != nil {
		return "", fmt.Errorf("GetHistoryForKey documento: %w", err)
	}
	defer func() { _ = hist.Close() }()

	type voce struct {
		TxId       string          `json:"txId"`
		Timestamp  string          `json:"timestamp"`
		IsDelete   bool            `json:"isDelete,omitempty"`
		Documento  json.RawMessage `json:"documento"`
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
			entry.Documento = nil
		} else if len(km.GetValue()) == 0 {
			entry.Documento = json.RawMessage("null")
		} else {
			v := make([]byte, len(km.GetValue()))
			copy(v, km.GetValue())
			entry.Documento = v
		}
		out = append(out, entry)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Uso questa funzione interna quando un passo del flusso deve allegare un atto.
func registraDocumentoAllegato(ctx contractapi.TransactionContextInterface, idCaso, idReperto, tipo, cid, chiaveCifrata, riferimentoEnte, descrizione string) (string, error) {
	if idCaso == "" {
		return "", fmt.Errorf("idCaso obbligatorio per documento allegato")
	}
	idDoc := ctx.GetStub().GetTxID()
	if idDoc == "" {
		return "", fmt.Errorf("id scrittura vuoto")
	}
	aut, err := autoreRegistrante(ctx)
	if err != nil {
		return "", err
	}
	pub := &model.DocumentoPublic{IDDocumento: idDoc, CID: cid}
	priv := &model.DocumentoPrivate{
		IDCaso:          idCaso,
		IDReperto:       idReperto,
		TipoDocumento:   tipo,
		Descrizione:     descrizione,
		RegistratoIl:    registratoIlUTC(ctx),
		Autore:          aut,
		RiferimentoEnte: riferimentoEnte,
		ChiaveCifrata:   chiaveCifrata,
	}
	if err := putDocumento(ctx, pub, priv, idReperto); err != nil {
		return "", err
	}
	if err := aggiornaPuntatoreDocumentoSuReperto(ctx, idReperto, idDoc, tipo); err != nil {
		return "", err
	}
	return idDoc, nil
}
