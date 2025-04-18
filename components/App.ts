// components/App.ts

import { ThemeManager } from './ThemeManager';
import { DocumentTitle } from './DocumentTitle';
import { Modal } from './Modal';
import { SectionManager } from './SectionManager';
import { ToastManager } from './ToastManager';
import { TableOfContents } from './TableOfContents'; // Import the new component
import { LeetCoachDocument, DocumentSection, DocumentMetadata, TextDocumentSection, DrawingDocumentSection } from './types';
import { DOCUMENT } from "./samples";

/**
 * Main application initialization
 */
class LeetCoachApp {
    private themeManager: ThemeManager | null = null;
    private documentTitle: DocumentTitle | null = null;
    private modal: Modal | null = null;
    private sectionManager: SectionManager | null = null;
    private toastManager: ToastManager | null = null;
    private tableOfContents: TableOfContents | null = null; // Add property for TOC instance

    constructor() {
        this.initializeComponents();
        this.bindEvents();
    }

    /** Initialize all application components */
    private initializeComponents(): void {
        this.themeManager = ThemeManager.init();
        this.modal = Modal.init();
        this.toastManager = ToastManager.init();
        this.documentTitle = DocumentTitle.init();

        // Instantiate SectionManager first
        this.sectionManager = SectionManager.init();

        // Instantiate TableOfContents, passing the callback for adding sections
        this.tableOfContents = TableOfContents.init({
            onAddSectionClick: () => {
                // When TOC add button is clicked, tell SectionManager to open the type selector
                // Add section at the end (insertAfterId = null)
                this.sectionManager?.openSectionTypeSelector(null);
            }
        });

        // Connect SectionManager to TableOfContents
        if (this.sectionManager && this.tableOfContents) {
             this.sectionManager.setTocComponent(this.tableOfContents);
        }

        console.log('LeetCoach application initialized');
    }

    /** Bind application-level events */
    private bindEvents(): void {
        // --- Theme switcher events (remain the same) ---
        const lightBtn = document.getElementById('light-theme-btn');
        const darkBtn = document.getElementById('dark-theme-btn');
        const systemBtn = document.getElementById('system-theme-btn');
        if (lightBtn) lightBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.LIGHT));
        if (darkBtn) darkBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.DARK));
        if (systemBtn) systemBtn.addEventListener('click', () => ThemeManager.setTheme(ThemeManager.SYSTEM));


        // --- Mobile menu toggle (remain the same) ---
        const mobileMenuButton = document.getElementById('mobile-menu-button');
        const sidebar = document.querySelector('.md\\:w-64'); // Correct selector for the sidebar container
        // Note: Close button is now handled INSIDE TableOfContents.ts
        if (mobileMenuButton && sidebar) {
            mobileMenuButton.addEventListener('click', () => {
               // Tell the TOC component to handle opening
               this.tableOfContents?.toggleDrawer(); // Or openDrawer() if toggle preferred
            });
        }


        // --- Save button (remain the same) ---
        const saveButton = document.querySelector('header button.bg-blue-600'); // More specific selector
        if (saveButton) {
            saveButton.addEventListener('click', this.saveDocument.bind(this));
        }


        // --- Export button (remain the same) ---
        const exportButton = document.querySelector('header button.bg-gray-200'); // More specific selector
        if (exportButton) {
            exportButton.addEventListener('click', this.exportDocument.bind(this));
        }
    }


    /** Load document data into the components */
    public loadDocument(doc: LeetCoachDocument): void {
        console.log("Loading document:", doc.metadata.id);
        if (this.documentTitle) {
            this.documentTitle.setTitle(doc.title);
        }
        if (this.sectionManager) {
            // loadSections will now also trigger the TOC update via the connected component
            this.sectionManager.loadSections(doc.sections);
        } else {
             console.error("SectionManager not initialized, cannot load document sections.");
        }
    }


    /** Save document */
    private saveDocument(): void {
        console.log("Attempting to save document...");

        if (!this.documentTitle || !this.sectionManager || !this.toastManager) {
            console.error("Cannot save: Core components not initialized.");
            this.toastManager?.showToast('Save Failed', 'Could not save document.', 'error');
            return;
        }

        const currentTimestamp = new Date().toISOString();
        const documentData: LeetCoachDocument = {
            metadata: {
                id: "doc-placeholder-uuid", // Replace with actual ID
                schemaVersion: "1.0",
                lastSavedAt: currentTimestamp
            },
            title: this.documentTitle.getTitle(),
            sections: this.sectionManager.getDocumentSections() // Get data from manager
        };

        try {
            const jsonString = JSON.stringify(documentData, null, 2);
            console.log("--- LeetCoach Document State (JSON) ---");
            console.log(jsonString);
            console.log("---------------------------------------");
            this.toastManager.showToast('Document Saved', 'Content logged to console.', 'success');
            // Update last saved time display in DocumentTitle component
            this.documentTitle.updateLastSavedTime();

        } catch (error) {
            console.error("Error serializing document data:", error);
            this.toastManager.showToast('Save Error', 'Could not prepare data for saving.', 'error');
        }
    }


    /** Export document (Placeholder) */
    private exportDocument(): void {
        if (this.toastManager) {
            this.toastManager.showToast('Export started', 'Your document is being prepared for export.', 'info');
            setTimeout(() => {
                this.toastManager?.showToast('Export complete', 'Document export simulation finished.', 'success');
            }, 1500);
        }
    }


    /** Initialize the application */
    public static init(): LeetCoachApp {
        return new LeetCoachApp();
    }
}


// Initialize the application when the DOM is fully loaded
document.addEventListener('DOMContentLoaded', () => {
    const lc = LeetCoachApp.init();
    // Ensure sample document is loaded *after* components are initialized and connected
    lc.loadDocument(DOCUMENT);
});
