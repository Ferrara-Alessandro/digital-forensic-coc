// Strutture del reperto: parte pubblica sul ledger e parte privata tra PG e PM.
package model

// Dati reperto che tutti i nodi possono leggere: stato, custode, id degli atti collegati.
type RepertoPublic struct {
	ID_Reperto              string `json:"idReperto"`
	ID_Caso                 string `json:"idCaso,omitempty" metadata:",optional"`
	Tipo_Analisi            string `json:"tipoAnalisi,omitempty" metadata:",optional"`
	Stato                   string `json:"stato"`
	CustodeAttuale          string `json:"custodeAttuale"`
	LaboratorioDestinazione string `json:"laboratorioDestinazione,omitempty" metadata:",optional"`
	IDVerbaleSequestro      string `json:"idVerbaleSequestro,omitempty" metadata:",optional"`
	IDDecretoAccertamento   string `json:"idDecretoAccertamento,omitempty" metadata:",optional"`
	IDRelazioneTecnica      string `json:"idRelazioneTecnica,omitempty" metadata:",optional"`
	IDVerbaleRiconsegna     string `json:"idVerbaleRiconsegna,omitempty" metadata:",optional"`
}

// Dati reperto riservati: li vedono solo PG e PM nella collezione privata.
type RepertoPrivate struct {
	ID_Caso           string `json:"idCaso"`
	ID_Agente         string `json:"idAgente"`
	ID_Distretto      string `json:"idDistretto"`
	Data_Ora_Prelievo string `json:"dataOraPrelievo"`
	Descrizione_Bene  string `json:"descrizioneBene"`
}

// Oggetto che restituisco quando leggo un reperto, se ho accesso anche ai dati privati.
type Reperto struct {
	ID_Caso                 string `json:"idCaso"`
	ID_Agente               string `json:"idAgente,omitempty" metadata:",optional"`
	ID_Distretto            string `json:"idDistretto,omitempty" metadata:",optional"`
	ID_Reperto              string `json:"idReperto"`
	Data_Ora_Prelievo       string `json:"dataOraPrelievo,omitempty" metadata:",optional"`
	Tipo_Analisi            string `json:"tipoAnalisi,omitempty" metadata:",optional"`
	Descrizione_Bene        string `json:"descrizioneBene,omitempty" metadata:",optional"`
	Stato                   string `json:"stato"`
	CustodeAttuale          string `json:"custodeAttuale"`
	LaboratorioDestinazione string `json:"laboratorioDestinazione,omitempty" metadata:",optional"`
	IDVerbaleSequestro      string `json:"idVerbaleSequestro,omitempty" metadata:",optional"`
	IDDecretoAccertamento   string `json:"idDecretoAccertamento,omitempty" metadata:",optional"`
	IDRelazioneTecnica      string `json:"idRelazioneTecnica,omitempty" metadata:",optional"`
	IDVerbaleRiconsegna     string `json:"idVerbaleRiconsegna,omitempty" metadata:",optional"`
}
