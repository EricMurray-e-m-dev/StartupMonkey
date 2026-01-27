import { NextRequest, NextResponse } from "next/server";

const COLLECTOR_URL = process.env.COLLECTOR_URL || "http://localhost:3001";

export async function POST(
    request: NextRequest,
    { params }: { params: Promise<{ id: string }> }
) {
    try {
        const { id } = await params;

        const response = await fetch(`${COLLECTOR_URL}/actions/${id}/approve`, {
            method: "POST",
        });

        if (!response.ok) {
            const error = await response.text();
            return NextResponse.json(
                { error: error || "Failed to approve action" },
                { status: response.status }
            );
        }

        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to approve action:", error);
        return NextResponse.json(
            { error: "Failed to approve action" },
            { status: 500 }
        );
    }
}