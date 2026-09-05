'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');

test('recognizes permissions declared on multiline route registrations', () => {
  const output = path.join(os.tmpdir(), `itsm-acl-${process.pid}.yaml`);

  try {
    execFileSync(process.execPath, [
      path.resolve(__dirname, '../generate-acl-manifest.js'),
      '--check',
      '--output',
      output,
    ]);

    const manifest = fs.readFileSync(output, 'utf8');
    assert.match(manifest, /path: \/api\/v1\/ai\/incidents\/:id\/analyze\n\s+permission: ai\.read/);
    assert.doesNotMatch(manifest, /MISSING ACL/);
  } finally {
    fs.rmSync(output, { force: true });
  }
});
