package contract

import "github.com/hyperledger/fabric-contract-api-go/contractapi"

// Contratto con tutte le funzioni che espongo su Fabric per la catena di custodia.
type SmartContract struct {
	contractapi.Contract
}
