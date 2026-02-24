import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET(request: Request) {
    try {
        const { searchParams } = new URL(request.url);
        const databaseId = searchParams.get('database_id');
        
        const params = databaseId ? `?database_id=${databaseId}` : '';
        const response = await fetch(`${COLLECTOR_URL}/metrics/latest${params}`, {
            cache: 'no-store'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch metrics:", error);
        return NextResponse.json({}, { status: 500 });
    }
}