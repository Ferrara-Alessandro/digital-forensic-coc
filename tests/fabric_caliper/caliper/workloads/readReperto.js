'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class ReadRepertoWorkload extends WorkloadModuleBase {
    async submitTransaction() {
        const request = {
            contractId: this.roundArguments.contractId,
            contractFunction: 'ReadReperto',
            invokerIdentity: 'Admin',
            invokerMspId: 'PGMSP',
            contractArguments: [this.roundArguments.repertoId],
            readOnly: true,
        };
        await this.sutAdapter.sendRequests(request);
    }
}

function createWorkloadModule() {
    return new ReadRepertoWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
