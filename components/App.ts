// components/App.ts

import { ThemeManager } from './ThemeManager';
import { DocumentTitle } from './DocumentTitle';
import { Modal } from './Modal';
import { SectionManager } from './SectionManager';
import { ToastManager } from './ToastManager'; // We'll create this next

/**
 * Main application initialization
 */
class LeetCoachApp {
    private themeManager: ThemeManager | null = null;
    private documentTitle: DocumentTitle | null = null;
    private modal: Modal | null = null;
    private sectionManager: SectionManager | null = null;
    private toastManager: ToastManager | null = null;

    constructor() {
        this.initializeComponents();
        this.bindEvents();
        // Other initialization will go here
    }

    /**
     * Initialize all application components
     */
    private initializeComponents(): void {
        // Initialize in dependency order
        this.themeManager = ThemeManager.init();
        this.modal = Modal.init();
        this.documentTitle = DocumentTitle.init();
        this.sectionManager = SectionManager.init();
        this.toastManager = ToastManager.init();
        
        // Log initialization
        console.log('LeetCoach application initialized');
    }

    /**
     * Bind application-level events
     */
    private bindEvents(): void {
        // Theme switcher buttons
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

        // Mobile menu toggle
        const mobileMenuButton = document.getElementById('mobile-menu-button');
        const sidebar = document.querySelector('.md\\:w-64');
        const closeSidebarButton = document.getElementById('close-sidebar');
        
        if (mobileMenuButton && sidebar) {
            mobileMenuButton.addEventListener('click', () => {
                sidebar.classList.toggle('hidden');
                sidebar.classList.toggle('block');
            });
        }
        
        if (closeSidebarButton && sidebar) {
            closeSidebarButton.addEventListener('click', () => {
                sidebar.classList.add('hidden');
                sidebar.classList.remove('block');
            });
        }

        // Save button
        const saveButton = document.querySelector('button.bg-blue-600:not(#llm-submit):not(#llm-apply)');
        if (saveButton) {
            saveButton.addEventListener('click', this.saveDocument.bind(this));
        }

        // Export button
        const exportButton = document.querySelector('button.bg-gray-200');
        if (exportButton) {
            exportButton.addEventListener('click', this.exportDocument.bind(this));
        }
    }

    /**
     * Save document
     */
    private saveDocument(): void {
        // Placeholder for save functionality
        const timestamp = new Date();
        const hours = timestamp.getHours();
        const minutes = timestamp.getMinutes();
        const ampm = hours >= 12 ? 'PM' : 'AM';
        const formattedHours = hours % 12 || 12;
        const formattedMinutes = minutes < 10 ? `0${minutes}` : minutes;
        
        // Update timestamp
        const lastSavedElement = document.getElementById('last-saved-time');
        if (lastSavedElement) {
            lastSavedElement.textContent = `Last saved: Today at ${formattedHours}:${formattedMinutes} ${ampm}`;
        }
        
        // Show success toast
        if (this.toastManager) {
            this.toastManager.showToast('Document saved', 'Your document has been saved successfully.', 'success');
        }
    }

    /**
     * Export document
     */
    private exportDocument(): void {
        // Placeholder for export functionality
        if (this.toastManager) {
            this.toastManager.showToast('Export started', 'Your document is being prepared for export.', 'info');
            
            // Simulate export process
            setTimeout(() => {
                this.toastManager?.showToast('Export complete', 'Your document has been exported successfully.', 'success');
            }, 1500);
        }
    }

    /**
     * Initialize the application
     */
    public static init(): LeetCoachApp {
        return new LeetCoachApp();
    }
}

// Initialize the application when the DOM is fully loaded
document.addEventListener('DOMContentLoaded', () => {
    LeetCoachApp.init();
});
