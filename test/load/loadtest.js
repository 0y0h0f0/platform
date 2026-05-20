import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration');
const registerDuration = new Trend('register_duration');
const meDuration = new Trend('me_duration');
const listProjectsDuration = new Trend('list_projects_duration');
const createTaskDuration = new Trend('create_task_duration');

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const VUS = parseInt(__ENV.VUS) || 100;
const DURATION = __ENV.DURATION || '60s';

export const options = {
  stages: [
    { duration: '10s', target: Math.floor(VUS * 0.2) },
    { duration: '20s', target: Math.floor(VUS * 0.5) },
    { duration: '20s', target: VUS },
    { duration: '30s', target: VUS },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.05'],
    login_duration: ['p(99)<100'],
  },
  noConnectionReuse: false,
};

function randomString(length) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export default function () {
  group('auth flow', () => {
    const username = `load_${randomString(8)}`;
    const password = 'test123456';
    const email = `${username}@test.com`;

    const registerPayload = JSON.stringify({
      username: username,
      email: email,
      password: password,
    });

    const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'register' },
    });
    registerDuration.add(registerRes.timings.duration);
    errorRate.add(!check(registerRes, { 'register status 200': (r) => r.status === 200 }));

    if (registerRes.status !== 200) return;

    const loginPayload = JSON.stringify({
      account: username,
      password: password,
    });

    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'login' },
    });
    loginDuration.add(loginRes.timings.duration);
    errorRate.add(!check(loginRes, { 'login status 200': (r) => r.status === 200 }));

    if (loginRes.status !== 200) return;

    const body = JSON.parse(loginRes.body);
    const token = body.data.token;
    const authHeaders = {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    };

    group('authenticated reads', () => {
      const meRes = http.get(`${BASE_URL}/api/v1/users/me`, {
        headers: authHeaders,
        tags: { name: 'me' },
      });
      meDuration.add(meRes.timings.duration);
      errorRate.add(!check(meRes, { 'me status 200': (r) => r.status === 200 }));

      const listRes = http.get(`${BASE_URL}/api/v1/projects?limit=20`, {
        headers: authHeaders,
        tags: { name: 'list_projects' },
      });
      listProjectsDuration.add(listRes.timings.duration);
      errorRate.add(!check(listRes, { 'list projects status 200': (r) => r.status === 200 }));
    });

    group('create project and task', () => {
      const projectPayload = JSON.stringify({
        name: `project_${randomString(6)}`,
        description: 'load test project',
      });

      const projectRes = http.post(`${BASE_URL}/api/v1/projects`, projectPayload, {
        headers: authHeaders,
        tags: { name: 'create_project' },
      });
      errorRate.add(!check(projectRes, { 'create project status 200': (r) => r.status === 200 }));

      if (projectRes.status !== 200) return;

      const project = JSON.parse(projectRes.body).data;
      const projectId = project.id;

      const taskPayload = JSON.stringify({
        project_id: projectId,
        title: `task_${randomString(6)}`,
        content: 'load test task',
        priority: 1,
      });

      const taskRes = http.post(`${BASE_URL}/api/v1/tasks`, taskPayload, {
        headers: authHeaders,
        tags: { name: 'create_task' },
      });
      createTaskDuration.add(taskRes.timings.duration);
      errorRate.add(!check(taskRes, { 'create task status 200': (r) => r.status === 200 }));

      if (taskRes.status !== 200) return;

      const task = JSON.parse(taskRes.body).data;
      const taskId = task.id;

      const listTasksRes = http.get(
        `${BASE_URL}/api/v1/tasks?project_id=${projectId}&limit=20`,
        { headers: authHeaders, tags: { name: 'list_tasks' } }
      );
      errorRate.add(
        !check(listTasksRes, { 'list tasks status 200': (r) => r.status === 200 })
      );

      const commentPayload = JSON.stringify({ content: 'load test comment' });
      const commentRes = http.post(
        `${BASE_URL}/api/v1/tasks/${taskId}/comments`,
        commentPayload,
        { headers: authHeaders, tags: { name: 'create_comment' } }
      );
      errorRate.add(
        !check(commentRes, { 'create comment status 200': (r) => r.status === 200 })
      );
    });

    sleep(1);
  });
}

export function handleSummary(data) {
  return {
    'test/load/summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: '  ', enableColors: true }),
  };
}

function textSummary(data, opts) {
  const { state, metrics } = data;
  let out = '';
  out += `\n========== Load Test Summary ==========\n`;
  out += `Duration: ${state.testRunDurationMs / 1000}s\n`;
  out += `VUs: ${state.vusMax} max\n`;
  out += `Requests: ${metrics.http_reqs?.values?.count || 0} total\n`;
  out += `\n--- HTTP Timing ---\n`;
  if (metrics.http_req_duration) {
    const d = metrics.http_req_duration.values;
    out += `  avg: ${(d.avg || 0).toFixed(2)}ms\n`;
    out += `  p50: ${(d.med || 0).toFixed(2)}ms\n`;
    out += `  p95: ${(d['p(95)'] || 0).toFixed(2)}ms\n`;
    out += `  p99: ${(d['p(99)'] || 0).toFixed(2)}ms\n`;
  }
  out += `\n--- Error Rate ---\n`;
  if (metrics.http_req_failed) {
    const f = metrics.http_req_failed.values;
    out += `  rate: ${((f.rate || 0) * 100).toFixed(2)}%\n`;
  }
  out += `\n--- Login Duration ---\n`;
  if (metrics.login_duration) {
    const d = metrics.login_duration.values;
    out += `  avg: ${(d.avg || 0).toFixed(2)}ms p99: ${(d['p(99)'] || 0).toFixed(2)}ms\n`;
  }
  out += `========================================\n`;
  return out;
}
