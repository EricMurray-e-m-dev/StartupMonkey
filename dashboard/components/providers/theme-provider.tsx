"use client";

import { ThemeProvider as NextThemesProvider, type ThemeProviderProps as NextThemesProviderProps } from "next-themes";

type ThemeProviderProps = NextThemesProviderProps;

export function ThemeProvider({ children, ...props}: ThemeProviderProps){
    return <NextThemesProvider {...props}>{children}</NextThemesProvider>;
}