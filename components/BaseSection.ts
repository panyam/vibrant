// components/BaseSection.ts

import { Modal } from './Modal';
import { SectionData, SectionCallbacks } from './Section'; // Assuming Section.ts still defines these for now, or move them
import { SectionType, DocumentSection, TextContent, DrawingContent, PlotContent } from './types';

// --- Placeholders for icons/titles - can be moved/refined ---
const TEXT_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" /></svg>`;
const DRAWING_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L6.832 19.82a4.5 4.5 0 01-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 011.13-1.897L16.863 4.487zm0 0L19.5 7.125" /></svg>`;
const PLOT_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5M9 11.25v1.5M12 9v3.75m3-6v6" /></svg>`;
const TYPE_TITLES: Record<SectionType, string> = {
    text: "Text Section",
    drawing: "Drawing Section",
    plot: "Plot Section"
};
// --- End Placeholders ---


/**
 * Abstract base class for all document sections.
 * Handles common functionality like title editing, controls (move, delete, LLM),
 * and switching between view and edit modes using templates from the registry.
 */
export abstract class BaseSection {
    protected data: SectionData;
    protected element: HTMLElement; // The root element for this section (<div id="section-...">)
    protected callbacks: SectionCallbacks;
    protected modal: Modal;
    protected mode: 'view' | 'edit' = 'view';

    // Common DOM elements within the section structure
    protected sectionHeaderElement: HTMLElement | null;
    protected contentContainer: HTMLElement | null; // The .section-content div
    protected titleElement: HTMLElement | null;
    protected typeIconElement: HTMLElement | null;
    protected deleteButton: HTMLElement | null;
    protected moveUpButton: HTMLElement | null;
    protected moveDownButton: HTMLElement | null;
    protected settingsButton: HTMLElement | null;
    protected llmButton: HTMLElement | null;

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        this.data = data;
        this.element = element;
        this.callbacks = callbacks;
        this.modal = Modal.getInstance();

        // Find common structural elements
        this.sectionHeaderElement = this.element.querySelector('.section-header');
        this.contentContainer = this.element.querySelector('.section-content');
        this.titleElement = this.element.querySelector('.section-title');
        this.typeIconElement = this.element.querySelector('.section-type-icon');
        this.deleteButton = this.element.querySelector('.section-delete');
        this.moveUpButton = this.element.querySelector('.section-move-up');
        this.moveDownButton = this.element.querySelector('.section-move-down');
        this.settingsButton = this.element.querySelector('.section-settings');
        this.llmButton = this.element.querySelector('.section-ai');

        if (!this.contentContainer) {
            console.error(`Section content container not found for section ID: ${this.data.id}`);
        }

