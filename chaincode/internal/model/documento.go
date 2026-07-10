// Strutture per un atto digitale su IPFS.
package model

// Parte che tutti i peer vedono: id documento e cid su IPFS.
type DocumentoPublic struct {
	IDDocumento string `json:"idDocumento"`
	CID         string `json:"cid"`
}

// Descrizione, tipo atto, chiave di cifratura: solo chi ha la collezione privata.
type DocumentoPrivate struct {
	IDCaso          string `json:"idCaso"`
	IDReperto       string `json:"idReperto,omitempty" metadata:",optional"`
	TipoDocumento   string `json:"tipoDocumento"`
	Descrizione     string `json:"descrizione"`
	RegistratoIl    string `json:"registratoIl"`
	Autore          string `json:"autore"`
	RiferimentoEnte string `json:"riferimentoEnte,omitempty" metadata:",optional"`
	ChiaveCifrata   string `json:"chiaveCifrata,omitempty" metadata:",optional"`
}

// Documento completo in lettura (pubblico e privato se permesso).
type Documento struct {
	IDDocumento     string `json:"idDocumento"`
	CID             string `json:"cid"`
	IDCaso          string `json:"idCaso"`
	IDReperto       string `json:"idReperto,omitempty" metadata:",optional"`
	TipoDocumento   string `json:"tipoDocumento"`
	Descrizione     string `json:"descrizione"`
	RegistratoIl    string `json:"registratoIl"`
	Autore          string `json:"autore"`
	RiferimentoEnte string `json:"riferimentoEnte,omitempty" metadata:",optional"`
	ChiaveCifrata   string `json:"chiaveCifrata,omitempty" metadata:",optional"`
}
