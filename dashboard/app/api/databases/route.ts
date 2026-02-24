import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET(request: Request) {
    try {
        const { searchParams } = new URL(request.url);
        const enabledOnly = searchParams.get('enabled_only') || 'false';
        
        const response = await fetch(
            `${COLLECTOR_URL}/databases?enabled_only=${enabledOnly}`,
            { cache: 'no-store' }
        );
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch databases:", error);
        return NextResponse.json([], { status: 500 });
    }
}

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const response = await fetch(`${COLLECTOR_URL}/databases`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to register database:", error);
        return NextResponse.json({ error: 'Failed to register database' }, { status: 500 });
    }
}