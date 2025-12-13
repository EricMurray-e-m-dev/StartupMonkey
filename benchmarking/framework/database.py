"""
Database operations for benchmarking results.
Handles PostgreSQL connection and data storage.
"""

import psycopg2
from psycopg2.extras import execute_values
from typing import Dict, List, Optional
import os
from datetime import datetime


class BenchmarkDatabase:
    def __init__(self, db_config: Optional[Dict] = None):
        """
        Initialize database connection.
        
        Args:
            db_config: Database configuration dict with keys:
                       host, port, database, user, password
        """
        if db_config is None:
            # Default to localhost PostgreSQL
            db_config = {
                'host': os.getenv('BENCHMARK_DB_HOST', 'localhost'),
                'port': int(os.getenv('BENCHMARK_DB_PORT', 5432)),
                'database': os.getenv('BENCHMARK_DB_NAME', 'startupmonkey_benchmarks'),
                'user': os.getenv('BENCHMARK_DB_USER', 'ericmurray'),
                'password': os.getenv('BENCHMARK_DB_PASSWORD', '')
            }
        
        self.db_config = db_config
        self.conn = None
    
    def connect(self):
        """Establish database connection."""
        self.conn = psycopg2.connect(**self.db_config)
    
    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()
    
    def __enter__(self):
        """Context manager entry."""
        self.connect()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()
    
    def initialize_schema(self, schema_file: str = 'framework/schema.sql'):
        """
        Initialize database schema from SQL file.
        
        Args:
            schema_file: Path to schema.sql file
        """
        with open(schema_file, 'r') as f:
            schema_sql = f.read()
        
        with self.conn.cursor() as cur:
            cur.execute(schema_sql)
            self.conn.commit()
    
    def create_benchmark_run(self, run_id: str, app_name: str, stage: int,
                            user_count: int, duration_seconds: int,
                            target_host: str, notes: Optional[str] = None) -> str:
        """
        Create a new benchmark run record.
        
        Returns:
            run_id of the created run
        """
        query = """
        INSERT INTO benchmark_runs 
        (run_id, app_name, stage, user_count, duration_seconds, target_host, notes)
        VALUES (%s, %s, %s, %s, %s, %s, %s)
        RETURNING run_id
        """
        
        with self.conn.cursor() as cur:
            cur.execute(query, (run_id, app_name, stage, user_count, 
                               duration_seconds, target_host, notes))
            self.conn.commit()
            return cur.fetchone()[0]
    
    def insert_locust_metrics(self, run_id: str, metrics: List[Dict]):
        """
        Insert Locust metrics for multiple endpoints.
        
        Args:
            run_id: Benchmark run identifier
            metrics: List of metric dicts with keys:
                    endpoint, method, request_count, failure_count, etc.
        """
        query = """
        INSERT INTO locust_metrics (
            run_id, endpoint, method, request_count, failure_count,
            median_response_time, average_response_time,
            min_response_time, max_response_time,
            p50_response_time, p95_response_time, p99_response_time,
            requests_per_second, failures_per_second
        ) VALUES %s
        """
        
        values = [
            (
                run_id,
                m['endpoint'],
                m['method'],
                m['request_count'],
                m['failure_count'],
                m['median_response_time'],
                m['average_response_time'],
                m['min_response_time'],
                m['max_response_time'],
                m['p50_response_time'],
                m['p95_response_time'],
                m['p99_response_time'],
                m['requests_per_second'],
                m['failures_per_second']
            )
            for m in metrics
        ]
        
        with self.conn.cursor() as cur:
            execute_values(cur, query, values)
            self.conn.commit()
    
    def insert_collector_metrics(self, run_id: str, metrics: Dict):
        """
        Insert Collector metrics (database health).
        
        Args:
            run_id: Benchmark run identifier
            metrics: Dict with database health metrics
        """
        query = """
        INSERT INTO collector_metrics (
            run_id, active_connections, max_connections,
            cache_hit_rate, sequential_scans, index_scans,
            query_health, connection_health, cache_health, overall_health
        ) VALUES (
            %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
        )
        """
        
        with self.conn.cursor() as cur:
            cur.execute(query, (
                run_id,
                metrics.get('active_connections'),
                metrics.get('max_connections'),
                metrics.get('cache_hit_rate'),
                metrics.get('sequential_scans'),
                metrics.get('index_scans'),
                metrics.get('query_health'),
                metrics.get('connection_health'),
                metrics.get('cache_health'),
                metrics.get('overall_health')
            ))
            self.conn.commit()
    
    def create_run_summary(self, run_id: str):
        """
        Create aggregated summary for a run.
        Automatically calculates totals from locust_metrics and collector_metrics.
        """
        query = """
        INSERT INTO run_summary (
            run_id, total_requests, total_failures, failure_rate,
            avg_response_time, p50_response_time, p95_response_time, p99_response_time,
            requests_per_second, db_cache_hit_rate, db_overall_health
        )
        SELECT 
            %s,
            SUM(lm.request_count),
            SUM(lm.failure_count),
            ROUND((SUM(lm.failure_count)::NUMERIC / NULLIF(SUM(lm.request_count), 0) * 100), 2),
            AVG(lm.average_response_time),
            AVG(lm.p50_response_time),
            AVG(lm.p95_response_time),
            AVG(lm.p99_response_time),
            SUM(lm.requests_per_second),
            AVG(cm.cache_hit_rate),
            AVG(cm.overall_health)
        FROM locust_metrics lm
        LEFT JOIN collector_metrics cm ON lm.run_id = cm.run_id
        WHERE lm.run_id = %s
        """
        
        with self.conn.cursor() as cur:
            cur.execute(query, (run_id, run_id))
            self.conn.commit()
    
    def get_stage_comparison(self, app_name: str) -> List[Dict]:
        """
        Get comparison data across all stages for an app.
        
        Returns:
            List of dicts with stage comparison metrics
        """
        query = """
        SELECT * FROM stage_comparison
        WHERE app_name = %s
        ORDER BY stage, user_count
        """
        
        with self.conn.cursor() as cur:
            cur.execute(query, (app_name,))
            columns = [desc[0] for desc in cur.description]
            return [dict(zip(columns, row)) for row in cur.fetchall()]
    
    def get_endpoint_comparison(self, app_name: str, endpoint: str) -> List[Dict]:
        """
        Get comparison data for a specific endpoint across stages.
        
        Returns:
            List of dicts with endpoint metrics
        """
        query = """
        SELECT * FROM endpoint_comparison
        WHERE app_name = %s AND endpoint = %s
        ORDER BY stage, user_count
        """
        
        with self.conn.cursor() as cur:
            cur.execute(query, (app_name, endpoint))
            columns = [desc[0] for desc in cur.description]
            return [dict(zip(columns, row)) for row in cur.fetchall()]