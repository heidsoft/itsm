import { ticketService, Ticket } from '@/lib/services/ticket-service';
import { incidentService, Incident } from '@/lib/services/incident-service';
import { problemService, Problem } from '@/lib/services/problem-service';

export interface GlobalSearchResult {
  tickets: Ticket[];
  incidents: Incident[];
  problems: Problem[];
}

export const globalSearchApi = {
  search: async (query: string): Promise<GlobalSearchResult> => {
    const [tickets, incidents, problems] = await Promise.all([
      ticketService.getTickets({ query }),
      incidentService.getIncidents({ query }),
      problemService.getProblems({ query }),
    ]);

    return {
      tickets: tickets.items,
      incidents: incidents.items,
      problems: problems.items,
    };
  },
};
