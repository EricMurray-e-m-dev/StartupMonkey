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

        const isDiscord = webhookConfig.url.includes('discord.com/api/webhooks');

        let body;
        if (isDiscord) {
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

// ===== HEALTH ENDPOINT =====

app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        metrics: metricsStore.getAll().length,
        detections: detectionsStore.getAll().length,
        actions: actionsStore.getAll().length
    });
});

// ===== METRICS ENDPOINTS =====

app.get('/metrics/latest', (req, res) => {
    const databaseId = req.query.database_id || null;
    const latest = metricsStore.getLatest(databaseId);
    console.log('Serving latest metric for:', databaseId || 'all');
    res.json(latest || {});
});

app.get('/metrics/history', (req, res) => {
    const databaseId = req.query.database_id || null;
    const limit = parseInt(req.query.limit) || 20;
    const history = metricsStore.getHistory(limit, databaseId);
    console.log('Serving metrics history:', history.length, 'for:', databaseId || 'all');
    res.json(history);
});

// ===== DETECTIONS ENDPOINTS =====

app.get('/detections', (req, res) => {
    const databaseId = req.query.database_id || null;
    const all = detectionsStore.getAll(databaseId);
    console.log('Serving detections:', all.length, 'for:', databaseId || 'all');
    res.json(all);
});

// ===== ACTIONS ENDPOINTS =====

app.get('/actions', (req, res) => {
    const databaseId = req.query.database_id || null;
    const all = actionsStore.getAll(databaseId);
    console.log('Serving actions:', all.length, 'for:', databaseId || 'all');
    res.json(all);
});

app.post('/actions/:id/approve', async (req, res) => {
    const { id } = req.params;
    
    try {
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

app.post('/actions/:id/rollback', async (req, res) => {
    const { id } = req.params;
    
    try {
        const nc = await connect({ servers: process.env.NATS_URL || 'nats://localhost:4222' });
        const sc = StringCodec();
        
        nc.publish('actions.rollback', sc.encode(JSON.stringify({ action_id: id })));
        await nc.flush();
        await nc.close();
        
        console.log('Action rollback published:', id);
        res.json({ success: true, message: 'Rollback requested' });
    } catch (err) {
        console.error('Failed to request rollback:', err);
        res.status(500).json({ error: err.message });
    }
});

// ===== DATABASE ENDPOINTS =====

app.get('/databases', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const enabledOnly = req.query.enabled_only === 'true';
    
    knowledgeClient.listDatabases({ enabled_only: enabledOnly }, (err, response) => {
        if (err) {
            console.error('Failed to list databases:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Serving databases:', response.databases?.length || 0);
        res.json(response.databases || []);
    });
});

app.get('/databases/:id', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const { id } = req.params;

    knowledgeClient.getDatabase({ database_id: id }, (err, response) => {
        if (err) {
            console.error('Failed to get database:', err);
            return res.status(500).json({ error: err.message });
        }
        if (!response.success) {
            return res.status(404).json({ error: response.message || 'Database not found' });
        }
        console.log('Serving database:', id);
        res.json(response);
    });
});

app.post('/databases', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const db = req.body;
    
    // Generate ID from name if not provided
    if (!db.database_id && db.database_name) {
        db.database_id = db.database_name.toLowerCase().replace(/[^a-z0-9]/g, '_');
    }
    
    // Parse connection string for host/port
    let host = 'localhost';
    let port = 5432;
    
    try {
        const url = new URL(db.connection_string);
        host = url.hostname || 'localhost';
        port = parseInt(url.port) || 5432;
    } catch (e) {
        // Keep defaults if parse fails
    }

    const request = {
        database_id: db.database_id,
        connection_string: db.connection_string,
        database_type: db.database_type || 'postgres',
        database_name: db.database_name,
        host: host,
        port: port,
        version: 'unknown',
        registered_at: Math.floor(Date.now() / 1000),
        enabled: db.enabled !== false,
        metadata: {}
    };

    knowledgeClient.registerDatabase(request, (err, response) => {
        if (err) {
            console.error('Failed to register database:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Database registered:', db.database_id);
        res.json(response);
    });
});

app.put('/databases/:id', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const { id } = req.params;
    const updates = req.body;

    const request = {
        database_id: id,
        connection_string: updates.connection_string,
        database_name: updates.database_name,
        enabled: updates.enabled
    };

    knowledgeClient.updateDatabase(request, (err, response) => {
        if (err) {
            console.error('Failed to update database:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Database updated:', id);
        res.json(response);
    });
});

app.delete('/databases/:id', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    const { id } = req.params;

    knowledgeClient.unregisterDatabase({ database_id: id }, (err, response) => {
        if (err) {
            console.error('Failed to unregister database:', err);
            return res.status(500).json({ error: err.message });
        }
        console.log('Database unregistered:', id);
        res.json(response);
    });
});

// ===== CONFIG ENDPOINTS =====

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
        
        fetchWebhookConfig();
        
        res.json(response);
    });
});

// ===== STATUS ENDPOINT =====

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

// ===== FLUSH ENDPOINT =====

app.post('/flush', (req, res) => {
    if (!knowledgeClient) {
        return res.status(503).json({ error: 'Knowledge service not connected' });
    }

    knowledgeClient.flushAllData({}, (err, response) => {
        if (err) {
            console.error('Failed to flush data:', err);
            return res.status(500).json({ error: err.message });
        }
        
        metricsStore.clear();
        detectionsStore.clear();
        actionsStore.clear();
        
        console.log('All data flushed');
        res.json(response);
    });
});

// ===== START SERVER =====

const PORT = process.env.COLLECTOR_PORT || 3001;

connectKnowledge();

setTimeout(fetchWebhookConfig, 2000);
setInterval(fetchWebhookConfig, 30000);

startCollector().then(() => {
    app.listen(PORT, () => {
        console.log(`Dashboard BFF running on http://localhost:${PORT}`);
    });
});