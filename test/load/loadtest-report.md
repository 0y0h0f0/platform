# Load Test Report

## Environment

| Item | Value |
|------|-------|
| Tool | vegeta v12.13.0 |
| Target | `http://127.0.0.1:8080` (api-gateway) |
| PostgreSQL | 16 (Docker, shared with test services) |
| Redis | 7 (Docker, shared with test services) |
| Go Services | 1 instance each (api-gateway, user-service, task-service) |
| Machine | Linux 6.17.0-23-generic, 16GB RAM, 8 vCPU (AMD Ryzen) |
| Rate Limits | auth=5/s, IP=60/s, user=100/s (production defaults) |

## Test Scenarios

### Scenario 1: Login (4 QPS sustained, 30s)

Target: `POST /api/v1/auth/login` with bcrypt cost=10.

```bash
vegeta attack -rate=4 -duration=30s -targets=vegeta_login.txt | vegeta report -type=text
```

**Rationale for 4 QPS:** The auth path rate limit is 5 req/s per IP. This test runs at 80% of that limit to get clean measurements without rate-limiting noise.

### Scenario 2: Authenticated Read - Me (50 QPS, 30s)

Target: `GET /api/v1/users/me` with valid JWT.

```bash
vegeta attack -rate=50 -duration=30s -targets=vegeta_me.txt | vegeta report -type=text
```

**Rationale for 50 QPS:** The IP rate limit for non-auth paths is 60 req/s. This tests at ~83% of that limit.

### Scenario 3: Authenticated Read - List Projects (50 QPS, 30s)

Target: `GET /api/v1/projects?limit=20` with valid JWT. Hits DB via gRPC.

```bash
vegeta attack -rate=50 -duration=30s -targets=vegeta_listp.txt | vegeta report -type=text
```

### Scenario 4: Authenticated Write - Create Task (50 QPS, 30s)

Target: `POST /api/v1/tasks` with valid JWT. Hits DB write + operation log.

```bash
vegeta attack -rate=50 -duration=30s -targets=vegeta_task.txt | vegeta report -type=text
```

## Results

### Scenario 1: Login (4 QPS)

| Metric | Value | SLO | Pass? |
|--------|-------|-----|-------|
| Total requests | 120 | - | - |
| Success rate | 100.00% | - | - |
| Mean latency | 56.6ms | - | - |
| P50 | 48.4ms | - | - |
| P95 | 130.4ms | - | - |
| P99 | 221.1ms | < 100ms | **FAIL** |
| Max | 229.3ms | - | - |
| Status | 200:120 | - | - |

**Note:** Login P99 exceeds the 100ms SLO. The bottleneck is bcrypt cost=10 (~50ms per verification) combined with DB user lookup and JWT generation. Reducing bcrypt cost to 8 would cut verification time to ~12ms but weakens password security — the tradeoff is documented as an interview talking point.

### Scenario 2: GetMe (50 QPS)

| Metric | Value | SLO | Pass? |
|--------|-------|-----|-------|
| Total requests | 1,500 | - | - |
| Success rate | 100.00% | < 5% errors | PASS |
| Mean latency | 1.57ms | - | - |
| P50 | 1.53ms | - | - |
| P95 | 2.14ms | < 200ms | PASS |
| P99 | 2.55ms | < 500ms | PASS |
| Max | 3.14ms | - | - |
| Status | 200:1500 | - | - |

### Scenario 3: List Projects (50 QPS)

| Metric | Value | SLO | Pass? |
|--------|-------|-----|-------|
| Total requests | 1,500 | - | - |
| Success rate | 100.00% | < 5% errors | PASS |
| Mean latency | 1.84ms | - | - |
| P50 | 1.79ms | - | - |
| P95 | 2.45ms | < 200ms | PASS |
| P99 | 2.76ms | < 500ms | PASS |
| Max | 3.44ms | - | - |
| Status | 200:1500 | - | - |

