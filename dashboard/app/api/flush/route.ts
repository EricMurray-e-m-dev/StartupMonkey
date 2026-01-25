import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function POST() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/flush`, {
            method: 'POST'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to flush:", error);
        return NextResponse.json({ error: 'Failed to flush' }, { status: 500 });
    }
}