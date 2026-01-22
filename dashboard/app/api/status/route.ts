import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/status`, {
            cache: 'no-store'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch status:", error);
        return NextResponse.json({ error: 'Failed to fetch status' }, { status: 500 });
    }
}