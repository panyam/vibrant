// components/SectionManager.ts

import { Modal } from './Modal';
import { Section, SectionData, SectionType } from './Section';

/**
 * Manages document sections
 */
export class SectionManager {
  private sections: Map<string, Section> = new Map();
  private sectionData: SectionData[] = [];
  private nextSectionId: number = 1;
  private sectionsContainer: HTMLElement | null;
  private tocList: HTMLElement | null;
  private emptyStateEl: HTMLElement | null;
  private sectionTemplate: HTMLElement | null;
  private tocItemTemplate: HTMLElement | null;
  private modal: Modal;

  constructor() {
    // Get container elements
    this.sectionsContainer = document.getElementById('sections-container');
    this.tocList = document.getElementById('toc-list');
    this.emptyStateEl = document.getElementById('empty-state');
    
    // Get templates
    this.sectionTemplate = document.getElementById('section-template');
    this.tocItemTemplate = document.querySelector('.toc-item-template');
    
    // Get modal instance
    this.modal = Modal.getInstance();
    
    this.bindEvents();
  }

  /**
   * Bind event listeners for section management
   */
  private bindEvents(): void {
    // Bind add section buttons
    document.addEventListener('click', (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const addSectionBtn = target.closest('#create-first-section, #add-section-btn, .add-section-button');
      
      if (addSectionBtn) {
        e.preventDefault();
        
        // Get the section after which to insert (null means at the end)
        let insertAfterId: string | null = null;
        if (addSectionBtn.classList.contains('add-section-button')) {
          const sectionEl = addSectionBtn.closest('[data-section-id]');
          if (sectionEl) {
            insertAfterId = (sectionEl as HTMLElement).dataset.sectionId || null;
          }
        }
        
        this.openSectionTypeSelector(insertAfterId);
      }
    });
    
    // Bind section type selection
    document.addEventListener('click', (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      // const sectionTypeOption = target.closest('.section-type-option');
      const sectionTypeOption = target.closest('.section-type-option, button.section-type-option');

      
      if (sectionTypeOption && this.modal.getCurrentTemplate() === 'section-type-selector') {
        e.preventDefault();
        
        // Get section type
        let sectionType: SectionType = 'text';
        // const typeText = sectionTypeOption.querySelector('span')?.textContent?.trim().toLowerCase();
        const typeText = sectionTypeOption.querySelector('span')?.textContent?.trim().toLowerCase() || '';
        
        if (typeText === 'drawing') {
          sectionType = 'drawing';
        } else if (typeText === 'plot') {
          sectionType = 'plot';
        }
        
        // Get insert after ID from modal data
        const modalContent = this.modal.getContentElement();
        const insertAfterId = modalContent?.dataset.insertAfterId || null;
        
        // Create section
        this.createSection(sectionType, insertAfterId);
        
        // Close modal
        this.modal.hide();
      }
    });
  }

  /**
   * Open section type selector modal
   */
  private openSectionTypeSelector(insertAfterId: string | null = null): void {
    console.log('Available templates:', Array.from(document.querySelectorAll('[id$="-template"]')).map(el => el.id));
    this.modal.show('section-type-selector', { insertAfterId: insertAfterId || '' });
    console.log('Opening section type selector modal');
  }

  /**
   * Create a new section
   */
  private createSection(type: SectionType, insertAfterId: string | null = null): void {
    if (!this.sectionsContainer || !this.sectionTemplate) return;
    
    // Create section data
    const sectionId = `section-${this.nextSectionId++}`;
    const sectionTitle = `New ${type.charAt(0).toUpperCase() + type.slice(1)} Section`;
    
    // Determine order based on insert position
    let order = this.sectionData.length + 1;
    if (insertAfterId) {
      const insertAfterIndex = this.sectionData.findIndex(s => s.id === insertAfterId);
      if (insertAfterIndex !== -1) {
        order = this.sectionData[insertAfterIndex].order + 1;
        
        // Increment order of all sections after this one
        this.sectionData.forEach(section => {
          if (section.order >= order) {
            section.order++;
          }
        });
      }
    }
    
    // Create section object
    const sectionData: SectionData = {
      id: sectionId,
      type,
      title: sectionTitle,
      content: '',
      order
    };
    
    // Add to sections array
    this.sectionData.push(sectionData);
    
    // Sort sections by order
    this.sectionData.sort((a, b) => a.order - b.order);
    
    // Clone the template
    const sectionEl = this.sectionTemplate.cloneNode(true) as HTMLElement;
    sectionEl.classList.remove('hidden');
    sectionEl.id = sectionId;
    sectionEl.dataset.sectionId = sectionId;
    sectionEl.dataset.sectionType = type;
    
    // Set section number
    const sectionNumber = sectionEl.querySelector('.section-number');
    if (sectionNumber) {
      sectionNumber.textContent = `${order}.`;
    }
    
    // Add to container
    this.sectionsContainer.appendChild(sectionEl);
    // Add to container at the correct position
    if (insertAfterId) {
      const insertAfterEl = document.getElementById(insertAfterId);
      if (insertAfterEl && insertAfterEl.nextElementSibling) {
        this.sectionsContainer.insertBefore(sectionEl, insertAfterEl.nextElementSibling);
      } else {
        this.sectionsContainer.appendChild(sectionEl);
      }
    } else {
      this.sectionsContainer.appendChild(sectionEl);
    }
    
    // Create Section instance with callbacks
    const section = new Section(sectionData, sectionEl, {
      onDelete: this.deleteSection.bind(this),
      onMoveUp: this.moveSectionUp.bind(this),
      onMoveDown: this.moveSectionDown.bind(this),
      onTitleChange: this.updateSectionTitle.bind(this),
      onContentChange: this.updateSectionContent.bind(this)
    });
    
    // Add to sections map
    this.sections.set(sectionId, section);
    
    // Update section numbers
    this.updateSectionNumbers();
    
    // Update table of contents
    this.updateTableOfContents();
    
    // Handle empty state
    this.handleEmptyState();
  }

