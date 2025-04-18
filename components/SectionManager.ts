// components/SectionManager.ts

import { Modal } from './Modal';
// Import the BaseSection and derived classes
import { BaseSection } from './BaseSection';
import { TextSection } from './TextSection';
import { DrawingSection } from './DrawingSection';
import { PlotSection } from './PlotSection';
// Import types and interfaces (ensure SectionData/Callbacks are moved or accessible)
import { DocumentSection, SectionType, TextContent, DrawingContent, PlotContent, SectionData, SectionCallbacks } from './types'; // Assuming interfaces moved here


/**
 * Manages document sections using the BaseSection hierarchy.
 */
export class SectionManager {
  // Now using BaseSection type for the map
  private sections: Map<string, BaseSection> = new Map();
  private sectionData: SectionData[] = [];
  private nextSectionId: number = 1;
  private sectionsContainer: HTMLElement | null;
  private tocList: HTMLElement | null;
  private emptyStateEl: HTMLElement | null;
  private sectionTemplate: HTMLElement | null; // The main <div id="section-template">
  private tocItemTemplate: HTMLElement | null;
  private modal: Modal;

  // Predefined title suggestions (keep as before)
  private static readonly TEXT_TITLES: string[] = [ /* ... keep same content ... */ ];
  private static readonly DRAWING_TITLES: string[] = [ /* ... keep same content ... */ ];
  private static readonly PLOT_TITLES: string[] = [ /* ... keep same content ... */ ];

  private static getRandomTitle(type: SectionType): string {
    const titles = type === 'drawing' ? this.DRAWING_TITLES : type === 'plot' ? this.PLOT_TITLES : this.TEXT_TITLES;
    const randomIndex = Math.floor(Math.random() * titles.length);
    return titles[randomIndex] || `New ${type.charAt(0).toUpperCase() + type.slice(1)} Section`; // Fallback
  }

  constructor() {
    this.sectionsContainer = document.getElementById('sections-container');
    this.tocList = document.getElementById('toc-list');
    this.emptyStateEl = document.getElementById('empty-state');
    this.sectionTemplate = document.getElementById('section-template');
    this.tocItemTemplate = document.querySelector('.toc-item-template');
    this.modal = Modal.getInstance();

    // Ensure the main section template is present
     if (!this.sectionTemplate) {
       console.error("CRITICAL: Main section template (#section-template) not found!");
       // Optionally throw an error or disable section creation
       return;
     }
     // Ensure the template is hidden initially in the DOM if not already handled by CSS/HTML
     this.sectionTemplate.classList.add('hidden');


    this.bindEvents();
    this.handleEmptyState(); // Initial check
  }

