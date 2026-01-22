/* eslint-disable @typescript-eslint/no-require-imports */
const { connect, StringCodec } = require('nats');
const express = require('express');
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');

const metricsStore = require('../stores/metricsStore');
const detectionsStore = require('../stores/detectionsStore');
const actionsStore = require('../stores/actionsStore');

const sc = StringCodec();

// Load Knowledge proto
const PROTO_PATH = path.join(__dirname, '../../proto/knowledge.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
});
const knowledgeProto = grpc.loadPackageDefinition(packageDefinition).knowledge;

let knowledgeClient = null;

function connectKnowledge() {
    const knowledgeAddr = process.env.KNOWLEDGE_ADDRESS || 'localhost:50053';
    knowledgeClient = new knowledgeProto.KnowledgeService(
        knowledgeAddr,
        grpc.credentials.createInsecure()
    );
    console.log('Knowledge client created for', knowledgeAddr);
}

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
app.use(express.json());

// Existing endpoints
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

// ===== CONFIG ENDPOINTS (proxy to Knowledge) =====

app.get('/config', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    knowledgeClient.GetSystemConfig({}, (err, response) => {
        if (err) {
            console.error('Failed to get config:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Serving system config');
        res.json(response);
    });
});

app.post('/config', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const config = req.body;
    knowledgeClient.SaveSystemConfig({ config }, (err, response) => {
        if (err) {
            console.error('Failed to save config:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Config saved');
        res.json(response);
    });
});

app.get('/status', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    knowledgeClient.GetSystemStatus({}, (err, response) => {
        if (err) {
            console.error('Failed to get status:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Serving system status');
        res.json(response);
    });
});

app.post('/flush', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    knowledgeClient.FlushAllData({}, (err, response) => {
        if (err) {
            console.error('Failed to flush data:', err);
            return res.status(500).json({ error: err.message });
        }
        
        // Also clear local stores
        metricsStore.clear();
        detectionsStore.clear();
        actionsStore.clear();
        
        console.log('All data flushed');
        res.json(response);
    });
});

const PORT = 3001;

connectKnowledge();
startCollector().then(() => {
    app.listen(PORT, () => {
        console.log(`Collector HTTP server running on http://localhost:${PORT}`);
    });
});