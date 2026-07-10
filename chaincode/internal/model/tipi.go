// Costanti per stati del reperto e tipi di documento.
package model

import "fmt"

// Stati del reperto sul world state.
const (
	StatoSequestrato     = "SEQUESTRATO"
	StatoAttesaTrasporto = "ATTESA_TRASPORTO"
	StatoInTransito      = "IN_TRANSITO"
	StatoInAnalisi       = "IN_ANALISI"
	StatoAttesaRitiro    = "ATTESA_RITIRO"
)

// Tipi di atto ammessi.
const (
	TipoVerbaleSopralluogo  = "VERBALE_SOPRALLUOGO"
	TipoVerbaleSequestro    = "VERBALE_SEQUESTRO"
	TipoDecretoAccertamento = "DECRETO_ACCERTAMENTO"
	TipoRelazioneTecnica    = "RELAZIONE_TECNICA"
	TipoVerbaleRiconsegna   = "VERBALE_RICONSEGNA"
)

func ValidateTipoDocumento(t string) error {
	switch t {
	case TipoVerbaleSopralluogo, TipoVerbaleSequestro, TipoDecretoAccertamento,
		TipoRelazioneTecnica, TipoVerbaleRiconsegna:
		return nil
	default:
		return fmt.Errorf("tipoDocumento non consentito: %q", t)
	}
}
