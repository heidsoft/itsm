I have identified the causes of the "map is not a function" error and the issues with the Tickets and Knowledge Base routes.

### Problem Analysis
1.  **Ticket Route Crash (`TicketList.tsx`)**:
    *   **Cause**: The `selectedTickets` state was recently changed from an array to a `Set<number>` (to optimize performance), but `TicketList.tsx` still treats it as an array in several places.
    *   **Specific Errors**:
        *   Calling `.map()` on a Set: `selectedTickets.map(...)` throws `TypeError: map is not a function`.
        *   Accessing `.length` on a Set: `selectedTickets.length` returns `undefined` (should be `.size`).

2.  **Knowledge Base Route Potential Crash (`ArticleList.tsx`)**:
    *   **Cause**: The component assumes `categories` and `tags` are always arrays. If the API returns `null`, `undefined`, or an object (due to backend changes or mock data issues), calling `.map()` will crash the page.

### Proposed Plan
I will apply the following fixes to ensure stability:

1.  **Fix `src/components/ticket/TicketList.tsx`**:
    *   Convert `selectedTickets` to an array using `Array.from(selectedTickets)` before calling `.map()`.
    *   Replace `.length` with `.size` for checking the number of selected items.

2.  **Harden `src/modules/knowledge/components/ArticleList.tsx`**:
    *   Add defensive checks to ensure `categories` and `tags` are arrays before mapping.
    *   Example: `(Array.isArray(categories) ? categories : []).map(...)`

3.  **Verify**:
    *   Run the build and type check to ensure no other type errors remain.
    *   (Optional) Create a reproduction test case if needed.

These changes will resolve the runtime crashes and improve the robustness of the UI.