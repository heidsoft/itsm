'use strict';

/**
 * Deployment/image-build contract tests, derived from
 * docs/deployment-optimization.md (§3 Docker images, §5 build entrypoint).
 *
 * These lock the parts that can silently regress while the build stays green:
 *   - AI service: CPU-only torch must come from a dedicated --index-url.
 *     With --extra-index-url pip still resolves `torch>=2.1.0` from PyPI and
 *     pulls multi-GB CUDA wheels; if the PyTorch index is unreachable it
 *     silently falls back to that wheel with no build failure.
 *   - `make build-images` must actually call the unified 4-service builder.
 *     It used to be an alias of `make build` (compose, 2 images, REGISTRY
 *     ignored), which contradicted both docs/deployment-optimization.md §5
 *     and docs/dev-commands-reference.md.
 *   - Every image keeps multi-stage + pinned base + non-root runtime, and
 *     every build context keeps a .dockerignore.
 *
 * Assertions target executable lines (comment lines are stripped first), so
 * an explanatory comment cannot keep a stale contract green.
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');
const read = (rel) => fs.readFileSync(path.join(root, rel), 'utf8');
const codeOnly = (text) => text.split('\n').filter((l) => !/^\s*#/.test(l)).join('\n');

const DOCKERFILES = {
  backend: 'itsm-backend/Dockerfile.prod',
  frontend: 'itsm-frontend/Dockerfile',
  'ai-service': 'itsm-ai-service/Dockerfile',
  guidance_sidecar: 'itsm-backend/guidance_sidecar/Dockerfile',
};

const DOCKERIGNORES = [
  'itsm-backend/.dockerignore',
  'itsm-frontend/.dockerignore',
  'itsm-ai-service/.dockerignore',
  'itsm-backend/guidance_sidecar/.dockerignore',
];

const aiDockerfile = read(DOCKERFILES['ai-service']);
const sidecarDockerfile = read(DOCKERFILES.guidance_sidecar);
const builder = read('scripts/build-images.sh');
const makefile = read('Makefile');
const docsGate = read('.github/workflows/docs-gate.yml');

function assertTorchIsolatedIndex(dockerfile) {
  const code = codeOnly(dockerfile);
  assert.match(code, /pip install --index-url /, 'torch must be installed from a dedicated index');
  assert.doesNotMatch(
    code,
    /--extra-index-url/,
    'extra index still allows PyPI CUDA torch wheels to win resolution'
  );
  assert.match(code, /pip install -r requirements\.txt/);
  assert.doesNotMatch(code, /PIP_NO_CACHE_DIR/, 'would disable the pip cache mount');
  assert.match(code, /--mount=type=cache,target=\/root\/\.cache\/pip/);
}

test('AI service installs CPU-only torch from a dedicated index, then other deps from PyPI', () => {
  assertTorchIsolatedIndex(aiDockerfile);
  // Two separate RUN commands: torch first, requirements second. A single
  // combined `pip install -r requirements.txt --extra-index-url ...` was the
  // old (ineffective) shape.
  const code = codeOnly(aiDockerfile);
  const torchLine = code.split('\n').findIndex((l) => l.includes('pip install --index-url'));
  const reqLine = code.split('\n').findIndex((l) => l.includes('pip install -r requirements.txt'));
  assert.ok(torchLine >= 0 && reqLine > torchLine, 'torch must be installed before requirements.txt');
});

test('reverting to --extra-index-url makes the torch contract fail (reverse check)', () => {
  const broken = aiDockerfile
    .replace(/pip install --index-url "\$\{TORCH_INDEX\}" torch/g,
             'pip install -r requirements.txt --extra-index-url "${TORCH_INDEX}"')
    .replace(/RUN --mount=type=cache,target=\/root\/\.cache\/pip \\\n    \. \/opt\/venv\/bin\/activate \\\n    && pip install -r requirements\.txt\n/, '');
  assert.notEqual(broken, aiDockerfile);
  assert.throws(() => assertTorchIsolatedIndex(broken));
});

test('every service image is multi-stage, pinned and non-root', () => {
  for (const [name, rel] of Object.entries(DOCKERFILES)) {
    const code = codeOnly(read(rel));
    const stages = code.match(/^FROM\s/gm) || [];
    assert.ok(stages.length >= 2, `${name}: expected multi-stage build, found ${stages.length} FROM`);
    const froms = code.split('\n').filter((l) => /^FROM\s/.test(l));
    for (const from of froms) {
      assert.doesNotMatch(from, /:latest\b/, `${name}: floating base image ${from}`);
      assert.match(from, /[\w.-]+:[\w.-]+/, `${name}: base image must be pinned ${from}`);
    }
    assert.match(code, /^USER\s+(?!root\b)\S+/m, `${name}: runtime must not run as root`);
    assert.match(read(rel), /^# syntax=docker\/dockerfile:1\.7/m, `${name}: cache mounts need frontend 1.7+`);
  }
});

test('Python builders use a BuildKit pip cache mount and keep pip cache enabled', () => {
  for (const [name, text] of [['ai-service', aiDockerfile], ['sidecar', sidecarDockerfile]]) {
    const code = codeOnly(text);
    assert.match(code, /--mount=type=cache,target=\/root\/\.cache\/pip/, `${name}: missing pip cache mount`);
    assert.doesNotMatch(code, /PIP_NO_CACHE_DIR/, `${name}: pip cache is disabled`);
  }
});

test('every image build context has a .dockerignore excluding .git', () => {
  for (const rel of DOCKERIGNORES) {
    assert.ok(fs.existsSync(path.join(root, rel)), `missing ${rel}`);
    assert.match(read(rel), /^\.git$/m, `${rel}: .git must be excluded from the context`);
  }
});

test('unified builder covers all four services and enables inline cache', () => {
  for (const svc of Object.keys(DOCKERFILES)) {
    assert.match(builder, new RegExp(`"${svc}\\|`), `build-images.sh missing ${svc}`);
  }
  assert.match(builder, /BUILDKIT_INLINE_CACHE=1/);
  assert.match(builder, /DOCKER_BUILDKIT=1/);
});

test('make build-images calls the unified builder instead of aliasing compose build', () => {
  const target = makefile.match(/^build-images:[^\n]*$/m);
  assert.ok(target, 'build-images target missing');
  assert.doesNotMatch(target[0], /:\s*build\s*(?:#|$)/, 'must not be a compose build alias');
  const body = makefile.split('\n').slice(makefile.split('\n').findIndex((l) => /^build-images:/.test(l)) + 1)
    .filter((l) => /^\t/.test(l)).join('\n');
  assert.match(body, /scripts\/build-images\.sh/);
  assert.match(body, /REGISTRY="\$\(REGISTRY\)"/, 'REGISTRY must reach the builder');
});

test('docs-gate re-runs these contracts when a Dockerfile or Makefile changes', () => {
  for (const name of ['pull_request', 'push']) {
    const end = name === 'pull_request' ? 'push' : 'workflow_call';
    const re = new RegExp(`^  ${name}:[\\s\\S]*?(?=^  ${end}:)`, 'm');
    const section = docsGate.match(re);
    assert.ok(section, `missing ${name} trigger section`);
    for (const p of ['**/Dockerfile*', '**/.dockerignore', 'Makefile', 'scripts/build-images.sh']) {
      assert.ok(section[0].includes(`- '${p}'`), `docs-gate ${name} trigger missing ${p}`);
    }
  }
});
