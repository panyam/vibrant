// components/TextSection.ts

import { BaseSection } from './BaseSection';
import { SectionData, SectionCallbacks, TextContent } from './types';

// --- Removed TinyMCE imports ---

// Import markdown-it
import MarkdownIt from 'markdown-it';

export class TextSection extends BaseSection {

    // --- Removed TinyMCE editorInstance ---
    // private editorInstance: Editor | null = null;
    private editorInstance: any = null; // Placeholder for Milkdown later

    // Add markdown-it instance
    private _md: MarkdownIt;

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
        // Initialize markdown-it
        // You can configure it here if needed (e.g., enable HTML tags, add plugins)
    }

    get md(): MarkdownIt {
      if (!this._md) {
        this._md = new MarkdownIt({
            html: true, // Allow HTML tags in Markdown source
            linkify: true, // Autoconvert URL-like text to links
            typographer: true, // Enable some language-neutral replacement + quotes beautification
        });
      }
      return this._md
    }

    protected populateViewContent(): void {
        const viewContent = this.contentContainer?.querySelector('.section-view-content');
        if (viewContent) {
            // --- Use markdown-it to render ---
            const markdownContent = typeof this.data.content === 'string' ? this.data.content : '';
            const placeholder = '<p class="text-gray-400 dark:text-gray-500 italic">Click to add content...</p>';

            try {
                 // Render markdown, or use placeholder if markdown is empty
                const renderedHtml = this.md.render(markdownContent.trim() ? markdownContent : placeholder);
                viewContent.innerHTML = renderedHtml;
            } catch (error) {
                 console.error(`Error rendering Markdown for section ${this.data.id}:`, error);
                 viewContent.innerHTML = `<p class="text-red-500">Error displaying content.</p>`; // Display error
            }
        } else {
            console.warn(`View content area not found for text section ${this.data.id}`);
        }
    }

    protected populateEditContent(): void {
        const editorTarget = this.contentContainer?.querySelector('.text-editor-target');
        if (editorTarget instanceof HTMLElement) {
             // --- Placeholder for Milkdown initialization ---
            console.log(`Placeholder: Initialize Milkdown editor in section ${this.data.id}`);
            editorTarget.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic">Edit Mode: Milkdown editor will load here.</p>`;
            // TODO: Initialize Milkdown here
            // this.editorInstance = await Editor.make()...create();

        } else {
            console.warn(`Edit content target area (.text-editor-target) not found for text section ${this.data.id}`);
        }
    }

    protected bindViewModeEvents(): void {
        const viewContent = this.contentContainer?.querySelector('.section-view-content');
        if (viewContent) {
            viewContent.removeEventListener('click', this.handleViewClick);
            viewContent.addEventListener('click', this.handleViewClick.bind(this));
        }
    }

    private handleViewClick(): void {
        this.switchToEditMode();
    }

    protected bindEditModeEvents(): void {
        const saveButton = this.contentContainer?.querySelector('.section-edit-save');
        const cancelButton = this.contentContainer?.querySelector('.section-edit-cancel');

        if (saveButton) {
            saveButton.removeEventListener('click', this.handleSaveClick);
            saveButton.addEventListener('click', this.handleSaveClick.bind(this));
        }
        if (cancelButton) {
            cancelButton.removeEventListener('click', this.handleCancelClick);
            cancelButton.addEventListener('click', this.handleCancelClick.bind(this));
        }
    }

    private handleSaveClick(): void {
        this.switchToViewMode(true);
    }
    private handleCancelClick(): void {
        this.switchToViewMode(false);
    }


    protected getContentFromEditMode(): TextContent {
        // --- Placeholder for Milkdown content retrieval ---
        if (this.editorInstance) {
            console.warn(`Placeholder: Attempting to get Markdown content from Milkdown instance for section ${this.data.id}. NOT IMPLEMENTED YET.`);
            // TODO: Implement Milkdown content retrieval (getMarkdown())
            // For now, return the last known data content to avoid breaking save flow completely
            // This assumes data.content *should* be Markdown eventually
            return typeof this.data.content === 'string' ? this.data.content : '';
        }
        console.warn(`Editor instance not found when getting content for section ${this.data.id}.`);
        // Fallback: return existing data (likely HTML still at this stage, but we expect Markdown)
         return typeof this.data.content === 'string' ? this.data.content : '';
    }

    public switchToViewMode(saveChanges: boolean): void {
        let contentToSave: TextContent | undefined = undefined;

        if (this.mode === 'edit') {
            if (saveChanges && this.editorInstance) {
                try {
                    // --- Get content using the (currently placeholder) method ---
                    contentToSave = this.getContentFromEditMode();
                } catch (error) {
                    console.error(`Error getting content from editor before destroy for section ${this.data.id}:`, error);
                }
            }

            if (this.editorInstance) {
                try {
                    // --- Placeholder for Milkdown destruction ---
                    console.log(`Placeholder: Attempting to destroy editor instance for ${this.data.id}`);
                    // TODO: Call Milkdown's destroy/cleanup method
                    // e.g., this.editorInstance.destroy?.();
                    this.editorInstance = null;
                    console.log(`Placeholder: Editor instance reference removed for ${this.data.id}`);
                } catch (error) {
                    console.error(`Error destroying editor instance for section ${this.data.id}:`, error);
                     this.editorInstance = null; // Ensure it's nulled even on error
                }
            }
        }

        // --- Save logic (using potentially stale/placeholder content for now) ---
        if (this.mode === 'edit' && saveChanges && contentToSave !== undefined) {
            const newContent = contentToSave; // This is expected to be Markdown eventually
             // Compare with current data (which might still be HTML or Markdown)
            if (newContent !== this.data.content) {
                this.data.content = newContent;
                this.callbacks.onContentChange?.(this.data.id, this.data.content);
                console.log(`Section ${this.data.id} content potentially saved (as Markdown eventually).`);
            } else {
                console.log(`Section ${this.data.id} content unchanged.`);
            }
        }

        // --- Switch to View Mode ---
        this.mode = 'view';
        console.log(`Switching ${this.data.id} to view mode.`);
        if (this.loadTemplate('view')) {
            this.populateViewContent(); // Uses markdown-it now
            this.bindViewModeEvents();
        } else {
            console.error("Failed to load view template for section", this.data.id);
        }
    }

    public switchToEditMode(): void {
         if (this.mode === 'edit') {
            // TODO: Maybe focus Milkdown editor?
            // this.editorInstance?.focus();
            return;
        }

        if (this.editorInstance) {
            console.warn(`Found existing editor instance when switching to edit mode for ${this.data.id}. Attempting cleanup.`);
             try {
                 // --- Placeholder for Milkdown destruction ---
                  console.log(`Placeholder: Attempting to destroy editor instance for ${this.data.id}`);
                  // TODO: Call Milkdown's destroy/cleanup method
                  this.editorInstance = null;
             } catch (e) { console.error("Error during cleanup remove:", e); }
             this.editorInstance = null;
        }

        this.mode = 'edit';
        console.log(`Switching ${this.data.id} to edit mode.`);
        if (this.loadTemplate('edit')) {
            this.populateEditContent(); // Will eventually initialize Milkdown
            this.bindEditModeEvents();
        } else {
             console.error("Failed to load edit template for section", this.data.id);
             this.mode = 'view'; // Revert mode if template fails
        }
    }


     /** Implement the abstract method from BaseSection */
     protected resizeContentForFullscreen(isEntering: boolean): void {
         console.log(`TextSection ${this.data.id}: Resizing content trigger. Is entering fullscreen: ${isEntering}`);
         // TODO: Check if Milkdown needs explicit resize handling
         // If Milkdown editor is active, might need to notify it:
         // if (this.mode === 'edit' && this.editorInstance) {
         //    console.log("Placeholder: Trigger Milkdown resize/update if needed.");
         //    // e.g., this.editorInstance.update?.(); // Or specific resize command
         // }
     }
}
