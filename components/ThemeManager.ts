// components/theme.ts

/**
 * Theme management for LeetCoach application
 */
export class ThemeManager {
    // Theme options
    static LIGHT = 'light';
    static DARK = 'dark';
    static SYSTEM = 'system';
    
    /**
     * Initialize theme based on saved preference or system default
     */
    static initialize(): void {
        const savedTheme = localStorage.getItem('theme');
        
        if (savedTheme === ThemeManager.DARK || 
            (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }
    
    /**
     * Set theme and save preference
     */
    static setTheme(theme: string): void {
        if (theme === ThemeManager.SYSTEM) {
            localStorage.removeItem('theme');
            if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
                document.documentElement.classList.add('dark');
            } else {
                document.documentElement.classList.remove('dark');
            }
        } else if (theme === ThemeManager.DARK) {
            localStorage.setItem('theme', ThemeManager.DARK);
            document.documentElement.classList.add('dark');
        } else {
            localStorage.setItem('theme', ThemeManager.LIGHT);
            document.documentElement.classList.remove('dark');
        }
    }
    
    /**
     * Get current theme
     */
    static getCurrentTheme(): string {
        const savedTheme = localStorage.getItem('theme');
        if (!savedTheme) return ThemeManager.SYSTEM;
        return savedTheme;
    }

    /**
     * Initialize the ThemeManager
     */
    public static init(): ThemeManager {
        return new ThemeManager ();
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    ThemeManager.initialize();
});
