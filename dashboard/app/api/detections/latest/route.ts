import { NextResponse } from "next/server";
import { connect } from "nats";
import { Detection } from "@/types/detection";

const detectionHistory: Detection[] = [];
const MAX_HISTORY = 50;

export async function GET() {
    try {
        // TODO: replace hardcoded url
        const nc = await connect({servers: 'nats://localhost:4222'});

        const sub = nc.subscribe('detections');

        const collectDetections = new Promise<void>((resolve) => {
            setTimeout(async () => {
                await sub.drain();
                await nc.close();
                resolve();
            }, 2000);
        });

        (async() => {
            for await (const msg of sub) {
                const detection: Detection  = JSON.parse(new TextDecoder().decode(msg.data));

                const exists = detectionHistory.some(d => d.id === detection.id);
                if(!exists) {
                    detectionHistory.unshift(detection);
                    if (detectionHistory.length > MAX_HISTORY) {
                        detectionHistory.pop();
                    }
                    console.log("Stored detection: ", detection.title)
                }
            }
        })();

        await collectDetections;

        return NextResponse.json(detectionHistory);

    } catch (error) {
        console.error("failed to fetch detections from NATS:", error)
        return NextResponse.json(detectionHistory)
    }
}