import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

const errorRate = new Rate('login_errors');
const loginDuration = new Trend('login_duration');

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const POOL_SIZE = parseInt(__ENV.POOL_SIZE || '500');

const credentials = new SharedArray('credentials', function () {
  const data = [];
  for (let i = 0; i < POOL_SIZE; i++) {
    data.push({
      account: `load_login_${i}`,
      email: `load_login_${i}@test.local`,
      password: `Pass${i % 10000}word`,
    });
  }
  return data;
});

export const options = {
  scenarios: {
    login_throughput: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 100,
      maxVUs: 400,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<80', 'p(99)<100'],
    http_req_failed: ['rate<0.01'],
    login_duration: ['p(95)<80', 'p(99)<100'],
  },
};

export default function () {
  const cred = credentials[__VU % credentials.length];

  const payload = JSON.stringify({
    account: cred.account,
    password: cred.password,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'POST /api/v1/auth/login' },
  };

  const res = http.post(`${BASE_URL}/api/v1/auth/login`, payload, params);
  loginDuration.add(res.timings.duration);
  errorRate.add(res.status !== 200);

  if (res.status !== 200) {
    console.error(`VU ${__VU}: login failed status=${res.status} body=${res.body}`);
  }
}

export function handleSummary(data) {
  return {
    'test/load/login-throughput-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data),
  };
}

function textSummary(data) {
  const m = data.metrics;
  let out = '\n========== Login Throughput Test ==========\n';
  out += `Duration: ${(data.state.testRunDurationMs / 1000).toFixed(1)}s\n`;
  out += `Target: 1000 login req/s\n`;

  if (m.http_reqs) {
    const v = m.http_reqs.values;
    out += `Actual req/s: ${(v.rate || 0).toFixed(1)}\n`;
    out += `Total requests: ${v.count}\n`;
  }

  out += '\n--- Login Latency (ms) ---\n';
  if (m.http_req_duration) {
    const d = m.http_req_duration.values;
    out += `  avg: ${(d.avg || 0).toFixed(2)}  p50: ${(d.med || 0).toFixed(2)}  p90: ${(d['p(90)'] || 0).toFixed(2)}  p95: ${(d['p(95)'] || 0).toFixed(2)}\n`;
    out += `  min: ${(d.min || 0).toFixed(2)}  max: ${(d.max || 0).toFixed(2)}\n`;
  }

  out += '\n--- SLO Thresholds ---\n';
  if (m.http_req_duration && m.http_req_duration.thresholds) {
    for (const [k, v] of Object.entries(m.http_req_duration.thresholds)) {
      out += `  ${k}: ${v.ok ? 'PASS' : 'FAIL'}\n`;
    }
  }

  out += '\n--- Error Rate ---\n';
  if (m.http_req_failed) {
    const f = m.http_req_failed.values;
    out += `  http_req_failed: ${((f.rate || 0) * 100).toFixed(2)}%\n`;
  }
  out += '================================================\n';
  return out;
}
