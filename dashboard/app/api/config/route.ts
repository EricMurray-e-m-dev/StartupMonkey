import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/config`, {
            cache: 'no-store'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch config:", error);
        return NextResponse.json({ error: 'Failed to fetch config' }, { status: 500 });
    }
}

export async function POST(request: Request) {
    try {
        const config = await request.json();
        const response = await fetch(`${COLLECTOR_URL}/config`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to save config:", error);
        return NextResponse.json({ error: 'Failed to save config' }, { status: 500 });
    }
}