const { connect, StringCodec } = require('nats')
const metricsStore = require('../stores/metricsStore');
const detectionsStore = require('../stores/detectionsStore');
const actionsStore = require('../stores/actionsStore');

const sc = StringCodec();

async function startCollector() {
    try {
        const natsURL = process.env.NATS_URL || 'nats://localhost:4222';
        const nc = await connect({ servers: natsURL });

        const metricsSub = nc.subscribe('metrics');
        (async () => {
            for await (const msg of metricsSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    metricsStore.add(data);
                    console.log("Stored metric: ", data.database_id);
                } catch (error) {
                    console.error("Error processing metric: ", error);
                }
            }
        })();

        const detectionsSub = nc.subscribe('detections');
        (async () => {
            for await (const msg of detectionsSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    detectionsStore.add(data);
                    console.log("Stored detection: ", data.title);
                } catch (error) {
                    console.error("Error processing detection: ", error);
                }
            }
        })();

        const actionSub = nc.subscribe('actions.status');
        (async () => {
            for await (const msg of actionSub) {
                try {
                    const data = JSON.parse(sc.decode(msg.data));
                    actionsStore.add(data);
                    console.log("Stored action: ", data.action_id, data.status);
                } catch (error) {
                    console.error("Error processing action: ", error);
                }
            }
        })();


        console.log("Subscribed to all topics")
    } catch (error) {
        console.error("failed to connect to NATS: ", error);
        process.exit(1);
    }
}

startCollector();