  /** Bind event listeners for section management */
  private bindEvents(): void {
    // Bind add section buttons (logic remains similar)
    document.addEventListener('click', (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const addSectionBtn = target.closest('#create-first-section, #add-section-btn, .add-section-button');

      if (addSectionBtn) {
        e.preventDefault();
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

    // Bind section type selection (logic remains similar)
     document.addEventListener('click', (e: MouseEvent) => {
       const target = e.target as HTMLElement;
       const sectionTypeOption = target.closest('.section-type-option, button.section-type-option');

       if (sectionTypeOption && this.modal.getCurrentTemplate() === 'section-type-selector') {
         e.preventDefault();
         let sectionType: SectionType = 'text';
         const typeText = sectionTypeOption.querySelector('span')?.textContent?.trim().toLowerCase() || '';

         if (typeText === 'drawing') sectionType = 'drawing';
         else if (typeText === 'plot') sectionType = 'plot';

         const modalContent = this.modal.getContentElement();
         // Retrieve data attribute directly from the modal instance's stored data if possible,
         // otherwise fallback to DOM dataset
         const modalData = this.modal.getCurrentData();
         const insertAfterId = modalData?.insertAfterId || modalContent?.dataset.insertAfterId || null;


         this.createSection(sectionType, insertAfterId);
         this.modal.hide();
       }
     });
  }

  /** Open section type selector modal */
  private openSectionTypeSelector(insertAfterId: string | null = null): void {
    // Pass insertAfterId as data to the modal
    this.modal.show('section-type-selector', { insertAfterId: insertAfterId || '' });
    console.log('Opening section type selector modal', { insertAfterId });
  }

  /**
   * Create a new section instance based on the type.
   * Handles DOM creation, data storage, sorting, and updates.
   */
  private createSection(
    type: SectionType,
    insertAfterId: string | null = null,
    // Optional parameters for loading data
    initialData?: Partial<SectionData>
  ): BaseSection | null { // Return the created section instance or null

    if (!this.sectionsContainer || !this.sectionTemplate) {
      console.error("Cannot create section: Container or template missing.");
      return null;
    }

    // --- Section Data Creation ---
    const sectionId = initialData?.id || `section-${this.nextSectionId++}`;
    let sectionOrder: number;
    const providedOrder = initialData?.order;

    if (providedOrder !== undefined) {
      sectionOrder = providedOrder;
    } else if (insertAfterId) {
      const insertAfterIndex = this.sectionData.findIndex(s => s.id === insertAfterId);
      if (insertAfterIndex !== -1) {
        sectionOrder = this.sectionData[insertAfterIndex].order + 0.5; // Temporary order for sorting
      } else {
        sectionOrder = this.sectionData.length; // Append to end if insertAfterId not found
      }
    } else {
      sectionOrder = this.sectionData.length; // Append to end
    }

    // Define the core data, merging initialData if provided
    const sectionData: SectionData = {
      id: sectionId,
      type: type,
      title: initialData?.title || SectionManager.getRandomTitle(type),
      content: initialData?.content || this.getDefaultContent(type), // Get default content
      order: sectionOrder, // Will be normalized later
    };

    // --- DOM Element Creation ---
    const sectionEl = this.sectionTemplate.cloneNode(true) as HTMLElement;
    sectionEl.classList.remove('hidden');
    sectionEl.id = sectionId; // Use the final section ID
    sectionEl.dataset.sectionId = sectionId;
    sectionEl.dataset.sectionType = type;

    // Find the correct insertion point in the DOM *before* adding to data array
    let insertBeforeEl: HTMLElement | null = null;
    if (providedOrder === undefined && insertAfterId) { // Only use DOM insertion for user actions
       const insertAfterEl = this.sectionsContainer.querySelector(`[data-section-id="${insertAfterId}"]`);
       if (insertAfterEl) {
         insertBeforeEl = insertAfterEl.nextElementSibling as HTMLElement | null;
       }
    }

    // Add element to container at the correct position
    if (insertBeforeEl) {
      this.sectionsContainer.insertBefore(sectionEl, insertBeforeEl);
    } else {
       // Append to end if no specific position or loading data
       this.sectionsContainer.appendChild(sectionEl);
    }


    // --- Section Instance Creation ---
    let sectionInstance: BaseSection;
    const callbacks: SectionCallbacks = {
      onDelete: this.deleteSection.bind(this),
      onMoveUp: this.moveSectionUp.bind(this),
      onMoveDown: this.moveSectionDown.bind(this),
      onTitleChange: this.updateSectionTitle.bind(this),
      onContentChange: this.updateSectionContent.bind(this)
    };

    switch (type) {
      case 'text':
        sectionInstance = new TextSection(sectionData, sectionEl, callbacks);
        break;
      case 'drawing':
        sectionInstance = new DrawingSection(sectionData, sectionEl, callbacks);
        break;
      case 'plot':
        sectionInstance = new PlotSection(sectionData, sectionEl, callbacks);
        break;
      default:
        console.warn(`Unhandled section type "${type}" during creation. Defaulting to Text.`);
        sectionData.type = 'text'; // Correct the type
        sectionInstance = new TextSection(sectionData, sectionEl, callbacks);
    }

    // --- Update State and UI ---
    // Add to sections map and data array
    this.sections.set(sectionId, sectionInstance);
    this.sectionData.push(sectionData);

    // Don't sort/renumber if loading multiple sections, do it once at the end
    if (providedOrder === undefined) {
       this.normalizeOrdersAndRenumber();
       this.updateTableOfContents(); // Update TOC immediately for single adds
       this.handleEmptyState();
    }

    return sectionInstance; // Return the created instance
  }

  /** Returns appropriate default content for a section type */
  private getDefaultContent(type: SectionType): SectionData['content'] {
    switch (type) {
      case 'text': return ''; // Empty paragraph or string
      case 'drawing': return { format: 'placeholder_drawing', data: {} };
      case 'plot': return { format: 'placeholder_plot', data: {} };
      default: return '';
    }
  }

  /**
   * Loads multiple sections from document data.
   * Clears existing sections first.
   */
  public loadSections(sectionsData: DocumentSection[]): void {
    if (!this.sectionsContainer) return;

    console.log("Loading sections:", sectionsData.length);

    // 1. Clear existing sections
    this.clearAllSections();

    let maxIdNum = 0;

    // 2. Create sections from data (ensure sorted by order first)
    // Use a stable sort if order numbers might be identical temporarily
    sectionsData
      .sort((a, b) => a.order - b.order)
      .forEach(data => {
        // Use createSection with initialData, providing the order
        const createdSection = this.createSection(data.type, null, data);

        // Track max ID number to set nextSectionId correctly
         if (createdSection) {
           const idNumMatch = data.id.match(/\d+$/);
           if (idNumMatch) {
             maxIdNum = Math.max(maxIdNum, parseInt(idNumMatch[0], 10));
           }
         }
      });

    // Set nextSectionId higher than any loaded ID
    this.nextSectionId = maxIdNum + 1;

    // 3. Final updates after all sections are loaded
    this.normalizeOrdersAndRenumber(); // Renumber correctly based on final order
    this.updateTableOfContents();
    this.handleEmptyState();
    console.log("Sections loaded and rendered.");
  }

   /** Clears all section elements, data, and map entries */
   private clearAllSections(): void {
     if (!this.sectionsContainer) return;
     this.sectionsContainer.innerHTML = ''; // Clear DOM
     this.sections.clear(); // Clear map
     this.sectionData = []; // Clear data array
     this.nextSectionId = 1; // Reset ID counter
     this.updateTableOfContents(); // Clear TOC
     this.handleEmptyState(); // Show empty state if needed
   }

   /** Normalize order values and update numbering */
   private normalizeOrdersAndRenumber(): void {
     // Sort based on current order values (including temporary 0.5 increments)
     this.sectionData.sort((a, b) => a.order - b.order);

     // Assign new sequential integer orders and update DOM/instances
     this.sectionData.forEach((sectionDataItem, index) => {
       sectionDataItem.order = index + 1; // Normalize order in data array
       const sectionInstance = this.sections.get(sectionDataItem.id);
       if (sectionInstance) {
         sectionInstance.updateNumber(index + 1); // Update instance & DOM number
       }
     });
   }

  /** Delete a section */
  private deleteSection(sectionId: string): void {
    if (!confirm('Are you sure you want to delete this section?')) return;

    const sectionInstance = this.sections.get(sectionId);
    if (sectionInstance) {
      // Remove from DOM
      sectionInstance.removeElement();

      // Remove from data array and map
      this.sectionData = this.sectionData.filter(s => s.id !== sectionId);
      this.sections.delete(sectionId);

      // Update numbering and TOC
      this.normalizeOrdersAndRenumber();
      this.updateTableOfContents();
      this.handleEmptyState();
    }
  }

  /** Move a section up */
  private moveSectionUp(sectionId: string): void {
    const sectionIndex = this.sectionData.findIndex(s => s.id === sectionId);
    if (sectionIndex <= 0) return; // Already at the top

    // Adjust order slightly to ensure it moves up after sort
    this.sectionData[sectionIndex].order -= 1.5;

    this.reorderAndRenumber();
  }

  /** Move a section down */
  private moveSectionDown(sectionId: string): void {
    const sectionIndex = this.sectionData.findIndex(s => s.id === sectionId);
    if (sectionIndex === -1 || sectionIndex >= this.sectionData.length - 1) return; // Invalid or at bottom

    // Adjust order slightly to ensure it moves down after sort
    this.sectionData[sectionIndex].order += 1.5;

    this.reorderAndRenumber();
  }

  /** Reorders sections in DOM based on data, renumbers, and updates TOC */
  private reorderAndRenumber(): void {
     this.normalizeOrdersAndRenumber(); // Sorts data, updates orders and DOM numbers
     this.reorderSectionsInDOM();  // Physically moves elements in DOM
     this.updateTableOfContents();
  }


  /** Reorder section elements in the DOM based on the sorted sectionData */
  private reorderSectionsInDOM(): void {
    if (!this.sectionsContainer) return;
    // Assumes sectionData is already sorted by order
    this.sectionData.forEach(data => {
      const sectionEl = document.getElementById(data.id);
      if (sectionEl) {
        this.sectionsContainer!.appendChild(sectionEl); // Re-append in sorted order
      }
    });
  }

  /** Update section title in data and TOC */
  private updateSectionTitle(sectionId: string, newTitle: string): void {
    const sectionData = this.sectionData.find(s => s.id === sectionId);
    if (sectionData) {
      sectionData.title = newTitle;
      this.updateTableOfContents(); // Update TOC immediately
    }
  }

  /** Update section content in data (called by BaseSection on save) */
  private updateSectionContent(sectionId: string, newContent: SectionData['content']): void {
    const sectionData = this.sectionData.find(s => s.id === sectionId);
    if (sectionData) {
      sectionData.content = newContent;
      // No UI update needed here as the section handles its own view rendering
      console.log(`Content updated in SectionManager data for ${sectionId}`);
    }
  }

  /** Update table of contents */
  private updateTableOfContents(): void {
    if (!this.tocList || !this.tocItemTemplate) return;

    // Clear current TOC (except template)
    const currentItems = this.tocList.querySelectorAll('li:not(.toc-item-template)');
    currentItems.forEach(item => item.remove());

    // Add TOC items for each section (using sorted data)
    this.sectionData.forEach(section => {
      const tocItem = this.tocItemTemplate?.cloneNode(true) as HTMLElement;
      tocItem.classList.remove('hidden', 'toc-item-template');

      const link = tocItem.querySelector('a');
      const numberSpan = tocItem.querySelector('.toc-section-number');
      const titleSpan = tocItem.querySelector('.toc-section-title');

      if (link) link.href = `#${section.id}`; // Use simple hash link
      if (numberSpan) numberSpan.textContent = `${section.order}.`;
      if (titleSpan) titleSpan.textContent = section.title;

      // Click handler for smooth scrolling (optional)
      link?.addEventListener('click', (e) => {
         e.preventDefault();
         document.getElementById(section.id)?.scrollIntoView({ behavior: 'smooth' });
         // Optionally close mobile sidebar
         document.querySelector('.md\\:w-64')?.classList.add('hidden'); // Example
       });

      this.tocList?.appendChild(tocItem);
    });
  }

  /** Handle empty state visibility */
  private handleEmptyState(): void {
    if (!this.emptyStateEl || !this.sectionsContainer) return;
    if (this.sectionData.length === 0) {
      this.emptyStateEl.classList.remove('hidden');
    } else {
      this.emptyStateEl.classList.add('hidden');
    }
  }

  /** Get all section data formatted for the document model */
  public getDocumentSections(): DocumentSection[] {
     // Ensure data is sorted before retrieving
    return this.sectionData
      .sort((a, b) => a.order - b.order) // Stable sort just in case
      .map(sectionDataItem => {
        const sectionInstance = this.sections.get(sectionDataItem.id);
        if (sectionInstance) {
          return sectionInstance.getDocumentData();
        }
        console.warn(`Section instance not found for ID: ${sectionDataItem.id} during save.`);
        // Fallback: return data directly from manager (might be stale if save happens during edit)
         return {
           id: sectionDataItem.id,
           title: sectionDataItem.title,
           order: sectionDataItem.order,
           type: sectionDataItem.type,
           content: sectionDataItem.content
         } as DocumentSection; // Cast needed
      });
  }

  /** Initialize the component */
  public static init(): SectionManager {
    return new SectionManager();
  }
}
