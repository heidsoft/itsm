/**
 * k6 baseline — POST /auth/login
 *
 * Added by PR-0.4 as the seed sample for PR-4.1. PR-4.1 will turn this
 * into a multi-stage VU ramp that exercises the documented Top-10
 * endpoints and posts P95 against the perf-budget.yml threshold.
 *
 * Conventions (PR-4.2):
 *   - Read base URL from `K6_BASE_URL` env var.
 *   - Read credentials from `K6_USER_EMAIL` / `K6_USER_PASSWORD`.
 *   - Always export custom `Trend('ttfb')` so perf-budget.yml can read
 *     P95 from the JSON summary.
 *
 * Run locally:
 *   K6_BASE_URL=http://localhost:8090/api/v1 \
 *   K6_USER_EMAIL=admin@example.com \
 *   K6_USER_PASSWORD=$(cat secrets/admin.pwd) \
 *   k6 run itsm-backend/tests/load/baseline/login.js
 */

import http from 'k6/http';
import { Trend } from 'k6/metrics';
import { check } from 'k6';

const ttfb = new Trend('ttfb');

const baseUrl = __ENV.K6_BASE_URL || 'http://localhost:8090/api/v1';

export const options = {
  // Stage 1 only (1 RPS) — safe in dev. PR-4.1 adds the 10/50 RPS curves.
  scenarios: {
    smoke_login: {
      executor: 'constant-arrival-rate',
      rate: 1,
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 4,
      maxVUs: 8,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    ttfb: ['p(95)<500'], // ms; PR-4.2 will baseline against main P95 + 20%
  },
};

export default function () {
  const res = http.post(`${baseUrl}/auth/login`, JSON.stringify({
    email: __ENV.K6_USER_EMAIL || 'admin@example.com',
    password: __ENV.K6_USER_PASSWORD || 'changeme',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  ttfb.add(res.timings.waiting);

  check(res, {
    'login returns 200': (r) => r.status === 200,
    'login body has token': (r) => {
      try {
        return !!JSON.parse(r.body || '{}').data?.token;
      } catch (_e) {
        return false;
      }
    },
  });
}
