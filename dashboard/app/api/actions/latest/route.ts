import { NextResponse } from "next/server";
import { connect } from "nats";
import { ActionResult } from "@/types/actions";

const actionHistory: ActionResult[] = [];
const MAX_HISTORY = 50;

export async function GET() {
    try {
        console.log("Fetching actions from NATS")

        // TODO: remove hardcoded nats URL
        const nc = await connect({ servers: 'nats://localhost:4222'})
        const sub = nc.subscribe('actions.status');

        const collectActions = new Promise<void>((resolve) => {
            setTimeout(async () => {
                await sub.drain();
                await nc.close();
                resolve();
            }, 2000);
        });

        (async () => {
            for await (const msg of sub) {
                const action: ActionResult = JSON.parse(new TextDecoder().decode(msg.data));

                const existingIndex = actionHistory.findIndex(a => a.action_id === action.action_id);

                if (existingIndex === 0) {
                    actionHistory[existingIndex] = action;
                } else {
                    actionHistory.unshift(action);

                    if (actionHistory.length > MAX_HISTORY) {
                        actionHistory.pop();
                    }
                }

                console.log("Stored Action: ", action.action_id, action.status);
            }
        })();

        await collectActions

        console.log(`API Returning ${actionHistory.length} actions`);
        return NextResponse.json(actionHistory);
    } catch (error) {
        console.error("Failed to fetch actions:", error)
        return NextResponse.json(actionHistory);
    }
}