### Scenario 4: Create Task (50 QPS)

| Metric | Value | SLO | Pass? |
|--------|-------|-----|-------|
| Total requests | 1,500 | - | - |
| Success rate | 100.00% | < 5% errors | PASS |
| Mean latency | 4.54ms | - | - |
| P50 | 4.41ms | - | - |
| P95 | 5.72ms | < 200ms | PASS |
| P99 | 6.22ms | < 500ms | PASS |
| Max | 12.98ms | - | - |
| Status | 201:1500 | - | - |

## Aggregate SLO Summary

| SLO | Target | Scenario | Actual | Status |
|-----|--------|----------|--------|--------|
| Login P99 latency | < 100ms | Login | 221.1ms | **FAIL** |
| Read P95 latency | < 200ms | GetMe | 2.14ms | PASS |
| Read P95 latency | < 200ms | ListProjects | 2.45ms | PASS |
| Write P95 latency | < 200ms | CreateTask | 5.72ms | PASS |
| Read P99 latency | < 500ms | GetMe | 2.55ms | PASS |
| Write P99 latency | < 500ms | CreateTask | 6.22ms | PASS |
| Error rate | < 5% | All scenarios | 0.00% | PASS |

## Resource Usage (during Scenario 4 peak, 50 QPS write)

| Service | CPU (approx) | Memory (RSS) | Notes |
|---------|-------------|-------------|-------|
| api-gateway | 8-12% | ~25MB | Mostly JSON serialization + JWT verify |
| user-service | 2-5% | ~18MB | Idle during task writes (user lookup cached) |
| task-service | 15-20% | ~22MB | DB writes + operation log worker |
| postgres | 10-15% | ~80MB | 50 writes/s + ~100 reads/s |
| redis | 1-2% | ~8MB | Rate limit counter updates |

**Estimated headroom:** At 50 QPS, CPU usage is ~15-20% on task-service. The system can sustain approximately **200-300 QPS** for authenticated reads and **100-150 QPS** for writes before hitting CPU saturation (single-instance). Rate limits (IP=60/s, user=100/s) are the primary bottleneck before CPU becomes constrained.

## Bottleneck Analysis

### 1. Primary Bottleneck: bcrypt in Login Path
- bcrypt cost=10 takes ~50ms per password verification
- Login P99 = 221ms, exceeding the 100ms SLO
- **Mitigation options:**
  - Reduce bcrypt cost to 8 (~12ms, P99 would be ~60ms)
  - Add connection pooling for concurrent login requests
  - Consider rate-limiting login specifically to mask the latency

### 2. Rate Limiting as Throughput Cap
- Default rate limits (auth=5/s, IP=60/s, user=100/s) are intentionally conservative
- At full utilization, a single IP can sustain ~60 authenticated QPS
- With 10 different users from 10 different IPs, the system would scale to ~600 QPS
- **Design note:** These limits are a security feature, not a performance limitation

### 3. DB Write Path
- CreateTask P50 = 4.41ms, dominated by DB INSERT + operation log append
- Operation log uses async worker (batch write), so it doesn't add to request latency
- GORM with pgx driver shows good performance; connection pool (20 max, 5 idle) is adequate at 50 QPS

### 4. No Observed Bottlenecks at Tested Rates
- No connection pool exhaustion
- No Redis command timeouts
- No gRPC deadline exceeded errors
- No goroutine leaks observed (stable ~20-30 goroutines across all services)

## Recommendations

1. **For login SLO:** Either accept bcrypt cost=10 tradeoff (security over latency) or reduce to cost=8 with documented rationale
2. **For higher throughput:** Increase `defaultIPRate` from 60 to 200 if the service is behind a reverse proxy/load balancer that handles DDoS mitigation
3. **For production:** Add horizontal scaling (multiple gateway instances behind a load balancer) to go beyond single-instance limits
4. **Observability:** Jaeger trace backend already configured in docker-compose; DB spans are instrumented via xpgsql plugin
