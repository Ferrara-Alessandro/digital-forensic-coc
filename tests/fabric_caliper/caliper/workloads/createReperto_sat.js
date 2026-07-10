'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

let seq = 0;

// Variante per i test di saturazione: identificativo ad alta entropia per
// evitare collisioni quando piu' worker generano reperti in parallelo.
class CreateRepertoSatWorkload extends WorkloadModuleBase {
    async submitTransaction() {
        seq += 1;
        const rnd = Math.floor(Math.random() * 1e9);
        const id = `REP-SAT-${process.pid}-${Date.now()}-${seq}-${rnd}`;
        const payload = JSON.stringify({
            idCaso: 'CASO-BENCH',
            idAgente: 'AG-BENCH',
            idDistretto: 'PG-SALERNO',
            dataOraPrelievo: '2026-01-01T12:00:00Z',
            descrizioneBene: `reperto saturazione ${id}`,
        });

        const request = {
            contractId: this.roundArguments.contractId,
            contractFunction: 'CreaReperto',
            invokerIdentity: 'Admin',
            invokerMspId: 'PGMSP',
            contractArguments: [id],
            readOnly: false,
            targetPeers: ['PGMSP_localhost:7051', 'PMMSP_localhost:8051'],
            transientMap: {
                reperto_privato: Buffer.from(payload),
            },
        };
        await this.sutAdapter.sendRequests(request);
    }
}

function createWorkloadModule() {
    return new CreateRepertoSatWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
