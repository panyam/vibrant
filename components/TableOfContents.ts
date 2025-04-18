// components/TableOfContents.ts

import { ToastManager } from './ToastManager'; // Optional: for feedback if needed

// Define the structure of the data expected by the TOC
export interface TocItemInfo {
    id: string;
    title: string;
    order: number;
}

/**
 * Manages the Table of Contents UI component.
 * Renders the list of sections and handles interactions within the TOC sidebar.
 */
export class TableOfContents {
    private tocListElement: HTMLElement | null;
    private tocItemTemplateElement: HTMLElement | null;
    private addSectionButtonElement: HTMLElement | null;
    private closeSidebarButtonElement: HTMLElement | null;
    private sidebarElement: HTMLElement | null; // Reference to the sidebar container itself

    private currentTocItems: TocItemInfo[] = [];
    private onAddSectionClick: () => void; // Callback to trigger adding a section

    constructor(options: { onAddSectionClick: () => void }) {
        this.onAddSectionClick = options.onAddSectionClick;

        // Find elements within the main document structure related to TOC
        this.tocListElement = document.getElementById('toc-list');
        this.tocItemTemplateElement = document.querySelector('.toc-item-template');
        this.addSectionButtonElement = document.getElementById('add-section-btn');
        this.closeSidebarButtonElement = document.getElementById('close-sidebar');
        // Assuming the sidebar container has the '.md:w-64' class structure from index.html
        this.sidebarElement = document.querySelector('.w-full.md\\:w-64');

        if (!this.tocListElement || !this.tocItemTemplateElement || !this.addSectionButtonElement || !this.closeSidebarButtonElement || !this.sidebarElement) {
            console.error("TableOfContents: Could not find all required DOM elements. Check IDs and selectors.");
            // Optionally throw an error or disable functionality
        }

        this.bindEvents();
    }

    /** Binds event listeners for TOC controls. */
    private bindEvents(): void {
        // Use optional chaining in case elements weren't found
        this.addSectionButtonElement?.addEventListener('click', this.handleAddSectionClick.bind(this));
        this.closeSidebarButtonElement?.addEventListener('click', this.handleCloseSidebarClick.bind(this));
    }

    /** Handles the click on the "Add New Section" button. */
    private handleAddSectionClick(): void {
        console.log("TOC: Add Section button clicked.");
        this.onAddSectionClick(); // Trigger the callback provided during instantiation
        // Optionally close sidebar on mobile after clicking add
        if (this.isMobileLayout()) {
            this.hideSidebar();
        }
    }

    /** Handles the click on the "Close Sidebar" button (mobile only). */
    private handleCloseSidebarClick(): void {
        console.log("TOC: Close Sidebar button clicked.");
        this.hideSidebar();
    }

    /** Hides the sidebar element. */
    private hideSidebar(): void {
        this.sidebarElement?.classList.add('hidden');
        this.sidebarElement?.classList.remove('block'); // Ensure responsive classes are toggled correctly
    }

    /** Checks if the application is likely in a mobile layout. */
    private isMobileLayout(): boolean {
        const mobileMenuButton = document.getElementById('mobile-menu-button');
        return mobileMenuButton ? mobileMenuButton.offsetParent !== null : false;
    }

    /**
     * Public method to update the TOC with new section data.
     * @param sectionsInfo - An array of simplified section information.
     */
    public update(sectionsInfo: TocItemInfo[]): void {
        // Sort the incoming data by order just in case
        this.currentTocItems = [...sectionsInfo].sort((a, b) => a.order - b.order);
        this.render();
    }

    /** Clears and re-renders the TOC list based on currentTocItems. */
    private render(): void {
        if (!this.tocListElement || !this.tocItemTemplateElement) {
            return; // Exit if elements are missing
        }

        // Assign to local constants after the null check for type safety within closures
        const tocListElement = this.tocListElement;
        const tocItemTemplateElement = this.tocItemTemplateElement; // <-- FIX: Use this constant

        // Clear current TOC items (excluding the template)
        const currentItems = this.tocListElement.querySelectorAll('li:not(.toc-item-template)');
        currentItems.forEach(item => item.remove());

        // Render new items
        this.currentTocItems.forEach(section => {
            const tocItem = tocItemTemplateElement!.cloneNode(true) as HTMLElement; // Non-null assertion ok due to check above
            tocItem.classList.remove('hidden', 'toc-item-template');

            const link = tocItem.querySelector('a');
            const numberSpan = tocItem.querySelector('.toc-section-number');
            const titleSpan = tocItem.querySelector('.toc-section-title');

            if (link) {
                link.href = `#${section.id}`;
                const clickHandler = (e: MouseEvent) => this.handleTocLinkClick(e, section.id);
                // Remove previous listener before adding a new one
                // link.removeEventListener('click', this.handleTocLinkClick);
                // Bind the handler, passing the sectionId
                link.addEventListener('click', (e) => this.handleTocLinkClick(e, section.id));
            }
            if (numberSpan) numberSpan.textContent = `${section.order}.`;
            if (titleSpan) titleSpan.textContent = section.title;

            tocListElement.appendChild(tocItem);
        });
    }

    /**
     * Handles clicks on individual TOC links.
     * Scrolls to the section and handles sidebar visibility on mobile.
     */
    private handleTocLinkClick(event: MouseEvent, sectionId: string): void {
        event.preventDefault();
        const targetElement = document.getElementById(sectionId);

        if (targetElement) {
            targetElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
        } else {
            console.warn(`TOC: Target element with ID ${sectionId} not found for scrolling.`);
        }

        // Close sidebar only if in mobile layout
        if (this.isMobileLayout()) {
            this.hideSidebar();
        }
    }

    /**
     * Static initializer for the component.
     */
    public static init(options: { onAddSectionClick: () => void }): TableOfContents {
        return new TableOfContents(options);
    }
}
