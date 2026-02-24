"use client";

import { Moon, Sun, Settings } from "lucide-react";
import { useTheme } from "next-themes";
import { Button } from "../ui/button";
import { DatabaseSelector } from "../DatabaseSelector";
import Link from "next/link";

export function Header() {
    const { theme, setTheme } = useTheme();

    return (
        <header className="flex h-16 items-center justify-between border-b bg-background px-6">
            <div>
                <h2 className="text-lg font-semibold">StartupMonkey</h2>
                <p className="text-sm text-muted-foreground">
                    Database Performance Monitor
                </p>
            </div>

            <div className="flex items-center gap-4">
                {/* Database Selector */}
                <DatabaseSelector />

                {/* Settings Link */}
                <Link href="/settings">
                    <Button variant="ghost" size="icon">
                        <Settings className="h-5 w-5" />
                        <span className="sr-only">Settings</span>
                    </Button>
                </Link>

                {/* Theme Toggle */}
                <Button
                    variant="outline"
                    size="icon"
                    onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
                >
                    <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                    <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                    <span className="sr-only">Toggle Theme</span>
                </Button>
            </div>
        </header>
    );
}