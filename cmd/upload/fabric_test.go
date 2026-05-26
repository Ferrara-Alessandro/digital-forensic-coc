// Test di supporto sulla parte Fabric (senza rete reale).
package main

import (
	"encoding/json"
	"testing"
)

func TestBuildTransientReperto(t *testing.T) {
	transient, err := BuildTransientReperto(RepertoPrivatoInput{
		IDCaso:          "CASO-1",
		IDAgente:        "AG-1",
		IDDistretto:     "DIST-1",
		DataOraPrelievo: "2026-01-01T00:00:00Z",
		DescrizioneBene: "bene test",
	})
	if err != nil {
		t.Fatalf("BuildTransientReperto() err = %v", err)
	}
	if len(transient["reperto_privato"]) == 0 {
		t.Fatal("reperto_privato mancante")
	}
	if _, ok := transient["verbale_sequestro"]; ok {
		t.Fatal("non deve includere verbale_sequestro")
	}
	if _, ok := transient["evidenza"]; ok {
		t.Fatal("non deve includere evidenza")
	}
}

func TestBuildTransientDocumento(t *testing.T) {
	key := make([]byte, aesKeySize)
	m, err := BuildTransientDocumento("bafytestcid00000000000000000000000000000000000000000000", key)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(m["documento"], &payload); err != nil {
		t.Fatal(err)
	}
	if payload["cid"] == "" || payload["chiaveCifrata"] == "" {
		t.Fatalf("payload incompleto: %+v", payload)
	}
}

func TestBuildTransientEvidenza(t *testing.T) {
	key := make([]byte, aesKeySize)
	m, err := BuildTransientEvidenza("bafytestcid00000000000000000000000000000000000000000000", key)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(m["evidenza"], &payload); err != nil {
		t.Fatal(err)
	}
	if payload["cid"] == "" || payload["chiaveCifrata"] == "" {
		t.Fatalf("payload incompleto: %+v", payload)
	}
}
