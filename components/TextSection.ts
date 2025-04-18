// components/TextSection.ts

import { BaseSection } from './BaseSection';
import { SectionData, SectionCallbacks } from './Section'; // Or move interfaces if needed
import { TextContent } from './types';

export class TextSection extends BaseSection {

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
    }

    protected populateViewContent(): void {
        const viewContent = this.contentContainer?.querySelector('.section-view-content');
        if (viewContent) {
            // Use loaded content if available (ensure it's treated as TextContent)
            const initialContent = (typeof this.data.content === 'string' && this.data.content.length > 0)
                ? this.data.content
                : '<p class="text-gray-400 dark:text-gray-500 italic">Click to add content...</p>'; // Placeholder if empty
            viewContent.innerHTML = initialContent;
        } else {
            console.warn(`View content area not found for text section ${this.data.id}`);
        }
    }

    protected populateEditContent(): void {
        const editorContainer = this.contentContainer?.querySelector('.text-editor-container');
        if (editorContainer instanceof HTMLElement) { // Check if it's an element
             // Use loaded content if available
            const initialContent = (typeof this.data.content === 'string') ? this.data.content : '';
            editorContainer.innerHTML = initialContent;
            // Optional: Focus the editor immediately
            editorContainer.focus();
        } else {
            console.warn(`Edit content area not found for text section ${this.data.id}`);
        }
    }

    protected bindViewModeEvents(): void {
        const viewContent = this.contentContainer?.querySelector('.section-view-content');
        if (viewContent) {
            // Clear previous listener before adding
            viewContent.removeEventListener('click', this.handleViewClick);
            viewContent.addEventListener('click', this.handleViewClick.bind(this));
        }
    }

    // Handler function to ensure 'this' context is correct
    private handleViewClick(): void {
        this.switchToEditMode();
    }

    protected bindEditModeEvents(): void {
        const saveButton = this.contentContainer?.querySelector('.section-edit-save');
        const cancelButton = this.contentContainer?.querySelector('.section-edit-cancel');

        if (saveButton) {
            saveButton.addEventListener('click', () => {
                this.switchToViewMode(true); // Save changes
            });
        }
        if (cancelButton) {
            cancelButton.addEventListener('click', () => {
                this.switchToViewMode(false); // Discard changes
            });
        }

        // Optional: Add keydown listeners to the editor container if needed (e.g., Ctrl+S for save)
        const editorContainer = this.contentContainer?.querySelector('.text-editor-container') as HTMLElement;
         if (editorContainer) {
             // Example: Save on Ctrl+Enter (or Cmd+Enter)
             editorContainer.addEventListener('keydown', (e: KeyboardEvent) => {
                 if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                     e.preventDefault();
                     this.switchToViewMode(true);
                 }
                 // Could add Escape listener to cancel here too
                 if (e.key === 'Escape'){
                    e.preventDefault();
                    this.switchToViewMode(false);
                 }
             });
         }
    }

    protected getContentFromEditMode(): TextContent {
        const editorContainer = this.contentContainer?.querySelector('.text-editor-container');
        // Return contenteditable div's innerHTML, or empty string if not found
        return editorContainer instanceof HTMLElement ? editorContainer.innerHTML : '';
    }
}
