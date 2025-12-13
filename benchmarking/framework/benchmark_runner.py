#!/usr/bin/env python3
"""
Benchmark Runner - Orchestrates load testing with metrics collection.

Uses Locust CSV output (headless-safe, deterministic).
"""

import argparse
import subprocess
import time
import sys
import os
from datetime import datetime
from typing import List
import logging

from database import BenchmarkDatabase
from locust_collector import LocustCollector
from collector_client import CollectorClientSync

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class BenchmarkRunner:
    """Orchestrates benchmark runs with Locust and metrics collection."""

    def __init__(self, app_name: str, stage: int, target_host: str,
                 locust_file: str, db: BenchmarkDatabase):
        self.app_name = app_name
        self.stage = stage
        self.target_host = target_host
        self.locust_file = locust_file
        self.db = db

        # CSV-based collector
        self.locust_collector = LocustCollector()

        self.locust_process = None
        self.locust_log_file = None

        self.output_dir = "outputs"
        os.makedirs(self.output_dir, exist_ok=True)

    def generate_run_id(self, user_count: int) -> str:
        timestamp = datetime.utcnow().strftime('%Y%m%d_%H%M%S')
        return f"{self.app_name}_stage{self.stage}_u{user_count}_{timestamp}"

    def start_locust(
        self,
        run_id: str,
        user_count: int,
        spawn_rate: int = 5,
        duration: int = 120
    ) -> bool:
        """
        Start Locust in headless mode with CSV output.
        """
        csv_prefix = os.path.join(self.output_dir, run_id)
        log_path = os.path.join(self.output_dir, f"{run_id}.locust.log")

        cmd = [
            "locust",
            "-f", self.locust_file,
            "--host", self.target_host,
            "--users", str(user_count),
            "--spawn-rate", str(spawn_rate),
            "--run-time", f"{duration}s",
            "--headless",
            "--csv", csv_prefix,
            "--csv-full-history",
            "--only-summary"
        ]

        logger.info(f"Starting Locust: {user_count} users for {duration}s")
        logger.info(f"Command: {' '.join(cmd)}")

        try:
            self.locust_log_file = open(log_path, "w", encoding="utf-8")

            self.locust_process = subprocess.Popen(
                cmd,
                stdout=self.locust_log_file,
                stderr=subprocess.STDOUT,
                text=True
            )

            time.sleep(3)

            if self.locust_process.poll() is not None:
                logger.error(
                    f"Locust exited immediately with code "
                    f"{self.locust_process.returncode}"
                )
                return False

            return True

        except Exception as e:
            logger.error(f"Failed to start Locust: {e}", exc_info=True)
            return False

    def wait_for_locust_completion(self, timeout: int) -> bool:
        """
        Wait for Locust process to finish cleanly.
        """
        if not self.locust_process:
            return False

        try:
            self.locust_process.wait(timeout=timeout)
            return self.locust_process.returncode == 0
        except subprocess.TimeoutExpired:
            logger.error("Locust did not terminate within timeout")
            return False
        finally:
            if self.locust_log_file:
                self.locust_log_file.close()
                self.locust_log_file = None

    def stop_locust(self):
        if self.locust_process:
            logger.warning("Force stopping Locust...")
            self.locust_process.terminate()
            try:
                self.locust_process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.locust_process.kill()
            self.locust_process = None

    def run_single_test(
        self,
        user_count: int,
        duration: int = 120,
        cooldown: int = 30
    ) -> bool:
        run_id = self.generate_run_id(user_count)
        csv_prefix = os.path.join(self.output_dir, run_id)

        logger.info("=" * 80)
        logger.info(f"Starting benchmark run: {run_id}")
        logger.info("=" * 80)

        try:
            # DB record
            self.db.create_benchmark_run(
                run_id=run_id,
                app_name=self.app_name,
                stage=self.stage,
                user_count=user_count,
                duration_seconds=duration,
                target_host=self.target_host,
                notes="Automated benchmark run (CSV-based)"
            )

            # Start Collector (NATS)
            with CollectorClientSync() as collector:
                collector.start_collecting()
                time.sleep(2)

                # Start Locust
                if not self.start_locust(run_id, user_count, duration=duration):
                    return False

                # Keep NATS event loop alive
                start = time.time()
                while self.locust_process.poll() is None:
                    collector.keep_alive(1.0)
                    if time.time() - start > duration + 60:
                        logger.warning("Safety timeout reached")
                        break

                if not self.wait_for_locust_completion(timeout=30):
                    self.stop_locust()
                    return False

                collector.stop_collecting()

                # Give filesystem time to flush CSVs
                time.sleep(2)

                # ---- LOCUST METRICS (CSV) ----
                logger.info("Parsing Locust CSV metrics...")
                locust_metrics = self.locust_collector.extract_metrics(csv_prefix)

                if locust_metrics:
                    self.db.insert_locust_metrics(run_id, locust_metrics)
                else:
                    logger.warning("No Locust metrics found")

                # ---- COLLECTOR METRICS ----
                db_metrics = collector.get_aggregated_metrics()
                if db_metrics:
                    self.db.insert_collector_metrics(run_id, db_metrics)

                # ---- SUMMARY ----
                self.db.create_run_summary(run_id)

                logger.info(f"Run {run_id} completed successfully")

                if cooldown > 0:
                    logger.info(f"Cooldown: {cooldown}s")
                    time.sleep(cooldown)

                return True

        except Exception as e:
            logger.error(f"Benchmark failed: {e}", exc_info=True)
            self.stop_locust()
            return False

    def run_benchmark_suite(
        self,
        user_counts: List[int],
        duration: int = 120,
        cooldown: int = 30
    ) -> bool:
        success = 0

        for idx, users in enumerate(user_counts, 1):
            logger.info(f"\n>>> Test {idx}/{len(user_counts)}: {users} users")
            if self.run_single_test(users, duration, cooldown):
                success += 1
            else:
                logger.error(f"Test failed for {users} users")

        logger.info("=" * 80)
        logger.info(f"Completed {success}/{len(user_counts)} benchmark runs")
        logger.info("=" * 80)

        return success == len(user_counts)


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--app", required=True)
    parser.add_argument("--stage", type=int, choices=[0, 1, 2], required=True)
    parser.add_argument("--users", required=True)
    parser.add_argument("--duration", type=int, default=120)
    parser.add_argument("--cooldown", type=int, default=30)
    parser.add_argument("--target", default="http://localhost:3020")
    parser.add_argument("--locust-file")
    return parser.parse_args()


def main():
    args = parse_args()

    try:
        user_counts = [int(u.strip()) for u in args.users.split(",")]
    except ValueError:
        logger.error("Invalid --users format")
        sys.exit(1)

    locust_file = args.locust_file or os.path.join(args.app, "locustfile.py")
    if not os.path.exists(locust_file):
        logger.error(f"Locust file not found: {locust_file}")
        sys.exit(1)

    db = BenchmarkDatabase()
    db.connect()

    runner = BenchmarkRunner(
        app_name=args.app,
        stage=args.stage,
        target_host=args.target,
        locust_file=locust_file,
        db=db
    )

    success = runner.run_benchmark_suite(
        user_counts=user_counts,
        duration=args.duration,
        cooldown=args.cooldown
    )

    db.close()
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
