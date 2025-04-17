// components/Section.ts

import { Modal } from './Modal';

/**
 * Section types
 */
export type SectionType = 'text' | 'drawing' | 'plot';

/**
 * Section data interface
 */
export interface SectionData {
  id: string;
  type: SectionType;
  title: string;
  content: string;
  order: number;
}

/**
 * Callback interface for section events
 */
export interface SectionCallbacks {
  onDelete?: (sectionId: string) => void;
  onMoveUp?: (sectionId: string) => void;
  onMoveDown?: (sectionId: string) => void;
  onTitleChange?: (sectionId: string, newTitle: string) => void;
  onContentChange?: (sectionId: string, newContent: string) => void;
}

/**
 * Manages an individual document section
 */
export class Section {
  private data: SectionData;
  private element: HTMLElement;
  private callbacks: SectionCallbacks;
  private modal: Modal;
  
  // DOM elements within the section
  private titleElement: HTMLElement | null;
  private contentElement: HTMLElement | null;
  private deleteButton: HTMLElement | null;
  private moveUpButton: HTMLElement | null;
  private moveDownButton: HTMLElement | null;
  private settingsButton: HTMLElement | null;
  private llmButton: HTMLElement | null;
  
  /**
   * Create a new section
   */
  constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
    this.data = data;
    this.element = element;
    this.callbacks = callbacks;
    this.modal = Modal.getInstance();
    
    // Find DOM elements
    this.titleElement = element.querySelector('.section-title');
    this.contentElement = element.querySelector('.section-content');
    this.deleteButton = element.querySelector('.section-delete');
    this.moveUpButton = element.querySelector('.section-move-up');
    this.moveDownButton = element.querySelector('.section-move-down');
    this.settingsButton = element.querySelector('.section-settings');
    this.llmButton = element.querySelector('.section-ai');
    
