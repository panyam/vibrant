// components/SectionManager.ts

import { Modal } from './Modal';
import { Section, SectionData } from './Section';
import { DocumentSection, SectionType, TextContent, DrawingContent, PlotContent } from './types'; // Import the types


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

  // Predefined title suggestions for different section types
  private static readonly TEXT_TITLES: string[] = [
    "Requirements", "Functional Requirements", "Non-Functional Requirements",
    "API Design", "Data Model", "High-Level Design", "Detailed Design",
    "Assumptions", "Security Considerations", "Deployment Strategy",
    "Monitoring & Alerting", "Future Considerations", "Capacity Planning",
    "System Interfaces", "User Scenarios"
  ];
  private static readonly DRAWING_TITLES: string[] = [
    "Architecture Overview", "System Components", "Data Flow Diagram",
    "Sequence Diagram", "Network Topology", "Component Interactions",
    "Deployment View", "API Interactions", "Database Schema",
    "High-Level Architecture", "User Flow"
  ];
  private static readonly PLOT_TITLES: string[] = [
    "Scalability Analysis", "Latency vs Throughput", "QPS Estimates",
    "Storage Projections", "Cost Analysis", "Performance Metrics",
    "Resource Utilization", "Traffic Estimation", "Data Growth",
    "Benchmark Results"
  ];

  /**
   * Selects a random title from the appropriate list based on section type.
   */
  private static getRandomTitle(type: SectionType): string {
    const titles = type === 'drawing' ? this.DRAWING_TITLES : type === 'plot' ? this.PLOT_TITLES : this.TEXT_TITLES;
    const randomIndex = Math.floor(Math.random() * titles.length);
    return titles[randomIndex] || `New ${type.charAt(0).toUpperCase() + type.slice(1)} Section`; // Fallback
  }


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
  private createSection(type: SectionType, insertAfterId: string | null = null, title?: string, id?: string, content?: string | DrawingContent | PlotContent, orderOverride?: number): void {
    if (!this.sectionsContainer || !this.sectionTemplate) return;
    
    // Create section data
     const sectionId = id || `section-${this.nextSectionId++}`; // Use provided ID or generate
     let sectionOrder = orderOverride ?? this.sectionData.length + 1; // Use provided order or calculate
 
     // --- Logic for inserting at specific position (only applies if orderOverride is not set) ---
     // Determine order based on insert position
     // let order = this.sectionData.length + 1; // Moved calculation above
     if (orderOverride === undefined && insertAfterId) {
         const insertAfterIndex = this.sectionData.findIndex(s => s.id === insertAfterId);
         if (insertAfterIndex !== -1) {
             sectionOrder = this.sectionData[insertAfterIndex].order + 1;
 
             // Increment order of all sections after this one
             this.sectionData.forEach(section => {
                 if (section.order >= sectionOrder) {
                     section.order++;
                 }
             });
         }
     }
     // --- End insertion logic ---

    const sectionTitle = SectionManager.getRandomTitle(type);
    
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
      content: content || '', // Use provided content or default
      order: sectionOrder
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
      sectionNumber.textContent = `${sectionOrder}.`;
    }
    
    // Add to container
    this.sectionsContainer.appendChild(sectionEl);
    // Add to container at the correct position
    // If loading (orderOverride is set), simple append is fine as reorder happens later.
    // Otherwise, use insertion logic.
    if (orderOverride === undefined && insertAfterId) {
      const insertAfterEl = this.sectionsContainer.querySelector(`[data-section-id="${insertAfterId}"]`);
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
    
    // Don't update numbers/TOC here if loading multiple sections, do it once at the end
    if (orderOverride === undefined) {
        this.updateSectionNumbers(); // Update numbers if adding single section
    }
    
    // Update table of contents
    this.updateTableOfContents();
    
    // Handle empty state
    this.handleEmptyState();
  }

 
   /**
    * Loads multiple sections from document data.
    * Clears existing sections first.
    */
   public loadSections(sectionsData: DocumentSection[]): void {
     if (!this.sectionsContainer) return;
 
     // 1. Clear existing sections
     this.sectionsContainer.innerHTML = ''; // Clear DOM
     this.sections.clear(); // Clear map
     this.sectionData = []; // Clear data array
     this.nextSectionId = 1; // Reset ID counter (will be updated below)
 
     let maxIdNum = 0;
 
     // 2. Create sections from data (ensure sorted by order first)
     sectionsData
         .sort((a, b) => a.order - b.order)
         .forEach(data => {
             // Use the core createSection logic, providing all data
             this.createSection(data.type, null, data.title, data.id, data.content, data.order);
 
             // Track max ID number to set nextSectionId correctly
             const idNumMatch = data.id.match(/\d+$/);
             if (idNumMatch) {
                 maxIdNum = Math.max(maxIdNum, parseInt(idNumMatch[0], 10));
             }
         });
 
     // Set nextSectionId higher than any loaded ID
     this.nextSectionId = maxIdNum + 1;
 
     // 3. Final updates after all sections are loaded
     this.updateSectionNumbers(); // Renumber correctly based on final order
     this.updateTableOfContents();
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
    
    // Re-append elements to the container in the new order (use ! assumption as check done above)
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
    * Get all section data formatted for the document model.
    */
   public getDocumentSections(): DocumentSection[] {
        return this.sectionData
            .sort((a, b) => a.order - b.order) // Ensure sorted by order
            .map(sectionDataItem => {
                const sectionInstance = this.sections.get(sectionDataItem.id);
                if (sectionInstance) {
                    return sectionInstance.getDocumentData();
                }
                // Fallback or error handling if instance not found (shouldn't happen ideally)
                console.warn(`Section instance not found for ID: ${sectionDataItem.id}`);
                return null; // Or a default representation
            }).filter(section => section !== null) as DocumentSection[]; // Filter out nulls
    }


  /**
   * Initialize the component
   */
  public static init(): SectionManager {
    return new SectionManager();
  }
}