        this.updateDisplayTitle(); // Set initial title display
        this.updateTypeIcon();   // Set initial type icon
        this.bindCommonEvents();
        this.initializeSectionDisplay(); // Load initial view mode template
    }

    /** Updates the displayed title in the section header */
    protected updateDisplayTitle(): void {
        if (this.titleElement) {
            this.titleElement.textContent = this.data.title;
        }
    }

    /** Updates the type icon in the section header */
    protected updateTypeIcon(): void {
        if (this.typeIconElement) {
            let iconSvg: string;
            switch (this.data.type) {
                case 'drawing': iconSvg = DRAWING_ICON_SVG; break;
                case 'plot':    iconSvg = PLOT_ICON_SVG; break;
                case 'text':
                default:        iconSvg = TEXT_ICON_SVG; break;
            }
            this.typeIconElement.innerHTML = iconSvg;
            this.typeIconElement.setAttribute('title', TYPE_TITLES[this.data.type] || 'Section');
        }
    }

    /** Binds events for common controls (delete, move, LLM, title editing etc.) */
    protected bindCommonEvents(): void {
        // Title editing
        if (this.titleElement) {
            // Ensure only one listener is attached if constructor is called multiple times (unlikely but safe)
            this.titleElement.removeEventListener('click', this.startTitleEdit);
            this.titleElement.addEventListener('click', this.startTitleEdit.bind(this));
        }

        // Delete button
        if (this.deleteButton) {
            this.deleteButton.addEventListener('click', () => {
                this.callbacks.onDelete?.(this.data.id);
            });
        }

        // Move up button
        if (this.moveUpButton) {
            this.moveUpButton.addEventListener('click', () => {
                this.callbacks.onMoveUp?.(this.data.id);
            });
        }

        // Move down button
        if (this.moveDownButton) {
            this.moveDownButton.addEventListener('click', () => {
                this.callbacks.onMoveDown?.(this.data.id);
            });
        }

        // Settings button
        if (this.settingsButton) {
            this.settingsButton.addEventListener('click', this.openSettings.bind(this));
        }

        // LLM button
        if (this.llmButton) {
            this.llmButton.addEventListener('click', this.openLlmDialog.bind(this));
        }
    }

    /** Initializes the section by loading the view mode template */
    protected initializeSectionDisplay(): void {
        this.switchToViewMode(false); // Start in view mode, don't save anything
    }

    /** Handles the logic for editing the section title */
    protected startTitleEdit(e: Event): void {
        e.preventDefault();
        e.stopPropagation();

        if (!this.titleElement) return;

        const currentTitle = this.data.title; // Store original title for cancel
        const input = document.createElement('input');
        input.type = 'text';
        // Apply similar styling as the title for consistency, adjust as needed
        input.className = 'text-lg font-medium border border-gray-300 dark:border-gray-600 dark:bg-gray-700 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:text-white px-1 py-0.5';
        input.value = currentTitle;
        input.style.minWidth = '200px'; // Prevent input from being too small

        // Temporarily replace title span with input
        const titleSpan = this.titleElement; // Assuming titleElement is the span
        const parent = titleSpan.parentNode;
        if (!parent) return;

        parent.replaceChild(input, titleSpan);
        input.focus();
        input.select();

        const cleanup = () => {
            input.removeEventListener('blur', handleSave);
            input.removeEventListener('keydown', handleKeyDown);
            // Ensure the original title span is back if the input is removed
            if (!input.parentNode) { // If already removed by blur save/cancel
               return;
            }
             parent.replaceChild(titleSpan, input);
             titleSpan.textContent = this.data.title; // Ensure display is up-to-date
        };

        const handleSave = () => {
            const newTitle = input.value.trim();
            if (newTitle && newTitle !== this.data.title) {
                this.data.title = newTitle;
                this.callbacks.onTitleChange?.(this.data.id, newTitle);
            }
            // Restore title display (even if unchanged, removes input)
            parent.replaceChild(titleSpan, input);
            titleSpan.textContent = this.data.title; // Update displayed text
            cleanup();
        };

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                input.blur(); // Trigger save
            } else if (e.key === 'Escape') {
                e.preventDefault();
                // Restore original title visually and remove input
                parent.replaceChild(titleSpan, input);
                titleSpan.textContent = currentTitle; // Revert display
                this.data.title = currentTitle; // Revert data if needed (though not saved yet)
                cleanup();
            }
        };

        input.addEventListener('blur', handleSave, { once: true }); // Auto-save on blur
        input.addEventListener('keydown', handleKeyDown);
    }


    /** Opens the section settings modal (Placeholder) */
    protected openSettings(): void {
        console.log('Section settings clicked for', this.data.id);
        alert(`Settings for section: ${this.data.title} (ID: ${this.data.id})`);
    }

    /** Opens the LLM dialog modal for this section */
    protected openLlmDialog(): void {
        this.modal.show('llm-dialog', {
            sectionId: this.data.id,
            sectionType: this.data.type,
            sectionTitle: this.data.title
        });

        // Update the current section display in the LLM dialog (if modal content is ready)
        // Note: This might need a slight delay or callback if modal content isn't immediately available
        setTimeout(() => {
            const currentSectionElement = document.getElementById('llm-current-section');
            if (currentSectionElement) {
                currentSectionElement.textContent = this.data.title;
            }
        }, 0);
    }

    /**
     * Switches the section to View mode.
     * Optionally saves changes from Edit mode before switching.
     * Loads the appropriate view template from the registry.
     */
    public switchToViewMode(saveChanges: boolean): void {
        if (this.mode === 'edit' && saveChanges) {
            const newContent = this.getContentFromEditMode();
            // Check if content actually changed (simple check for strings, might need deeper compare for objects)
            if (JSON.stringify(newContent) !== JSON.stringify(this.data.content)) {
                 this.data.content = newContent;
                 this.callbacks.onContentChange?.(this.data.id, this.data.content as string); // Cast for simplicity, might need adjustment
                 console.log(`Section ${this.data.id} content saved.`);
            } else {
                 console.log(`Section ${this.data.id} content unchanged.`);
            }
        }

        this.mode = 'view';
        console.log(`Switching ${this.data.id} to view mode.`);
        if (this.loadTemplate('view')) {
            this.populateViewContent();
            this.bindViewModeEvents();
        } else {
            console.error("Failed to load view template for section", this.data.id);
        }
    }

    /**
     * Switches the section to Edit mode.
     * Loads the appropriate edit template from the registry.
     */
    public switchToEditMode(): void {
        if (this.mode === 'edit') return; // Already in edit mode

        this.mode = 'edit';
        console.log(`Switching ${this.data.id} to edit mode.`);
        if (this.loadTemplate('edit')) {
            this.populateEditContent();
            this.bindEditModeEvents();
        } else {
             console.error("Failed to load edit template for section", this.data.id);
             this.mode = 'view'; // Revert mode if template fails
        }
    }

    /**
     * Loads the specified template (view or edit) from the registry
     * and injects it into the section's content container.
     * Uses the hidden div approach for now.
     *
     * @returns True if the template was loaded successfully, false otherwise.
     */
    protected loadTemplate(mode: 'view' | 'edit'): boolean {
        if (!this.contentContainer) return false;

        const templateId = `${this.data.type}-section-${mode}`;
        const templateRegistry = document.getElementById('template-registry');
        if (!templateRegistry) {
            console.error("Template registry not found!");
            return false;
        }

        // TODO: Migrate registry to use <template> tag for better semantics and performance.
        // When migrating, find <template> and clone its '.content' DocumentFragment.
        const templateWrapper = templateRegistry.querySelector(`[data-template-id="${templateId}"]`);
        if (!templateWrapper) {
            console.error(`Template not found in registry: ${templateId}`);
            return false;
        }

        // Using hidden div: Clone the first child element which is the actual template root
        const templateRootElement = templateWrapper.firstElementChild?.cloneNode(true) as HTMLElement | null;
        if (!templateRootElement) {
             console.error(`Template content is empty for: ${templateId}`);
             return false;
        }

        // Clear previous content and append the new template
        this.contentContainer.innerHTML = '';
        this.contentContainer.appendChild(templateRootElement);

        return true;
    }

    // --- Abstract methods to be implemented by derived classes ---

    /** Populates the loaded View mode template with the section's current data. */
    protected abstract populateViewContent(): void;

    /** Populates the loaded Edit mode template with the section's current data and initializes editors. */
    protected abstract populateEditContent(): void;

    /** Binds event listeners specific to the elements within the View mode template. */
    protected abstract bindViewModeEvents(): void;

    /** Binds event listeners specific to the elements within the Edit mode template (e.g., save/cancel buttons). */
    protected abstract bindEditModeEvents(): void;

    /** Retrieves the current content state from the Edit mode UI elements. */
    protected abstract getContentFromEditMode(): SectionData['content'];


    // --- Public API ---

    /** Updates the displayed section number */
    public updateNumber(number: number): void {
        this.data.order = number;
        const numberElement = this.element.querySelector('.section-number');
        if (numberElement) {
            numberElement.textContent = `${number}.`;
        }
    }

    /** Returns a copy of the section's data */
    public getData(): SectionData {
        // Return a copy to prevent direct modification
        return JSON.parse(JSON.stringify(this.data));
    }

    /** Returns the section's ID */
    public getId(): string {
        return this.data.id;
    }

     /** Returns the section's current order */
     public getOrder(): number {
        return this.data.order;
    }

    /**
     * Gets the section data formatted for the document model.
     * Ensures content reflects the latest saved state.
     */
    public getDocumentData(): DocumentSection {
        // Important: Assumes this.data.content is kept up-to-date by switchToViewMode(true)
        // If called while in edit mode *before* saving, it returns the *last saved* content.
         const baseData = {
            id: this.data.id,
            title: this.data.title,
            order: this.data.order,
        };

        // Return type assertion based on the section's type
        switch (this.data.type) {
            case 'text':
                return { ...baseData, type: 'text', content: this.data.content as TextContent };
            case 'drawing':
                return { ...baseData, type: 'drawing', content: this.data.content as DrawingContent };
            case 'plot':
                return { ...baseData, type: 'plot', content: this.data.content as PlotContent };
            default:
                // Should not happen if types are handled correctly
                console.error(`Unknown section type in getDocumentData: ${this.data.type}`);
                // Fallback or throw error - returning as text for now
                return { ...baseData, type: 'text', content: String(this.data.content) };
        }
    }
}
