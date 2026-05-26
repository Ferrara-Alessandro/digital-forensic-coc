// Avvio del chaincode sul peer Fabric.
package main

import (
	"fmt"

	"coc/chaincode/internal/contract"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&contract.SmartContract{})
	if err != nil {
		panic(fmt.Errorf("creazione chaincode: %w", err))
	}
	if err := chaincode.Start(); err != nil {
		panic(fmt.Errorf("avvio chaincode: %w", err))
	}
}