    this.bindEvents();
    this.initializeContent();
  }
  
  /**
   * Bind event listeners for this section
   */
  private bindEvents(): void {
    // Title editing
    if (this.titleElement) {
      this.titleElement.addEventListener('click', this.startTitleEdit.bind(this));
    }
    
    // Delete button
    if (this.deleteButton) {
      this.deleteButton.addEventListener('click', () => {
        if (this.callbacks.onDelete) {
          this.callbacks.onDelete(this.data.id);
        }
      });
    }
    
    // Move up button
    if (this.moveUpButton) {
      this.moveUpButton.addEventListener('click', () => {
        if (this.callbacks.onMoveUp) {
          this.callbacks.onMoveUp(this.data.id);
        }
      });
    }
    
    // Move down button
    if (this.moveDownButton) {
      this.moveDownButton.addEventListener('click', () => {
        if (this.callbacks.onMoveDown) {
          this.callbacks.onMoveDown(this.data.id);
        }
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
  
  /**
   * Initialize content based on section type
   */
  private initializeContent(): void {
    if (!this.contentElement) return;
    
    // Clear current content
    const placeholder = this.contentElement.querySelector('.section-type-placeholder');
    if (placeholder) {
      placeholder.remove();
    }
    
    // Add type-specific content
    switch (this.data.type) {
      case 'text':
        this.initializeTextContent();
        break;
        
      case 'drawing':
        this.initializeDrawingContent();
        break;
        
      case 'plot':
        this.initializePlotContent();
        break;
    }
  }
  
  /**
   * Initialize content for text section type
   */
  private initializeTextContent(): void {
    if (!this.contentElement) return;
    
    // Create editable content area
    const textArea = document.createElement('div');
    textArea.className = 'min-h-[100px] border border-gray-200 dark:border-gray-700 rounded p-4 prose dark:prose-invert max-w-none';
    textArea.setAttribute('contenteditable', 'true');
    textArea.innerHTML = this.data.content || '<p>Click to start editing...</p>';
    
    // Add content change event
    textArea.addEventListener('blur', () => {
      const newContent = textArea.innerHTML;
      if (newContent !== this.data.content) {
        this.data.content = newContent;
        if (this.callbacks.onContentChange) {
          this.callbacks.onContentChange(this.data.id, newContent);
        }
      }
    });
    
    // Add to content element
    this.contentElement.appendChild(textArea);
  }
  
  /**
   * Initialize content for drawing section type
   */
  private initializeDrawingContent(): void {
    if (!this.contentElement) return;
    
    const drawingArea = document.createElement('div');
    drawingArea.className = 'min-h-[300px] border border-gray-200 dark:border-gray-700 rounded p-4 flex items-center justify-center';
    drawingArea.innerHTML = '<p class="text-gray-400 dark:text-gray-500">Drawing tools will be available soon</p>';
    
    this.contentElement.appendChild(drawingArea);
  }
  
  /**
   * Initialize content for plot section type
   */
  private initializePlotContent(): void {
    if (!this.contentElement) return;
    
    const plotArea = document.createElement('div');
    plotArea.className = 'min-h-[300px] border border-gray-200 dark:border-gray-700 rounded p-4 flex items-center justify-center';
    plotArea.innerHTML = '<p class="text-gray-400 dark:text-gray-500">Chart tools will be available soon</p>';
    
    this.contentElement.appendChild(plotArea);
  }
  
  /**
   * Start title editing
   */
  private startTitleEdit(e: Event): void {
    e.preventDefault();
    e.stopPropagation();
    
    if (!this.titleElement) return;
    
    // Create an input element
    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'px-2 py-1 w-full border border-gray-300 dark:border-gray-600 dark:bg-gray-700 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:text-white';
    input.value = this.data.title;
    
    // Replace title with input
    const originalText = this.titleElement.textContent;
    this.titleElement.textContent = '';
    this.titleElement.appendChild(input);
    
    // Focus input
    input.focus();
    input.select();
    
    // Handle input blur (save)
    const handleSave = () => {
      const newTitle = input.value.trim();
      
      // Only update if there's actual content
      if (newTitle) {
        this.titleElement!.textContent = newTitle;
        this.data.title = newTitle;
        
        if (this.callbacks.onTitleChange) {
          this.callbacks.onTitleChange(this.data.id, newTitle);
        }
      } else {
        // Restore original title if empty
        this.titleElement!.textContent = originalText;
      }
      
      // Remove events
      input.removeEventListener('blur', handleSave);
      input.removeEventListener('keydown', handleKeyDown);
    };
    
    // Handle keydown events
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        input.blur(); // Trigger save on Enter
      } else if (e.key === 'Escape') {
        e.preventDefault();
        this.titleElement!.textContent = originalText; // Restore original title
        
        // Remove events
        input.removeEventListener('blur', handleSave);
        input.removeEventListener('keydown', handleKeyDown);
      }
    };
    
    // Add event listeners
    input.addEventListener('blur', handleSave);
    input.addEventListener('keydown', handleKeyDown);
  }
  
  /**
   * Open section settings modal
   */
  private openSettings(): void {
    // This would show a settings modal when implemented
    console.log('Section settings clicked for', this.data.id);
    
    // Placeholder until settings modal is implemented
    alert(`Settings for section: ${this.data.title} (ID: ${this.data.id})`);
  }
  
  /**
   * Open LLM dialog for this section
   */
  private openLlmDialog(): void {
    this.modal.show('llm-dialog', {
      sectionId: this.data.id,
      sectionType: this.data.type,
      sectionTitle: this.data.title
    });
    
    // Update the current section display in the LLM dialog
    const currentSectionElement = document.getElementById('llm-current-section');
    if (currentSectionElement) {
      currentSectionElement.textContent = this.data.title;
    }
  }
  
  /**
   * Update the section number
   */
  public updateNumber(number: number): void {
    this.data.order = number;
    
    const numberElement = this.element.querySelector('.section-number');
    if (numberElement) {
      numberElement.textContent = `${number}.`;
    }
  }
  
  /**
   * Get the section data
   */
  public getData(): SectionData {
    return { ...this.data };
  }
  
  /**
   * Update the section data
   */
  public update(newData: Partial<SectionData>): void {
    // Update data object
    this.data = { ...this.data, ...newData };
    
    // Update title if needed
    if (newData.title && this.titleElement) {
      this.titleElement.textContent = newData.title;
    }
    
    // Update content if needed and different type
    if ((newData.type && newData.type !== this.data.type) || newData.content) {
      if (this.contentElement) {
        this.contentElement.innerHTML = '';
        this.initializeContent();
      }
    }
  }
}
