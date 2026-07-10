// Funzioni principali sul reperto fisico (scheda di custodia, non i file su IPFS).
package contract

import (
	"encoding/json"
	"fmt"

	"coc/chaincode/internal/model"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Solo PG puo' creare un reperto. Metadati sensibili arrivano nel transient.
func (s *SmartContract) CreaReperto(ctx contractapi.TransactionContextInterface, idReperto string) error {
	if err := requireMSPID(ctx, mspPG); err != nil {
		return err
	}
	exists, err := s.RepertoExists(ctx, idReperto)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("reperto con id %s gia' presente", idReperto)
	}

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("errore lettura transient map: %w", err)
	}
	privateRaw, ok := transientMap["reperto_privato"]
	if !ok || len(privateRaw) == 0 {
		return fmt.Errorf("chiave transient 'reperto_privato' mancante")
	}

	var privateData model.RepertoPrivate
	if err := json.Unmarshal(privateRaw, &privateData); err != nil {
		return fmt.Errorf("errore parsing transient reperto_privato: %w", err)
	}
	if privateData.ID_Distretto == "" {
		return fmt.Errorf("idDistretto obbligatorio nel transient")
	}
	if privateData.ID_Caso == "" {
		return fmt.Errorf("idCaso obbligatorio nel transient reperto_privato")
	}

	privateBytes, err := json.Marshal(privateData)
	if err != nil {
		return fmt.Errorf("errore serializzazione reperto privato: %w", err)
	}
	if err := ctx.GetStub().PutPrivateData(repertoPrivateCollection, idReperto, privateBytes); err != nil {
		return fmt.Errorf("errore scrittura PDC %s: %w", repertoPrivateCollection, err)
	}

	publicData := model.RepertoPublic{
		ID_Reperto:     idReperto,
		ID_Caso:        privateData.ID_Caso,
		Stato:          model.StatoSequestrato,
		CustodeAttuale: privateData.ID_Distretto,
	}
	publicBytes, err := json.Marshal(publicData)
	if err != nil {
		return fmt.Errorf("errore serializzazione reperto pubblico: %w", err)
	}
	return ctx.GetStub().PutState(idReperto, publicBytes)
}

func (s *SmartContract) ReadReperto(ctx contractapi.TransactionContextInterface, idReperto string) (*model.Reperto, error) {
	p, err := loadRepertoPublic(ctx, idReperto)
	if err != nil {
		return nil, err
	}
	msp, err := getCallerMSPID(ctx)
	if err != nil {
		return nil, err
	}
	out := mergeReperto(p, nil)
	if msp == mspPG || msp == mspPM {
		privateBytes, err := ctx.GetStub().GetPrivateData(repertoPrivateCollection, idReperto)
		if err != nil {
			return out, nil
		}
		if privateBytes == nil {
			return out, nil
		}
		var privateData model.RepertoPrivate
		if err := json.Unmarshal(privateBytes, &privateData); err != nil {
			return nil, fmt.Errorf("errore deserializzazione reperto privato %s: %w", idReperto, err)
		}
		return mergeReperto(p, &privateData), nil
	}
	if msp == mspLAB {
		privLab, err := loadRepertoPrivatePMLab(ctx, idReperto)
		if err == nil && privLab != nil {
			return mergeReperto(p, privLab), nil
		}
	}
	return out, nil
}

func (s *SmartContract) RepertoExists(ctx contractapi.TransactionContextInterface, idReperto string) (bool, error) {
	b, err := ctx.GetStub().GetState(idReperto)
	if err != nil {
		return false, fmt.Errorf("errore controllo esistenza reperto %s: %w", idReperto, err)
	}
	return b != nil, nil
}
