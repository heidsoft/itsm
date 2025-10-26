"use client";

import { useState, useCallback, useMemo } from "react";
import { TicketStatus, TicketPriority, TicketType } from '@/lib/services/ticket-service';
import { TicketQueryFilters } from "./useTickets";

export interface TicketFilterState {
  status: "all" | "open" | "in_progress" | "resolved" | "closed";
  priority: "all" | "p1" | "p2" | "p3" | "p4";
  type: "all" | TicketType;
  keyword: string;
  dateStart: string;
  dateEnd: string;
  sortBy: string;
}

export interface UseTicketFiltersReturn {
  // Filter state
  componentFilters: TicketFilterState;
  domainFilters: Partial<TicketQueryFilters>;
  
  // Actions
  updateComponentFilters: (filters: Partial<TicketFilterState>) => void;
  updateDomainFilters: (filters: Partial<TicketQueryFilters>) => void;
  resetFilters: () => void;
  
  // Mappers
  mapComponentToDomain: (cf: TicketFilterState) => Partial<TicketQueryFilters>;
  mapDomainToComponent: (df: Partial<TicketQueryFilters>) => TicketFilterState;
}

const DEFAULT_FILTER_STATE: TicketFilterState = {
  status: "all",
  priority: "all",
  type: "all",
  keyword: "",
  dateStart: "",
  dateEnd: "",
  sortBy: "createdAt_desc",
};

export const useTicketFilters = (): UseTicketFiltersReturn => {
  const [componentFilters, setComponentFilters] = useState<TicketFilterState>(DEFAULT_FILTER_STATE);

  // Map component filters to domain filters
  const mapComponentToDomain = useCallback((cf: TicketFilterState): Partial<TicketQueryFilters> => {
    const statusMap: Record<TicketFilterState["status"], TicketStatus | undefined> = {
      all: undefined,
      open: TicketStatus.OPEN,
      in_progress: TicketStatus.IN_PROGRESS,
      resolved: TicketStatus.RESOLVED,
      closed: TicketStatus.CLOSED,
    };
    
    const priorityMap: Record<TicketFilterState["priority"], TicketPriority | undefined> = {
      all: undefined,
      p1: TicketPriority.URGENT,
      p2: TicketPriority.HIGH,
      p3: TicketPriority.MEDIUM,
      p4: TicketPriority.LOW,
    };
    
    const dateRange = cf.dateStart && cf.dateEnd ? [cf.dateStart, cf.dateEnd] as [string, string] : undefined;
    
    return {
      status: statusMap[cf.status],
      priority: priorityMap[cf.priority],
      type: cf.type === "all" ? undefined : cf.type,
      keyword: cf.keyword || undefined,
      dateRange,
    };
  }, []);

  // Map domain filters to component filters
  const mapDomainToComponent = useCallback((df: Partial<TicketQueryFilters>): TicketFilterState => {
    const statusReverseMap: Record<string, TicketFilterState["status"]> = {
      [TicketStatus.OPEN]: "open",
      [TicketStatus.IN_PROGRESS]: "in_progress",
      [TicketStatus.RESOLVED]: "resolved",
      [TicketStatus.CLOSED]: "closed",
    };
    
    const priorityReverseMap: Record<string, TicketFilterState["priority"]> = {
      [TicketPriority.URGENT]: "p1",
      [TicketPriority.HIGH]: "p2",
      [TicketPriority.MEDIUM]: "p3",
      [TicketPriority.LOW]: "p4",
    };
    
    return {
      status: df.status ? statusReverseMap[df.status] : "all",
      priority: df.priority ? priorityReverseMap[df.priority] : "all",
      type: df.type || "all",
      keyword: df.keyword || "",
      dateStart: df.dateRange?.[0] || "",
      dateEnd: df.dateRange?.[1] || "",
      sortBy: "createdAt_desc",
    };
  }, []);

  // Update component filters
  const updateComponentFilters = useCallback((newFilters: Partial<TicketFilterState>) => {
    setComponentFilters(prev => ({ ...prev, ...newFilters }));
  }, []);

  // Update domain filters (converts to component filters internally)
  const updateDomainFilters = useCallback((newFilters: Partial<TicketQueryFilters>) => {
    const componentState = mapDomainToComponent(newFilters);
    setComponentFilters(prev => ({ ...prev, ...componentState }));
  }, [mapDomainToComponent]);

  // Reset filters to default
  const resetFilters = useCallback(() => {
    setComponentFilters(DEFAULT_FILTER_STATE);
  }, []);

  // Compute domain filters from current component state
  const domainFilters = useMemo(() => {
    return mapComponentToDomain(componentFilters);
  }, [componentFilters, mapComponentToDomain]);

  return {
    // Filter state
    componentFilters,
    domainFilters,
    
    // Actions
    updateComponentFilters,
    updateDomainFilters,
    resetFilters,
    
    // Mappers
    mapComponentToDomain,
    mapDomainToComponent,
  };
};
