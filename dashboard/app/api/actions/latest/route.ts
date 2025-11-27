import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET() {
    try {
        const response = await fetch(`${COLLECTOR_URL}/actions`, {
            cache: 'no-store'
        });
        
        const data = await response.json();
        console.log('actions from collector:', data.length, 'items');
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch actions from collector:", error);
        return NextResponse.json([]);
    }
}