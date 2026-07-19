'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const { evaluateContracts } = require('../check-engineering-contracts');

function fixture(content) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'engineering-contract-'));
  fs.writeFileSync(path.join(root, 'contract.txt'), content);
  return root;
}

const contract = {
  id: 'fixture',
  file: 'contract.txt',
  require: [/same-origin/],
  forbid: [/localhost:8090/],
};

test('passes when required policy exists and forbidden value is absent', () => {
  const result = evaluateContracts(fixture('same-origin'), [contract]);
  assert.equal(result.passed.length, 1);
  assert.deepEqual(result.failures, []);
});

test('reports missing and forbidden policy drift', () => {
  const result = evaluateContracts(fixture('localhost:8090'), [contract]);
  assert.equal(result.failures.length, 1);
  assert.match(result.failures[0].reason, /missing/);
  assert.match(result.failures[0].reason, /forbidden/);
});

test('reports a missing contract file', () => {
  const result = evaluateContracts(os.tmpdir(), [{ ...contract, file: 'missing-file' }]);
  assert.equal(result.failures[0].reason, 'file is missing');
});
