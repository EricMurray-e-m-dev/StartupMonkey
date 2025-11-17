import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = 'http://localhost:3001';

export async function GET() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/metrics/latest`, {
            cache: 'no-store'
        });
        
        const data = await response.json();
        console.log('metrics from collector:', data ? 'got data' : 'null');
        return NextResponse.json(data || {});
    } catch (error) {
        console.error("Failed to fetch from collector:", error);
        return NextResponse.json({});
    }
}