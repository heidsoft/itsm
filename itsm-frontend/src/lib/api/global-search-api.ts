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
      ticketService.listTickets({ search: query } as any),
      incidentService.listIncidents({ search: query } as any),
      problemService.listProblems({ search: query } as any),
    ]);

    return {
      tickets: (tickets as any).tickets || (tickets as any).items || [],
      incidents: (incidents as any).incidents || (incidents as any).items || [],
      problems: (problems as any).problems || (problems as any).items || [],
    };
  },
};
