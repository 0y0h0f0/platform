import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const meDuration = new Trend('me_duration');
const listDuration = new Trend('list_duration');

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const TOKEN = __ENV.TOKEN;
if (!TOKEN) { throw new Error('TOKEN env var required. Generate a fresh token first.'); }

export const options = {
  scenarios: {
    throughput: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${TOKEN}`,
  };

  const meRes = http.get(`${BASE_URL}/api/v1/users/me`, { headers });
  meDuration.add(meRes.timings.duration);
  errorRate.add(!check(meRes, { 'me 200': (r) => r.status === 200 }));

  const listRes = http.get(`${BASE_URL}/api/v1/projects?limit=20`, { headers });
  listDuration.add(listRes.timings.duration);
  errorRate.add(!check(listRes, { 'list 200': (r) => r.status === 200 }));
}

export function handleSummary(data) {
  return {
    'test/load/throughput-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data),
  };
}

function textSummary(data) {
  const m = data.metrics;
  let out = '\n========== 1k QPS Throughput Test ==========\n';
  out += `Duration: ${(data.state.testRunDurationMs / 1000).toFixed(1)}s\n`;
  out += `Target: 500 iterations/s (1000 HTTP req/s)\n`;

  if (m.http_reqs) {
    const v = m.http_reqs.values;
    out += `Actual req/s: ${(v.rate || 0).toFixed(1)}\n`;
    out += `Total requests: ${v.count}\n`;
  }

  out += '\n--- HTTP Timing ---\n';
  if (m.http_req_duration) {
    const d = m.http_req_duration.values;
    out += `  avg: ${(d.avg || 0).toFixed(2)}ms  p50: ${(d.med || 0).toFixed(2)}ms  p95: ${(d['p(95)'] || 0).toFixed(2)}ms  p99: ${(d['p(99)'] || 0).toFixed(2)}ms\n`;
  }

  out += '\n--- Endpoint Timing ---\n';
  if (m.me_duration && m.me_duration.values.count > 0) {
    const d = m.me_duration.values;
    out += `  GET /me:       avg=${d.avg.toFixed(2)}ms p99=${d['p(99)'].toFixed(2)}ms\n`;
  }
  if (m.list_duration && m.list_duration.values.count > 0) {
    const d = m.list_duration.values;
    out += `  GET /projects:  avg=${d.avg.toFixed(2)}ms p99=${d['p(99)'].toFixed(2)}ms\n`;
  }

  out += '\n--- Error Rate ---\n';
  if (m.http_req_failed) {
    const f = m.http_req_failed.values;
    out += `  http_req_failed: ${((f.rate || 0) * 100).toFixed(2)}%\n`;
  }
  out += '================================================\n';
  return out;
}
