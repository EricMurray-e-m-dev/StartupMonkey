"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, Activity, AlertTriangle, Zap } from "lucide-react";
import { cn } from "@/lib/utils";

const navigation = [
    { name: "Overview", href: "/", Icon: Home },
    { name: "Metrics", href: "/metrics", Icon: Activity },
    { name: "Detections", href: "/detections", Icon: AlertTriangle },
    { name: "Actions", href: "/actions", Icon: Zap },
];

export function Sidebar() {
    const pathName = usePathname();

    return (
        <div className="flex h-full w-64 flex-col border-r bg-sidebar">
            {/* Logo */}
            <div className="flex h-16 items-center border-b px-6">
                <h1 className="text-xl font-bold">StartupMonkey</h1>
            </div>

            {/* Navigation */}
            <nav className="flex-1 space-y-1 px-3 py-4">
                {navigation.map((item) => {
                    const isActive = pathName == item.href
                    return (
                        <Link
                            key={item.name}
                            href={item.href}
                            className={cn(
                                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                                isActive
                                ? "bg-sidebare-accent text-sidebar-accent-foreground"
                                : "text-sidebar-foreground hover:bg-sidebar-accent/50"
                            )}
                            >
                                <item.Icon className="h-5 w-5" />
                                {item.name}
                            </Link>
                    );
                })}
            </nav>

            {/* Footer */}
            <div className="border-t p-4">
                <p className="text-xs text-muted-foreground">
                    v0.0.5
                </p>
            </div>
        </div>
    );
}