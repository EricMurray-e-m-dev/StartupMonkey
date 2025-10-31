"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { Button } from "../ui/button";

export function Header() {
    const { theme, setTheme } = useTheme();

    return (
        <header className="flex h-16 items-center justify-between border-b bg-background px-6">
            <div>
                <h2 className="text-lg font-semibold">Database Performance Monitor</h2>
                <p className="text-sm text-muted-foreground">
                    Real-time autonomous optimisation
                </p>
            </div>

            {/* Theme Toggle */}
            <Button
                variant={"outline"}
                size={"icon"}
                onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>
                    <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                    <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                    <span className="sr-only">Toggle Theme</span>
                </Button>
        </header>
    );
}