  /**
   * Delete a section
   */
  private deleteSection(sectionId: string): void {
    if (!this.sectionsContainer) return;
    
    // Confirm deletion
    if (!confirm('Are you sure you want to delete this section?')) {
      return;
    }
    
    // Remove from DOM
    const sectionEl = document.getElementById(sectionId);
    if (sectionEl) {
      sectionEl.remove();
    }
    
    // Remove from data
    this.sectionData = this.sectionData.filter(s => s.id !== sectionId);
    
    // Remove from sections map
    this.sections.delete(sectionId);
    
    // Update section numbers
    this.updateSectionNumbers();
    
    // Update table of contents
    this.updateTableOfContents();
    
    // Handle empty state
    this.handleEmptyState();
  }

  /**
   * Move a section up
   */
  private moveSectionUp(sectionId: string): void {
    const sectionIndex = this.sectionData.findIndex(s => s.id === sectionId);
    if (sectionIndex <= 0) return; // Already at the top
    
    // Swap with previous section
    const previousIndex = sectionIndex - 1;
    
    // Swap orders
    const temp = this.sectionData[sectionIndex].order;
    this.sectionData[sectionIndex].order = this.sectionData[previousIndex].order;
    this.sectionData[previousIndex].order = temp;
    
    // Sort by order
    this.sectionData.sort((a, b) => a.order - b.order);
    
    // Reorder in DOM
    this.reorderSectionsInDOM();
    
    // Update section numbers
    this.updateSectionNumbers();
    
    // Update table of contents
    this.updateTableOfContents();
  }

  /**
   * Move a section down
   */
  private moveSectionDown(sectionId: string): void {
    const sectionIndex = this.sectionData.findIndex(s => s.id === sectionId);
    if (sectionIndex === -1 || sectionIndex >= this.sectionData.length - 1) return; // Invalid or already at bottom
    
    // Swap with next section
    const nextIndex = sectionIndex + 1;
    
    // Swap orders
    const temp = this.sectionData[sectionIndex].order;
    this.sectionData[sectionIndex].order = this.sectionData[nextIndex].order;
    this.sectionData[nextIndex].order = temp;
    
    // Sort by order
    this.sectionData.sort((a, b) => a.order - b.order);
    
    // Reorder in DOM
    this.reorderSectionsInDOM();
    
    // Update section numbers
    this.updateSectionNumbers();
    
    // Update table of contents
    this.updateTableOfContents();
  }

  /**
   * Reorder sections in the DOM based on their order
   */
  private reorderSectionsInDOM(): void {
    if (!this.sectionsContainer) return;
    
    // Get ordered section elements
    const orderedSections = this.sectionData.map(data => {
      return document.getElementById(data.id);
    }).filter(el => el !== null) as HTMLElement[];
    
    // Reorder in DOM
    orderedSections.forEach(sectionEl => {
      this.sectionsContainer!.appendChild(sectionEl);
    });
  }

  /**
   * Update section title
   */
  private updateSectionTitle(sectionId: string, newTitle: string): void {
    // Update in data
    const sectionData = this.sectionData.find(s => s.id === sectionId);
    if (sectionData) {
      sectionData.title = newTitle;
    }
    
    // Update table of contents
    this.updateTableOfContents();
  }

  /**
   * Update section content
   */
  private updateSectionContent(sectionId: string, newContent: string): void {
    // Update in data
    const sectionData = this.sectionData.find(s => s.id === sectionId);
    if (sectionData) {
      sectionData.content = newContent;
    }
  }

  /**
   * Update section numbers
   */
  private updateSectionNumbers(): void {
    // Update order in data structure
    this.sectionData.forEach((section, index) => {
      section.order = index + 1;
      
      // Update section instance
      const sectionInstance = this.sections.get(section.id);
      if (sectionInstance) {
        sectionInstance.updateNumber(index + 1);
      }
    });
  }

  /**
   * Update table of contents
   */
  private updateTableOfContents(): void {
    if (!this.tocList || !this.tocItemTemplate) return;
    
    // Clear current TOC (except template)
    const currentItems = this.tocList.querySelectorAll('li:not(.toc-item-template)');
    currentItems.forEach(item => item.remove());
    
    // Add TOC items for each section
    this.sectionData.forEach(section => {
      // Clone template
      const tocItem = this.tocItemTemplate?.cloneNode(true) as HTMLElement;
      tocItem.classList.remove('hidden', 'toc-item-template');
      
      // Update content
      const link = tocItem.querySelector('a');
      const numberSpan = tocItem.querySelector('.toc-section-number');
      const titleSpan = tocItem.querySelector('.toc-section-title');
      
      if (link) {
        link.setAttribute('href', `#${section.id}`);
      }
      
      if (numberSpan) {
        numberSpan.textContent = `${section.order}.`;
      }
      
      if (titleSpan) {
        titleSpan.textContent = section.title;
      }
      
      // Add to TOC
      this.tocList?.appendChild(tocItem);
    });
  }

  /**
   * Handle empty state visibility
   */
  private handleEmptyState(): void {
    if (!this.emptyStateEl || !this.sectionsContainer) return;
    
    if (this.sectionData.length === 0) {
      // Show empty state, hide sections
      this.emptyStateEl.classList.remove('hidden');
    } else {
      // Hide empty state
      this.emptyStateEl.classList.add('hidden');
    }
  }

  /**
   * Initialize the component
   */
  public static init(): SectionManager {
    return new SectionManager();
  }
}
