"""
Locust metrics collector (CSV-based).

Parses Locust CSV output generated in headless mode.
"""

import csv
import os
import logging
from typing import Dict, List

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class LocustCollector:
    """
    Collects Locust metrics from CSV files produced by headless runs.
    """

    def extract_metrics(self, csv_prefix: str) -> List[Dict]:
        """
        Extract per-endpoint metrics from Locust CSV output.

        Args:
            csv_prefix: Prefix path used with Locust --csv flag

        Returns:
            List of metrics dictionaries (one per endpoint)
        """
        stats_path = f"{csv_prefix}_stats.csv"

        if not os.path.exists(stats_path):
            logger.error(f"Locust stats file not found: {stats_path}")
            return []

        metrics: List[Dict] = []

        with open(stats_path, newline="", encoding="utf-8") as csvfile:
            reader = csv.DictReader(csvfile)

            for row in reader:
                # Skip aggregated row
                if row.get("Name") == "Aggregated":
                    continue

                # Locust stores method + path in Name or separate column
                name = row.get("Name", "").strip()
                method = row.get("Method", "GET").strip()

                # Split "GET /path" if needed
                if " " in name:
                    method, endpoint = name.split(" ", 1)
                else:
                    endpoint = name

                try:
                    metric = {
                        "endpoint": endpoint,
                        "method": method,
                        "request_count": int(row.get("Request Count", 0)),
                        "failure_count": int(row.get("Failure Count", 0)),
                        "median_response_time": float(row.get("Median Response Time", 0)),
                        "average_response_time": float(row.get("Average Response Time", 0)),
                        "min_response_time": float(row.get("Min Response Time", 0)),
                        "max_response_time": float(row.get("Max Response Time", 0)),
                        "p50_response_time": float(row.get("50%", row.get("Median Response Time", 0))),
                        "p95_response_time": float(row.get("95%", 0)),
                        "p99_response_time": float(row.get("99%", 0)),
                        "requests_per_second": float(row.get("Requests/s", 0)),
                        "failures_per_second": float(row.get("Failures/s", 0)),
                    }

                    metrics.append(metric)

                except ValueError as e:
                    logger.warning(f"Failed to parse row {row}: {e}")

        logger.info(f"Extracted Locust metrics for {len(metrics)} endpoints")
        return metrics
