// components/App.ts

import { ThemeManager } from './ThemeManager';
import { DocumentTitle } from './DocumentTitle';

/**
 * Main application initialization
 */
class LeetCoachApp {
    constructor() {
        this.initializeThemeSwitcher();
        this.initializeDocumentTitle();
        alert("Here...")
        // Other initialization will go here
    }

    /**
     * Set up theme switcher buttons
     */
    private initializeThemeSwitcher(): void {
        const lightBtn = document.getElementById('light-theme-btn');
        const darkBtn = document.getElementById('dark-theme-btn');
        const systemBtn = document.getElementById('system-theme-btn');

        if (lightBtn) {
            lightBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.LIGHT));
        }
        
        if (darkBtn) {
            darkBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.DARK));
        }
        
        if (systemBtn) {
            systemBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.SYSTEM));
        }
    }

    /**
     * Initialize document title editing functionality
     */
    private initializeDocumentTitle(): void {
        DocumentTitle.init();
    }
}

console.log("Here????")
// Initialize the application when the DOM is fully loaded
document.addEventListener('DOMContentLoaded', () => {
    new LeetCoachApp();
});
