// Costanti per stati del reperto e tipi di documento.
package model

import "fmt"

// Stati che uso nel campo stato del reperto.
const (
	StatoSequestrato     = "SEQUESTRATO"
	StatoAttesaTrasporto = "ATTESA_TRASPORTO"
	StatoInTransito      = "IN_TRANSITO"
	StatoInAnalisi       = "IN_ANALISI"
	StatoAttesaRitiro    = "ATTESA_RITIRO"
)

// Tipi di atto che accetto (verbali, decreto, relazione, ecc.).
const (
	TipoVerbaleSopralluogo  = "VERBALE_SOPRALLUOGO"
	TipoVerbaleSequestro    = "VERBALE_SEQUESTRO"
	TipoDecretoAccertamento = "DECRETO_ACCERTAMENTO"
	TipoRelazioneTecnica    = "RELAZIONE_TECNICA"
	TipoVerbaleRiconsegna   = "VERBALE_RICONSEGNA"
)

// Verifico che il tipo documento sia uno di quelli ammessi.
func ValidateTipoDocumento(t string) error {
	switch t {
	case TipoVerbaleSopralluogo, TipoVerbaleSequestro, TipoDecretoAccertamento,
		TipoRelazioneTecnica, TipoVerbaleRiconsegna:
		return nil
	default:
		return fmt.Errorf("tipoDocumento non consentito: %q", t)
	}
}

func IsTipoDocumento(t string) bool {
	return ValidateTipoDocumento(t) == nil
}
