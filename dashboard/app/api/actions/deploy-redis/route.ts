import { NextRequest, NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const EXECUTOR_URL = process.env.EXECUTOR_HTTP_URL || 'http://localhost:8084';

export async function POST(request: NextRequest) {
    try {
        const body = await request.json();

        const response = await fetch(`${EXECUTOR_URL}/api/deploy-redis`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            return NextResponse.json(
                { error: error || 'Failed to deploy Redis' },
                { status: response.status }
            );
        }

        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error('Failed to deploy Redis:', error);
        return NextResponse.json(
            { error: 'Failed to connect to executor service' },
            { status: 500 }
        );
    }
}