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

// Webhook configuration cache
let webhookConfig = null;

function connectKnowledge() {
    const knowledgeAddr = process.env.KNOWLEDGE_ADDRESS || 'localhost:50053';
    knowledgeClient = new knowledgeProto.KnowledgeService(
        knowledgeAddr,
        grpc.credentials.createInsecure()
    );
    console.log('Knowledge client created for', knowledgeAddr);
}

// Fetch webhook config periodically
function fetchWebhookConfig() {
    if (!knowledgeClient) return;
    
    knowledgeClient.getSystemConfig({}, (err, response) => {
        if (err) {
            // Only log if it's not a connection refused error (expected during startup)
            if (!err.message.includes('UNAVAILABLE')) {
                console.error('Failed to fetch webhook config:', err.message);
            }
            return;
        }
        if (response && response.webhook) {
            webhookConfig = response.webhook;
        }
    });
}

// Send webhook notification
async function sendWebhook(eventType, payload) {
    if (!webhookConfig || !webhookConfig.enabled || !webhookConfig.url) {
        return;
    }

    if (!webhookConfig.events || !webhookConfig.events.includes(eventType)) {
        return;
    }

    try {
        const headers = {
            'Content-Type': 'application/json',
        };

        if (webhookConfig.auth_header) {
            headers['Authorization'] = webhookConfig.auth_header;
        }

        // Check if Discord webhook
        const isDiscord = webhookConfig.url.includes('discord.com/api/webhooks');

        let body;
        if (isDiscord) {
            // Discord format
            body = JSON.stringify({
                embeds: [{
                    title: `StartupMonkey: ${eventType}`,
                    description: payload.title || payload.message || JSON.stringify(payload).slice(0, 200),
                    color: eventType.includes('failed') ? 15158332 : eventType.includes('completed') ? 3066993 : 3447003,
                    fields: [
                        { name: 'Database', value: payload.database_id || 'N/A', inline: true },
                        { name: 'Type', value: payload.action_type || payload.detector_name || 'N/A', inline: true },
                        { name: 'Status', value: payload.status || payload.severity || 'N/A', inline: true },
                    ],
                    timestamp: new Date().toISOString(),
                }]
            });
        } else {
            // Generic format
            body = JSON.stringify({
                event: eventType,
                timestamp: new Date().toISOString(),
                data: payload,
            });
        }

        const response = await fetch(webhookConfig.url, {
            method: 'POST',
            headers,
            body,
        });

        if (!response.ok) {
            console.error(`Webhook failed: ${response.status} ${response.statusText}`);
        } else {
            console.log(`Webhook sent: ${eventType}`);
        }
    } catch (err) {
        console.error('Webhook error:', err.message);
    }
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
                    
                    // Send webhook
                    sendWebhook('detection.created', data);
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
                    
                    // Send webhook based on status
                    const status = data.status?.toLowerCase();
                    if (status === 'queued') {
                        sendWebhook('action.queued', data);
                    } else if (status === 'completed') {
                        sendWebhook('action.completed', data);
                    } else if (status === 'failed') {
                        sendWebhook('action.failed', data);
                    } else if (status === 'rolled_back' || status === 'rolledback') {
                        sendWebhook('action.rolledback', data);
                    }
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

    knowledgeClient.getSystemConfig({}, (err, response) => {
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
    knowledgeClient.saveSystemConfig({ config }, (err, response) => {
        if (err) {
            console.error('Failed to save config:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Config saved');
        
        // Refresh webhook config immediately after save
        fetchWebhookConfig();
        
        res.json(response);
    });
});

app.get('/status', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    knowledgeClient.getSystemStatus({}, (err, response) => {
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

    knowledgeClient.flushAllData({}, (err, response) => {
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

app.post('/actions/:id/approve', async (req, res) => {
    const { id } = req.params;
    
    try {
        // Publish approval to NATS for Executor to pick up
        const nc = await connect({ servers: process.env.NATS_URL || 'nats://localhost:4222' });
        const sc = StringCodec();
        
        nc.publish('actions.approve', sc.encode(JSON.stringify({ action_id: id })));
        await nc.flush();
        await nc.close();
        
        console.log('Action approval published:', id);
        res.json({ success: true, message: 'Action approved' });
    } catch (err) {
        console.error('Failed to approve action:', err);
        res.status(500).json({ error: err.message });
    }
});

app.post('/actions/:id/reject', async (req, res) => {
    const { id } = req.params;
    
    try {
        const nc = await connect({ servers: process.env.NATS_URL || 'nats://localhost:4222' });
        const sc = StringCodec();
        
        nc.publish('actions.reject', sc.encode(JSON.stringify({ action_id: id })));
        await nc.flush();
        await nc.close();
        
        console.log('Action rejection published:', id);
        res.json({ success: true, message: 'Action rejected' });
    } catch (err) {
        console.error('Failed to reject action:', err);
        res.status(500).json({ error: err.message });
    }
});

const PORT = 3001;

connectKnowledge();

// Initial webhook config fetch after a short delay (let Knowledge connect first)
setTimeout(fetchWebhookConfig, 2000);

// Refresh webhook config every 30 seconds
setInterval(fetchWebhookConfig, 30000);

startCollector().then(() => {
    app.listen(PORT, () => {
        console.log(`Collector HTTP server running on http://localhost:${PORT}`);
    });
});