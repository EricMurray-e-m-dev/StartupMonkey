import { NextResponse } from "next/server";

export const dynamic = 'force-dynamic';

const COLLECTOR_URL = process.env.NEXT_PUBLIC_COLLECTOR_URL || 'http://localhost:3001';

export async function GET(
    request: Request,
    { params }: { params: Promise<{ id: string }> }
) {
    try {
        const { id } = await params;
        const response = await fetch(`${COLLECTOR_URL}/databases/${id}`, {
            cache: 'no-store'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to fetch database:", error);
        return NextResponse.json({ error: 'Failed to fetch database' }, { status: 500 });
    }
}

export async function PUT(
    request: Request,
    { params }: { params: Promise<{ id: string }> }
) {
    try {
        const { id } = await params;
        const body = await request.json();
        const response = await fetch(`${COLLECTOR_URL}/databases/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to update database:", error);
        return NextResponse.json({ error: 'Failed to update database' }, { status: 500 });
    }
}

export async function DELETE(
    request: Request,
    { params }: { params: Promise<{ id: string }> }
) {
    try {
        const { id } = await params;
        const response = await fetch(`${COLLECTOR_URL}/databases/${id}`, {
            method: 'DELETE'
        });
        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to delete database:", error);
        return NextResponse.json({ error: 'Failed to delete database' }, { status: 500 });
    }
}