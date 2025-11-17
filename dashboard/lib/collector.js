const { connect, StringCodec } = require('nats');
const express = require('express');
const metricsStore = require('../stores/metricsStore');
const detectionsStore = require('../stores/detectionsStore');
const actionsStore = require('../stores/actionsStore');

const sc = StringCodec();

async function startCollector() {
    try {
        const natsUrl = process.env.NATS_URL || 'nats://localhost:4222';
        const nc = await connect({ servers: natsUrl });
        
        console.log('Dashboard Collector connected to NATS at', natsUrl);

        // Subscribe to metrics
        const metricsSub = nc.subscribe('metrics');
        (async () => {
            for await (const msg of metricsSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    metricsStore.add(data);
                    console.log('Stored metric:', data.DatabaseID || data.database_id);
                } catch (err) {
                    console.error('Error processing metric:', err);
                }
            }
        })();

        // Subscribe to detections
        const detectionsSub = nc.subscribe('detections');
        (async () => {
            for await (const msg of detectionsSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    detectionsStore.add(data);
                    console.log('Stored detection:', data.title);
                } catch (err) {
                    console.error('Error processing detection:', err);
                }
            }
        })();

        // Subscribe to actions
        const actionsSub = nc.subscribe('actions.status');
        (async () => {
            for await (const msg of actionsSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    actionsStore.addOrUpdate(data);
                    console.log('Stored action:', data.action_id, data.status);
                } catch (err) {
                    console.error('Error processing action:', err);
                }
            }
        })();

        console.log('Subscribed to all topics');
        
    } catch (err) {
        console.error('Failed to connect to NATS:', err);
        process.exit(1);
    }
}

const app = express();

app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        metrics: metricsStore.getAll().length,
        detections: detectionsStore.getAll().length,
        actions: actionsStore.getAll().length
    });
});

app.get('/metrics/latest', (req, res) => {
    const latest = metricsStore.getLatest();
    console.log('Serving latest metric');
    res.json(latest || {});
});

app.get('/detections', (req, res) => {
    const all = detectionsStore.getAll();
    console.log('Serving detections:', all.length);
    res.json(all);
});

app.get('/actions', (req, res) => {
    const all = actionsStore.getAll();
    console.log('Serving actions:', all.length);
    res.json(all);
});

const PORT = 3001;

startCollector().then(() => {
    app.listen(PORT, () => {
        console.log(`Collector HTTP server running on http://localhost:${PORT}`);
    });
});