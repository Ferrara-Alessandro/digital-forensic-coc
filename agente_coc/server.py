"""
Server MCP per la catena di custodia forense su Hyperledger Fabric.
Espone le operazioni del chaincode 'reperto' come tool utilizzabili
da qualsiasi agente MCP-compatibile (Cursor, Claude Desktop, ecc.).
"""

import json
import subprocess
import sys
from pathlib import Path

from fastmcp import FastMCP

ROOT = Path(__file__).parent.parent.resolve()
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from generazione_reperti import genera_immagine, genera_testo
BIN = ROOT / "bin"
PKI = ROOT / "infrastruttura_blockchain" / "certificati_pki"

mcp = FastMCP("coc-fabric")


def _run(args: list[str]) -> dict:
    """Lancio il binario Go e restituisco il JSON di output."""
    result = subprocess.run(args, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(result.stderr.strip() or f"exit code {result.returncode}")
    return json.loads(result.stdout)


def _upload(*args) -> dict:
    return _run([str(BIN / "upload")] + list(args) + ["-pki", str(PKI), "-ipfs-timeout", "120s"])


def _workflow(*args) -> dict:
    return _run([str(BIN / "workflow")] + list(args) + ["-pki", str(PKI)])


def _download(*args) -> dict:
    return _run([str(BIN / "download")] + list(args) + ["-pki", str(PKI)])


# ---------------------------------------------------------------------------
# Reperto
# ---------------------------------------------------------------------------

@mcp.tool()
def crea_reperto(
    id_reperto: str,
    id_caso: str,
    id_agente: str,
    id_distretto: str,
    descrizione_bene: str,
    data_ora_prelievo: str = "",
) -> dict:
    """
    Crea la scheda di un reperto sequestrato sul ledger Fabric.
    I metadati riservati (agente, distretto, descrizione) finiscono
    nella collezione privata PG-PM; la parte pubblica è visibile a tutti.
    Restituisce il transaction ID del commit.
    """
    args = [
        "-mode", "reperto",
        "-id-reperto", id_reperto,
        "-id-caso", id_caso,
        "-id-agente", id_agente,
        "-id-distretto", id_distretto,
        "-descrizione-bene", descrizione_bene,
    ]
    if data_ora_prelievo:
        args += ["-data-ora-prelievo", data_ora_prelievo]
    return _upload(*args)


@mcp.tool()
def leggi_reperto(id_reperto: str, org: str = "pg") -> dict:
    """
    Legge la scheda pubblica del reperto dal ledger.
    Restituisce stato attuale, custode, riferimenti ai documenti allegati.
    """
    return _workflow("-mode", "leggi-reperto", "-id-reperto", id_reperto, "-org", org)


@mcp.tool()
def storia_reperto(id_reperto: str, org: str = "pg") -> dict:
    """
    Restituisce la cronologia completa delle transazioni sul reperto:
    ogni voce ha txId, timestamp e snapshot dello stato al momento del commit.
    Utile per audit e verifica dell'integrità della catena di custodia.
    """
    return _workflow("-mode", "storia-reperto", "-id-reperto", id_reperto, "-org", org)


# ---------------------------------------------------------------------------
# Ciclo di vita
# ---------------------------------------------------------------------------

@mcp.tool()
def richiedi_analisi(
    id_reperto: str,
    id_lab: str,
    tipo_analisi: str,
    cid_decreto: str,
    chiave_decreto: str,
) -> dict:
    """
    Il PM autorizza l'analisi: collega il decreto di accertamento al reperto,
    registra il laboratorio destinatario e porta lo stato a ATTESA_TRASPORTO.
    Richiede le credenziali PM (org=pm).
    """
    return _workflow(
        "-mode", "richiedi-analisi",
        "-id-reperto", id_reperto,
        "-id-lab", id_lab,
        "-tipo-analisi", tipo_analisi,
        "-cid-decreto", cid_decreto,
        "-chiave-decreto", chiave_decreto,
        "-org", "pm",
    )


@mcp.tool()
def avvia_trasporto(id_reperto: str, id_agente_pg: str) -> dict:
    """
    La PG segna che il reperto è in viaggio verso il laboratorio.
    Porta lo stato a IN_TRANSITO e aggiorna il custode attuale.
    """
    return _workflow(
        "-mode", "avvia-trasporto",
        "-id-reperto", id_reperto,
        "-id-agente-pg", id_agente_pg,
        "-org", "pg",
    )


@mcp.tool()
def ricevi_in_laboratorio(id_reperto: str, id_laboratorio: str) -> dict:
    """
    Il laboratorio conferma la ricezione del reperto.
    Porta lo stato a IN_ANALISI e aggiorna il custode.
    """
    return _workflow(
        "-mode", "ricevi-laboratorio",
        "-id-reperto", id_reperto,
        "-id-laboratorio", id_laboratorio,
        "-org", "lab",
    )


@mcp.tool()
def completa_analisi(
    id_reperto: str,
    cid_relazione: str,
    chiave_relazione: str,
) -> dict:
    """
    Il laboratorio carica la relazione tecnica e porta lo stato a ATTESA_RITIRO.
    """
    return _workflow(
        "-mode", "completa-analisi",
        "-id-reperto", id_reperto,
        "-cid-relazione", cid_relazione,
        "-chiave-relazione", chiave_relazione,
        "-org", "lab",
    )


@mcp.tool()
def prepara_riconsegna(id_reperto: str) -> dict:
    """
    Il laboratorio dichiara che il reperto è pronto per essere ritirato.
    Verifica che lo stato sia ATTESA_RITIRO.
    """
    return _workflow(
        "-mode", "prepara-riconsegna",
        "-id-reperto", id_reperto,
        "-org", "lab",
    )


@mcp.tool()
def deposita_in_sede(
    id_reperto: str,
    cid_verbale_riconsegna: str,
    chiave_verbale_riconsegna: str,
) -> dict:
    """
    La PG registra il verbale di riconsegna e chiude il ciclo:
    lo stato torna a SEQUESTRATO con custode uguale al distretto originale.
    """
    return _workflow(
        "-mode", "deposita-sede",
        "-id-reperto", id_reperto,
        "-cid-verbale-riconsegna", cid_verbale_riconsegna,
        "-chiave-verbale-riconsegna", chiave_verbale_riconsegna,
        "-org", "pg",
    )


# ---------------------------------------------------------------------------
# Generazione contenuti probatori (AI)
# ---------------------------------------------------------------------------

@mcp.tool()
def genera_evidenza_testo(
    descrizione: str,
    tipo: str = "",
    model: str = "llama3",
) -> dict:
    """
    Crea un file testuale di evidenza in generazione_reperti/testi/.
    Prova Ollama in locale; se non risponde usa un template.
    Restituisce percorso relativo, SHA-256 e tipo usato.
    """
    return genera_testo(descrizione, tipo=tipo, model=model)


@mcp.tool()
def genera_evidenza_immagine(descrizione: str, seed: int = 12345) -> dict:
    """
    Crea un PNG in generazione_reperti/immagini/ con Stable Diffusion.
    Richiede il venv in generazione_reperti/ (setup.sh).
    """
    return genera_immagine(descrizione, seed=seed)


# ---------------------------------------------------------------------------
# Documenti ed evidenze
# ---------------------------------------------------------------------------

@mcp.tool()
def leggi_documento(id_caso: str, id_documento: str, org: str = "pg") -> dict:
    """
    Legge i metadati pubblici e privati di un documento dal ledger.
    Restituisce CID, chiave cifrata (base64), tipo, autore e timestamp.
    Utile per recuperare la chiave prima di completa_analisi o deposita_in_sede.
    """
    return _workflow("-mode", "leggi-documento", "-id-caso", id_caso, "-id-documento", id_documento, "-org", org)


@mcp.tool()
def leggi_evidenza(id_caso: str, id_evidenza: str, org: str = "pg") -> dict:
    """
    Legge i metadati pubblici e privati di un'evidenza dal ledger.
    Restituisce CID, chiave cifrata (base64), classe e timestamp.
    """
    return _workflow("-mode", "leggi-evidenza", "-id-caso", id_caso, "-id-evidenza", id_evidenza, "-org", org)


@mcp.tool()
def registra_documento(
    file: str,
    id_caso: str,
    id_documento: str,
    tipo_documento: str,
    descrizione: str,
    id_reperto: str = "",
    riferimento_ente: str = "",
    ingest_org: str = "",
) -> dict:
    """
    Cifra il file, lo carica su IPFS e registra il documento sul ledger.
    tipo_documento deve essere uno tra: VERBALE_SOPRALLUOGO, VERBALE_SEQUESTRO,
    DECRETO_ACCERTAMENTO, RELAZIONE_TECNICA, VERBALE_RICONSEGNA.
    La collezione PDC viene scelta automaticamente in base al tipo.
    """
    args = [
        "-mode", "documento",
        "-file", file,
        "-id-caso", id_caso,
        "-id-documento", id_documento,
        "-tipo-documento", tipo_documento,
        "-descrizione-documento", descrizione,
    ]
    if id_reperto:
        args += ["-id-reperto-documento", id_reperto]
    if riferimento_ente:
        args += ["-riferimento-ente", riferimento_ente]
    if ingest_org:
        args += ["-ingest-org", ingest_org]
    return _upload(*args)


@mcp.tool()
def registra_evidenza(
    file: str,
    id_caso: str,
    id_evidenza: str,
    descrizione: str,
    id_reperto: str = "",
    classe: str = "",
) -> dict:
    """
    Cifra il file, lo carica su IPFS e registra l'evidenza sul ledger
    nella collezione privata PG-PM.
    """
    args = [
        "-mode", "evidenza",
        "-file", file,
        "-id-caso", id_caso,
        "-id-evidenza", id_evidenza,
        "-descrizione-evidenza", descrizione,
    ]
    if id_reperto:
        args += ["-id-reperto-evidenza", id_reperto]
    if classe:
        args += ["-classe-evidenza", classe]
    return _upload(*args)


@mcp.tool()
def scarica_documento(
    id_caso: str,
    id_documento: str,
    org: str = "pg",
    out_dir: str = "",
) -> dict:
    """
    Legge i metadati dal ledger, scarica il file da IPFS e lo decifra.
    Usa le credenziali dell'org indicata (pg | pm | lab) per la query Fabric.
    Restituisce il percorso del file ripristinato e i tempi di operazione.
    """
    out = out_dir or str(ROOT / "downloads")
    return _download(
        "-mode", "documento",
        "-id-caso", id_caso,
        "-id-documento", id_documento,
        "-org", org,
        "-out-dir", out,
    )


@mcp.tool()
def scarica_evidenza(
    id_caso: str,
    id_evidenza: str,
    org: str = "pg",
    out_dir: str = "",
) -> dict:
    """
    Legge i metadati dal ledger, scarica il file da IPFS e lo decifra.
    Usa le credenziali dell'org indicata (pg | pm | lab) per la query Fabric.
    """
    out = out_dir or str(ROOT / "downloads")
    return _download(
        "-mode", "evidenza",
        "-id-caso", id_caso,
        "-id-evidenza", id_evidenza,
        "-org", org,
        "-out-dir", out,
    )


if __name__ == "__main__":
    mcp.run()
