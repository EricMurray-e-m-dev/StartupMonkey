import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/detections`, {
            cache: 'no-store'
        });
        
        const data = await response.json();
        console.log('detections from collector:', data.length, 'items');
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch detections from collector:", error);
        return NextResponse.json([]);
    }
}