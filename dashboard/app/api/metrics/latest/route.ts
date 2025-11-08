import { NextResponse } from "next/server";
import { connect } from "nats";

export const dynamic = 'force-dynamic';

export async function GET() {
    try {
        // TODO: replace hardcoded url
        const nc = await connect({servers: 'nats://localhost:4222'});

        const sub = nc.subscribe('metrics', { max: 1 }); // One snapshot at a time

        const done = (async() => {
            for await (const msg of sub) {
                const data  = JSON.parse(new TextDecoder().decode(msg.data));
                await nc.close();
                return data;
            }
        })();

        const timeout = new Promise(resolve =>
            setTimeout(() => resolve(null), 10000)
        );

        const metrics = await Promise.race([done, timeout]);

        return NextResponse.json(metrics ?? {});

    } catch (error) {
        console.error("failed to fetch metrics from NATS:", error)
        return NextResponse.json({})
    }
}