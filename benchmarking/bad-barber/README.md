# Bad Barbers - Load Testing Configuration

## Overview

Load testing configuration for the Bad Barbers barbershop booking application. This app intentionally contains performance issues that StartupMonkey will detect and fix.

## User Profiles

### BadBarbersUser (Default)
Realistic user behavior:
- Views homepage
- Browses barbers/services
- Checks available slots
- Creates bookings
- Searches past bookings

**Use for:** Stage comparisons, realistic load testing

### AggressiveUser
Hammers slow endpoints repeatedly:
- Heavy load on `popular_barbers`
- Constant booking searches
- Rapid-fire slot checks

**Use for:** Quickly triggering detections, stress testing

### ReadOnlyUser
Only performs GET requests:
- No database writes
- Measures pure query performance

**Use for:** Baseline read performance testing

## Running Tests

### Full Benchmark Suite (All Stages)
```bash
# Stage 0: Baseline
python framework/benchmark_runner.py \
  --app bad-barbers \
  --stage 0 \
  --users 1,10,25,50,100

# Stage 1: After StartupMonkey optimizations
python framework/benchmark_runner.py \
  --app bad-barbers \
  --stage 1 \
  --users 1,10,25,50,100

# Stage 2: After all optimizations
python framework/benchmark_runner.py \
  --app bad-barbers \
  --stage 2 \
  --users 1,10,25,50,100
```

### Quick Test (Single User Count)
```bash
python framework/benchmark_runner.py \
  --app bad-barbers \
  --stage 0 \
  --users 50 \
  --duration 60
```

### Manual Locust (With Web UI)
```bash
locust -f bad-barbers/locustfile.py --host http://localhost:8080
# Open browser to http://localhost:8089
```

## Expected Performance

### Stage 0 (Baseline)
- P95 latency: 800-1500ms
- Failure rate: 2-5%
- RPS: 40-60

### Stage 1 (Safe Auto)
- P95 latency: 200-400ms (50-75% improvement)
- Failure rate: <1%
- RPS: 80-120

### Stage 2 (All Optimizations)
- P95 latency: 50-150ms (90% improvement)
- Failure rate: <0.1%
- RPS: 150-200

## Endpoints Under Test

| Endpoint | Issue | Detection |
|----------|-------|-----------|
| `/api/popular-barbers` | Expensive aggregation | Cache miss |
| `/api/bookings/search` | No index on phone | Missing index |
| `/api/bookings/today` | No index on date | Missing index |
| `/api/barbers/[id]/available-slots` | Connection exhaustion | Connection pool |
| All endpoints | New Client per request | Connection pool |

## Troubleshooting

**High failure rates at low user counts:**
- Check if bad-barbers app is running
- Verify database is seeded with data
- Check database connection limits

**No Collector metrics:**
- Ensure StartupMonkey is running
- Check NATS is accessible
- Verify Collector is publishing to NATS

**Locust won't start:**
- Check port 8089 is free
- Verify locustfile.py syntax
- Check target host is reachable