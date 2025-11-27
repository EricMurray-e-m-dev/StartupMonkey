import { NextResponse, NextRequest } from "next/server";

export const dynamic = 'force-dynamic';

const EXECUTOR_URL = process.env.EXECUTOR_HTTP_URL || 'http://localhost:8084';

export async function POST(request: NextRequest) {
    try {
        const body = await request.json();
        const { action_id } = body;

        if (!action_id) {
            return NextResponse.json(
                { error: 'action_id is required.'},
                { status: 400 }
            );
        }

        console.log("Requesting rollback for action: " + action_id)

        const response = await fetch(`${EXECUTOR_URL}/api/actions/${action_id}/rollback`, {
            method: 'POST',
        });

        if (!response.ok) {
            const errorText = await response.text();
            console.error("Rollback failed: " + errorText)
            return NextResponse.json(
                { error: errorText || 'Rollback failed.'},
                { status: response.status }
            );
        }

        const result = await response.json();
        console.log("Rollback successful: " + result);

        return NextResponse.json(request);
    } catch (error) {
        console.error("Rollback request error: " + error)
        return NextResponse.json(
            { error: 'Failed to connect to executor service.'},
            { status: 500 }
        );
    }
}