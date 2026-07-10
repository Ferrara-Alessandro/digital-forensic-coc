'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

const QUERIES = ['ReadReperto', 'RepertoExists', 'OttieniStoriaReperto'];

class QueryMixWorkload extends WorkloadModuleBase {
    async submitTransaction() {
        const fn = QUERIES[Math.floor(Math.random() * QUERIES.length)];
        const request = {
            contractId: this.roundArguments.contractId,
            contractFunction: fn,
            invokerIdentity: 'Admin',
            invokerMspId: 'PGMSP',
            contractArguments: [this.roundArguments.repertoId],
            readOnly: true,
        };
        await this.sutAdapter.sendRequests(request);
    }
}

function createWorkloadModule() {
    return new QueryMixWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
