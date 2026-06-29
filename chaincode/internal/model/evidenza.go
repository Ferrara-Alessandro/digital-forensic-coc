// Strutture per file digitali generici legati al caso.
package model

// Parte pubblica: id evidenza e cid IPFS.
type EvidenzaPublic struct {
	IDEvidenza string `json:"idEvidenza"`
	CID        string `json:"cid"`
}

// Metadati riservati a PG e PM (descrizione, classe, chiave AES).
type EvidenzaPrivate struct {
	IDCaso        string `json:"idCaso"`
	IDReperto     string `json:"idReperto,omitempty" metadata:",optional"`
	Descrizione   string `json:"descrizione"`
	Classe        string `json:"classe,omitempty" metadata:",optional"`
	RegistratoIl  string `json:"registratoIl"`
	Autore        string `json:"autore"`
	ChiaveCifrata string `json:"chiaveCifrata,omitempty" metadata:",optional"`
}

// Evidenza completa in lettura.
type Evidenza struct {
	IDEvidenza    string `json:"idEvidenza"`
	CID           string `json:"cid"`
	IDCaso        string `json:"idCaso"`
	IDReperto     string `json:"idReperto,omitempty" metadata:",optional"`
	Descrizione   string `json:"descrizione"`
	Classe        string `json:"classe,omitempty" metadata:",optional"`
	RegistratoIl  string `json:"registratoIl"`
	Autore        string `json:"autore"`
	ChiaveCifrata string `json:"chiaveCifrata,omitempty" metadata:",optional"`
}
