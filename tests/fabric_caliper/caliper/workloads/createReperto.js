'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

let seq = 0;

class CreateRepertoWorkload extends WorkloadModuleBase {
    async submitTransaction() {
        seq += 1;
        const id = `REP-CAL-${Date.now()}-${seq}`;
        const payload = JSON.stringify({
            idCaso: 'CASO-BENCH',
            idAgente: 'AG-BENCH',
            idDistretto: 'PG-SALERNO',
            dataOraPrelievo: '2026-01-01T12:00:00Z',
            descrizioneBene: `reperto caliper ${id}`,
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
    return new CreateRepertoWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
