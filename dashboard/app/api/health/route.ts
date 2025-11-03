import { NextResponse } from "next/server";

const startTime = Date.now();

export async function GET() {
    const uptime = Math.floor((Date.now() - startTime) / 1000);

    return NextResponse.json({
        status: "healthy",
        service: "dashboard",
        uptime_seconds: uptime,
        timestamp: Date.now(),
    });
}