/**
 * Search Engine Dashboard - Vue.js 3 Application
 * 
 * Provides real-time search, filtering, sorting, and pagination
 * for content visualization.
 */

const app = Vue.createApp({
    // Use custom delimiters to avoid conflict with Go templates
    delimiters: ['${', '}'],

    data() {
        return {
            // Content data
            contents: [],

            // Search & Filter state
            query: '',
            sortBy: '',
            sortOrder: 'desc',
            typeFilter: 'all',

            // Pagination state
            page: 1,
            pageSize: 5,
            total: 0,
            totalPages: 0,

            // UI state
            loading: false,
            syncing: false,
            error: null,

            // Debounce timer
            debounceTimer: null
        };
    },

    computed: {
        /**
         * Returns true if there is a next page available.
         * @returns {boolean}
         */
        hasNextPage() {
            return this.page < this.totalPages;
        },

        /**
         * Returns true if there is a previous page available.
         * @returns {boolean}
         */
        hasPrevPage() {
            return this.page > 1;
        }
    },

    methods: {
        /**
         * Fetches contents from the API with current filters and pagination.
         * Updates the contents list and pagination metadata.
         */
        async fetchContents() {
            this.loading = true;
            this.error = null;

            try {
                // Build query params
                const params = new URLSearchParams({
                    page: this.page.toString(),
                    page_size: this.pageSize.toString()
                });

                // Add search query if present
                if (this.query.trim()) {
                    params.set('q', this.query.trim());
                }

                // Add sort parameter
                if (this.sortBy) {
                    params.set('sort_by', this.sortBy);
                }

                // Add sort order parameter
                if (this.sortOrder) {
                    params.set('sort_order', this.sortOrder);
                }

                // Add type filter if not 'all'
                if (this.typeFilter && this.typeFilter !== 'all') {
                    params.set('type', this.typeFilter);
                }

                const response = await fetch(`/api/v1/contents?${params}`);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();

                // Update state
                this.contents = data.contents || [];
                this.total = data.pagination?.total || 0;
                this.totalPages = data.pagination?.total_pages || 0;
                this.page = data.pagination?.page || 1;

            } catch (err) {
                console.error('Failed to fetch contents:', err);
                this.error = err.message;
                this.contents = [];
            } finally {
                this.loading = false;
            }
        },

        /**
         * Debounced fetch to avoid excessive API calls during typing.
         * Waits 300ms after the last keystroke before fetching.
         */
        debouncedFetch() {
            clearTimeout(this.debounceTimer);
            this.debounceTimer = setTimeout(() => {
                this.page = 1; // Reset to first page on new search
                this.fetchContents();
            }, 300);
        },

        /**
         * Navigates to the next page if available.
         */
        nextPage() {
            if (this.hasNextPage) {
                this.page++;
                this.fetchContents();
                this.scrollToTop();
            }
        },

        /**
         * Navigates to the previous page if available.
         */
        prevPage() {
            if (this.hasPrevPage) {
                this.page--;
                this.fetchContents();
                this.scrollToTop();
            }
        },

        /**
         * Scrolls the page to the top smoothly.
         */
        scrollToTop() {
            window.scrollTo({ top: 0, behavior: 'smooth' });
        },

        /**
         * Formats an ISO date string to a localized date.
         * @param {string} dateStr - ISO date string
         * @returns {string} Formatted date
         */
        formatDate(dateStr) {
            if (!dateStr) return '-';

            try {
                const date = new Date(dateStr);
                return date.toLocaleDateString('tr-TR', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric'
                });
            } catch {
                return dateStr;
            }
        },

        /**
         * Triggers manual sync from all providers.
         */
        async syncProviders() {
            this.syncing = true;
            try {
                const response = await fetch('/api/v1/admin/sync', {
                    method: 'POST'
                });

                if (!response.ok) {
                    throw new Error(`Sync failed: ${response.statusText}`);
                }

                const data = await response.json();
                console.log('Sync completed:', data);

                // Refresh content list after sync
                await this.fetchContents();

                // Reload page to update total count in header
                window.location.reload();
            } catch (err) {
                console.error('Sync failed:', err);
                alert('Sync failed: ' + err.message);
            } finally {
                this.syncing = false;
            }
        }
    },

    watch: {
        /**
         * When sort field changes, reset to page 1 and fetch.
         */
        sortBy() {
            this.page = 1;
            this.fetchContents();
        },

        /**
         * When sort order changes, reset to page 1 and fetch.
         */
        sortOrder() {
            this.page = 1;
            this.fetchContents();
        },

        /**
         * When type filter changes, reset to page 1 and fetch.
         */
        typeFilter() {
            this.page = 1;
            this.fetchContents();
        }
    },

    /**
     * Lifecycle hook - fetch contents on mount.
     */
    mounted() {
        this.fetchContents();
    }
});

// Mount the app
app.mount('#app');
