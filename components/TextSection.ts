// components/TextSection.ts

import { BaseSection } from './BaseSection';
import { SectionData, SectionCallbacks, TextContent } from './types';

// Import TinyMCE core, theme, icons, and model
import tinymce, { Editor } from 'tinymce/tinymce';
import 'tinymce/themes/silver/theme';
import 'tinymce/icons/default/icons';
import 'tinymce/models/dom/model';


export class TextSection extends BaseSection {

    private editorInstance: Editor | null = null;

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
    }

    protected populateViewContent(): void {
        const viewContent = this.contentContainer?.querySelector('.section-view-content');
        if (viewContent) {
            const initialContent = (typeof this.data.content === 'string' && this.data.content.length > 0)
                ? this.data.content
                : '<p class="text-gray-400 dark:text-gray-500 italic">Click to add content...</p>';
            viewContent.innerHTML = initialContent;
        } else {
            console.warn(`View content area not found for text section ${this.data.id}`);
        }
    }

    protected populateEditContent(): void {
        const editorTarget = this.contentContainer?.querySelector('.text-editor-target');
        if (editorTarget instanceof HTMLElement) {
            const initialContent = (typeof this.data.content === 'string') ? this.data.content : '';
            const isDarkMode = document.documentElement.classList.contains('dark');

            // Base path for TinyMCE assets *as served by the static server*
            // This MUST match webpack's output.publicPath
            const tinyMCEPublicPath = '/static/js/gen/'; // Path relative to domain root

            tinymce.init({
                target: editorTarget,
                plugins: 'autolink lists link image charmap preview anchor searchreplace visualblocks code fullscreen insertdatetime media table help wordcount autoresize',
                toolbar: 'undo redo | formatselect | bold italic backcolor | \
                          alignleft aligncenter alignright alignjustify | \
                          bullist numlist outdent indent | removeformat | help',
                content_style: 'body { font-family:Helvetica,Arial,sans-serif; font-size:14px }',
                height: 300,
                menubar: false,
                statusbar: false,
                autoresize_bottom_margin: 10,
                autoresize_min_height: 200,

                // ** Paths for self-hosting **
                // Ensure these paths correctly point to where CopyPlugin placed the assets,
                // relative to the domain root, using the publicPath.
                skin_url: `${tinyMCEPublicPath}skins/ui/${isDarkMode ? 'oxide-dark' : 'oxide'}`,
                content_css: `${tinyMCEPublicPath}skins/content/${isDarkMode ? 'dark' : 'default'}/content.min.css`,

                license_key: 'gpl',

                setup: (editor: Editor) => {
                    editor.on('init', () => {
                        editor.setContent(initialContent || '');
                        this.editorInstance = editor;
                        editor.focus();
                        console.log(`TinyMCE initialized for section ${this.data.id} (Self-Hosted)`);
                    });
                }
            }).catch((error: any) => {
                console.error(`Error initializing TinyMCE for section ${this.data.id}:`, error);
                editorTarget.innerHTML = `<p class="text-red-500">Error initializing Rich Text Editor.</p>`;
            });

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
        if (this.editorInstance && this.editorInstance.initialized) {
            try {
                return this.editorInstance.getContent() || '';
            } catch (error) {
                console.error(`Error getting content from TinyMCE for section ${this.data.id}:`, error);
                return typeof this.data.content === 'string' ? this.data.content : '';
            }
        }
        console.warn(`TinyMCE instance not found or not initialized when getting content for section ${this.data.id}.`);
        return typeof this.data.content === 'string' ? this.data.content : '';
    }

    public switchToViewMode(saveChanges: boolean): void {
        let contentToSave: TextContent | undefined = undefined;

        if (this.mode === 'edit') {
            if (saveChanges && this.editorInstance && this.editorInstance.initialized) {
                try {
                    contentToSave = this.editorInstance.getContent() || '';
                } catch (error) {
                    console.error(`Error getting content from TinyMCE before destroy for section ${this.data.id}:`, error);
                }
            }

            if (this.editorInstance) {
                try {
                    console.log(`Attempting to remove TinyMCE instance for ${this.data.id}`);
                    tinymce.remove(this.editorInstance);
                    this.editorInstance = null;
                    console.log(`TinyMCE instance removed for ${this.data.id}`);
                } catch (error) {
                    console.error(`Error removing TinyMCE instance for section ${this.data.id}:`, error);
                    this.editorInstance = null;
                }
            }
        }

        if (this.mode === 'edit' && saveChanges && contentToSave !== undefined) {
            const newContent = contentToSave;
            if (newContent !== this.data.content) {
                this.data.content = newContent;
                this.callbacks.onContentChange?.(this.data.id, this.data.content);
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

    public switchToEditMode(): void {
         if (this.mode === 'edit') {
            this.editorInstance?.focus();
            return;
        }

        if (this.editorInstance) {
            console.warn(`Found existing editor instance when switching to edit mode for ${this.data.id}. Attempting cleanup.`);
            try {
                tinymce.remove(this.editorInstance);
            } catch (e) { console.error("Error during cleanup remove:", e); }
            this.editorInstance = null;
        }

        this.mode = 'edit';
        console.log(`Switching ${this.data.id} to edit mode.`);
        if (this.loadTemplate('edit')) {
            this.populateEditContent(); // This will initialize TinyMCE
            this.bindEditModeEvents();
        } else {
             console.error("Failed to load edit template for section", this.data.id);
             this.mode = 'view'; // Revert mode if template fails
        }
    }

 
     /** Implement the abstract method from BaseSection */
     protected resizeContentForFullscreen(isEntering: boolean): void {
         console.log(`TextSection ${this.data.id}: Resizing content trigger. Is entering fullscreen: ${isEntering}`);
         // For TinyMCE with autoresize, explicit resizing is often not needed here.
         // The browser layout handles width changes, and autoresize handles height.
         // If specific resizing logic were needed (e.g., for a different editor), it would go here.
         // Example: this.editorInstance?.execCommand('mceAutoResize'); // Might force a recalc if needed
     }
}
