/**
 * service-request-fulfillment scenario logic tests.
 *
 * The scenario's approval step drives the chain level by level because the
 * product's fulfillment gate is `security_approved` and one approve call only
 * advances one level. These tests pin that loop with fake read/approve
 * functions — no backend needed:
 *
 *   · a serial chain reaches the gate and stops voting (no wasted calls);
 *   · a quorum level that stops moving the status is `blocked`, not `failed`
 *     (environment gap: the flow's single identity cannot satisfy the level);
 *   · a request already at the gate (or rejected/cancelled) returns immediately.
 *
 * The `blocked` classification is the whole point: a stuck chain is missing
 * approver accounts, which is a provisioning task, not a product defect.
 */
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { approveThroughChain } from '../dist/agent/flow/scenarios/service-request.js';
import { CliError } from '../dist/lib/errors.js';

/** Fake request store: status advances one level per approve call, like the product. */
function chain(levels) {
  let idx = 0;
  return {
    read: async () => ({ status: levels[idx] }),
    approve: async () => {
      if (idx < levels.length - 1) idx++;
    },
    votes: () => idx,
  };
}

test('serial chain: votes once per level and stops at the fulfillment gate', async () => {
  const c = chain(['submitted', 'manager_approved', 'security_approved']);
  const final = await approveThroughChain(7, c.read, c.approve);
  assert.equal(final.status, 'security_approved');
  assert.equal(c.votes(), 2, 'two levels advanced = two votes; no extra call after the gate');
});

test('quorum level that stops moving the status is blocked, not failed', async () => {
  // A parallel level needs other approvers: approve succeeds (200) but the
  // status never leaves 'manager_approved'.
  let approveCalls = 0;
  const read = async () => ({ status: 'manager_approved' });
  const approve = async () => {
    approveCalls++;
  };
  await assert.rejects(
    () => approveThroughChain(7, read, approve),
    (err) => {
      assert.ok(err instanceof CliError, 'must be a CliError so the runner maps it to blocked');
      assert.equal(err.agentError.kind, 'unavailable');
      assert.match(err.message, /cannot advance past 'manager_approved'/);
      assert.match(err.message, /parallel level needs its other approvers/);
      return true;
    }
  );
  assert.equal(approveCalls, 1, 'stop at the first non-advancing vote; do not hammer the endpoint');
});

test('request already at the gate returns without voting', async () => {
  let approveCalls = 0;
  const final = await approveThroughChain(
    7,
    async () => ({ status: 'security_approved' }),
    async () => {
      approveCalls++;
    }
  );
  assert.equal(final.status, 'security_approved');
  assert.equal(approveCalls, 0);
});

test('rejected/cancelled requests return immediately', async () => {
  for (const status of ['rejected', 'cancelled']) {
    let approveCalls = 0;
    const final = await approveThroughChain(
      7,
      async () => ({ status }),
      async () => {
        approveCalls++;
      }
    );
    assert.equal(final.status, status);
    assert.equal(approveCalls, 0, `${status} must not be voted on`);
  }
});
