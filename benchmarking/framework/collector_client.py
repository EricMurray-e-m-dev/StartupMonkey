"""
Collector metrics client.
Subscribes to NATS metrics during load tests and aggregates results.
"""

import nats
import json
import asyncio
from typing import Dict, List, Optional
from datetime import datetime
import logging
import time

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class CollectorClient:
    """Collects database metrics from StartupMonkey Collector via NATS."""
    
    def __init__(self, nats_url: str = "nats://localhost:4222"):
        """
        Initialize Collector client.
        
        Args:
            nats_url: NATS server URL
        """
        self.nats_url = nats_url
        self.nc = None
        self.metrics_buffer = []
        self.subscription = None
    
    async def connect(self):
        """Connect to NATS."""
        try:
            self.nc = await nats.connect(self.nats_url)
            logger.info(f"Connected to NATS at {self.nats_url}")
        except Exception as e:
            logger.error(f"Failed to connect to NATS: {e}")
            raise
    
    async def disconnect(self):
        """Disconnect from NATS."""
        if self.subscription:
            await self.subscription.unsubscribe()
        
        if self.nc:
            await self.nc.close()
            logger.info("Disconnected from NATS")
    
    async def start_collecting(self):
        """
        Start collecting metrics from NATS.
        Subscribes to metrics topic and buffers messages.
        """
        if not self.nc:
            await self.connect()
        
        self.metrics_buffer = []  # Clear buffer
        
        async def message_handler(msg):
            try:
                data = json.loads(msg.data.decode())
                data['_received_at'] = datetime.utcnow().isoformat()
                self.metrics_buffer.append(data)
                logger.info(f"Received metric sample - Total: {len(self.metrics_buffer)}")
            except Exception as e:
                logger.error(f"Error handling metric message: {e}")
        
        # Subscribe to metrics topic
        self.subscription = await self.nc.subscribe(
            "metrics",
            cb=message_handler
        )
        
        logger.info("Started collecting metrics from NATS")
    
    async def stop_collecting(self):
        """
        Stop collecting metrics.
        Unsubscribes from NATS topic.
        """
        if self.subscription:
            await self.subscription.unsubscribe()
            self.subscription = None
        
        logger.info(f"Stopped collecting metrics. Collected {len(self.metrics_buffer)} samples")
    
    def get_collected_metrics(self) -> List[Dict]:
        """
        Get all collected metrics.
        
        Returns:
            List of metric dictionaries
        """
        return self.metrics_buffer.copy()
    
    def aggregate_metrics(self) -> Optional[Dict]:
        """
        Aggregate collected metrics into summary.
        
        Returns:
            Dict with aggregated database metrics
        """
        if not self.metrics_buffer:
            logger.warning("No metrics to aggregate")
            return None
        
        logger.info(f"Aggregating {len(self.metrics_buffer)} metric samples")
        
        # Extract relevant fields from Collector's actual format
        connection_samples = []
        cache_samples = []
        scan_samples = []
        health_samples = []
        
        for metric in self.metrics_buffer:
            # Parse measurements
            measurements = metric.get('measurements', {})
            
            if measurements:
                connection_samples.append({
                    'active': measurements.get('active_connections', 0),
                    'max': measurements.get('max_connections', 100)
                })
                
                cache_rate = measurements.get('cache_hit_rate', 0)
                if cache_rate > 0:
                    cache_samples.append(cache_rate)
                
                scan_samples.append({
                    'sequential': measurements.get('sequential_scans', 0),
                    'index': 0  # Not in measurements
                })
            
            # Parse health scores
            connection_health = metric.get('connection_health', 0)
            query_health = metric.get('query_health', 0)
            cache_health = metric.get('cache_health', 0)
            overall_health = metric.get('health_score', 0)
            
            if any([connection_health, query_health, cache_health, overall_health]):
                health_samples.append({
                    'query': query_health,
                    'connection': connection_health,
                    'cache': cache_health,
                    'overall': overall_health
                })
        
        # Calculate averages
        aggregated = {}
        
        if connection_samples:
            aggregated['active_connections'] = int(
                sum(s['active'] for s in connection_samples) / len(connection_samples)
            )
            aggregated['max_connections'] = connection_samples[0]['max']
        
        if cache_samples:
            aggregated['cache_hit_rate'] = round(
                sum(cache_samples) / len(cache_samples), 4
            )
        
        if scan_samples:
            aggregated['sequential_scans'] = int(
                sum(s['sequential'] for s in scan_samples) / len(scan_samples)
            )
            aggregated['index_scans'] = 0  # Not available in current format
        
        if health_samples:
            aggregated['query_health'] = round(
                sum(s['query'] for s in health_samples) / len(health_samples), 2
            )
            aggregated['connection_health'] = round(
                sum(s['connection'] for s in health_samples) / len(health_samples), 2
            )
            aggregated['cache_health'] = round(
                sum(s['cache'] for s in health_samples) / len(health_samples), 2
            )
            aggregated['overall_health'] = round(
                sum(s['overall'] for s in health_samples) / len(health_samples), 2
            )
        
        logger.info(f"Aggregation complete: {aggregated}")
        return aggregated


# Synchronous wrapper that actually works
class CollectorClientSync:
    """Synchronous wrapper around CollectorClient that keeps event loop alive."""
    
    def __init__(self, nats_url: str = "nats://localhost:4222"):
        self.nats_url = nats_url
        self.client = CollectorClient(nats_url)
        self.loop = None
    
    def __enter__(self):
        """Context manager entry."""
        # Create new event loop for this context
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)
        
        # Connect to NATS
        self.loop.run_until_complete(self.client.connect())
        
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        # Disconnect from NATS
        self.loop.run_until_complete(self.client.disconnect())
        
        # Close loop
        self.loop.close()
    
    def start_collecting(self):
        """Start collecting metrics."""
        self.loop.run_until_complete(self.client.start_collecting())
    
    def stop_collecting(self):
        """Stop collecting metrics."""
        self.loop.run_until_complete(self.client.stop_collecting())
    
    def keep_alive(self, duration: float):
        """
        Keep event loop running to process NATS callbacks.
        Call this periodically during test execution.
        
        Args:
            duration: How long to run loop in seconds
        """
        async def sleep_and_process():
            await asyncio.sleep(duration)
        
        self.loop.run_until_complete(sleep_and_process())
    
    def get_aggregated_metrics(self) -> Optional[Dict]:
        """Get aggregated metrics."""
        return self.client.aggregate_metrics()


if __name__ == '__main__':
    # Test the collector client
    print("Testing Collector Client...")
    print("Make sure StartupMonkey Collector is running and publishing to NATS!")
    
    with CollectorClientSync() as client:
        print("\nStarting metric collection...")
        client.start_collecting()
        
        print("Collecting for 30 seconds (keeping event loop alive)...")
        # THIS IS THE KEY - keep event loop running so callbacks fire
        for i in range(30):
            client.keep_alive(1.0)
            
        
        print("\nStopping collection...")
        client.stop_collecting()
        
        print("\nAggregated metrics:")
        metrics = client.get_aggregated_metrics()
        if metrics:
            for key, value in metrics.items():
                print(f"  {key}: {value}")
        else:
            print("  No metrics collected - check if Collector is running!")