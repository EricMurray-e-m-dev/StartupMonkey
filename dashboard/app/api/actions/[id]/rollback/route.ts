import { NextRequest, NextResponse } from "next/server";

const EXECUTOR_URL = process.env.EXECUTOR_URL || "http://localhost:8084";

export async function POST(
    request: NextRequest,
    { params }: { params: Promise<{ id: string }> }
) {
    try {
        const { id } = await params;

        console.log(`Rolling back action: ${id}`);

        const response = await fetch(`${EXECUTOR_URL}/api/actions/${id}/rollback`, {
            method: "POST",
        });

        if (!response.ok) {
            const error = await response.text();
            return NextResponse.json(
                { error: error || "Failed to rollback action" },
                { status: response.status }
            );
        }

        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error("Failed to rollback action:", error);
        return NextResponse.json(
            { error: "Failed to rollback action" },
            { status: 500 }
        );
    }
}