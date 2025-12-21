"use client";

import { create } from "zustand";
import { devtools } from "zustand/middleware";

export interface PaginationState {
  current: number;
  pageSize: number;
  total: number;
}

export interface TicketFilters {
  status?: string;
  priority?: string;
  type?: string;
  category?: string;
  assignee_id?: number;
  keyword?: string;
  dateRange?: [string, string];
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
}

interface TicketUIState {
  // Selection state
  selectedRowKeys: Array<string | number>;
  
  // Modal states
  modalVisible: boolean;
  editingTicket: any | null;
  templateModalVisible: boolean;
  batchModalVisible: boolean;
  
  // View states
  viewMode: "table" | "card" | "kanban";
  sidebarVisible: boolean;
  
  // Loading states
  actionLoading: boolean;
  uploadLoading: boolean;
}

interface TicketUIActions {
  // Selection actions
  setSelectedRowKeys: (keys: Array<string | number>) => void;
  clearSelection: () => void;
  
  // Modal actions
  setModalVisible: (visible: boolean) => void;
  setEditingTicket: (ticket: any | null) => void;
  setTemplateModalVisible: (visible: boolean) => void;
  setBatchModalVisible: (visible: boolean) => void;
  
  // View actions
  setViewMode: (mode: "table" | "card" | "kanban") => void;
  setSidebarVisible: (visible: boolean) => void;
  
  // Loading actions
  setActionLoading: (loading: boolean) => void;
  setUploadLoading: (loading: boolean) => void;
  
  // Reset actions
  resetUI: () => void;
}

type TicketUIStore = TicketUIState & TicketUIActions;

const initialUIState: TicketUIState = {
  selectedRowKeys: [],
  modalVisible: false,
  editingTicket: null,
  templateModalVisible: false,
  batchModalVisible: false,
  viewMode: "table",
  sidebarVisible: true,
  actionLoading: false,
  uploadLoading: false,
};

export const useTicketUIStore = create<TicketUIStore>()(
  devtools(
    (set) => ({
      ...initialUIState,
      
      // Selection actions
      setSelectedRowKeys: (selectedRowKeys) => 
        set({ selectedRowKeys }, false, "setSelectedRowKeys"),
      
      clearSelection: () => 
        set({ selectedRowKeys: [] }, false, "clearSelection"),
      
      // Modal actions
      setModalVisible: (modalVisible) => 
        set({ modalVisible }, false, "setModalVisible"),
      
      setEditingTicket: (editingTicket) => 
        set({ editingTicket }, false, "setEditingTicket"),
      
      setTemplateModalVisible: (templateModalVisible) => 
        set({ templateModalVisible }, false, "setTemplateModalVisible"),
      
      setBatchModalVisible: (batchModalVisible) => 
        set({ batchModalVisible }, false, "setBatchModalVisible"),
      
      // View actions
      setViewMode: (viewMode) => 
        set({ viewMode }, false, "setViewMode"),
      
      setSidebarVisible: (sidebarVisible) => 
        set({ sidebarVisible }, false, "setSidebarVisible"),
      
      // Loading actions
      setActionLoading: (actionLoading) => 
        set({ actionLoading }, false, "setActionLoading"),
      
      setUploadLoading: (uploadLoading) => 
        set({ uploadLoading }, false, "setUploadLoading"),
      
      // Reset actions
      resetUI: () => set(initialUIState, false, "resetUI"),
    }),
    {
      name: "ticket-ui-store",
    }
  )
);

// Selectors for optimized re-renders
export const useTicketSelection = () => useTicketUIStore((state) => ({
  selectedRowKeys: state.selectedRowKeys,
  setSelectedRowKeys: state.setSelectedRowKeys,
  clearSelection: state.clearSelection,
}));

export const useTicketModals = () => useTicketUIStore((state) => ({
  modalVisible: state.modalVisible,
  editingTicket: state.editingTicket,
  templateModalVisible: state.templateModalVisible,
  batchModalVisible: state.batchModalVisible,
  setModalVisible: state.setModalVisible,
  setEditingTicket: state.setEditingTicket,
  setTemplateModalVisible: state.setTemplateModalVisible,
  setBatchModalVisible: state.setBatchModalVisible,
}));

export const useTicketView = () => useTicketUIStore((state) => ({
  viewMode: state.viewMode,
  sidebarVisible: state.sidebarVisible,
  setViewMode: state.setViewMode,
  setSidebarVisible: state.setSidebarVisible,
}));

export const useTicketLoading = () => useTicketUIStore((state) => ({
  actionLoading: state.actionLoading,
  uploadLoading: state.uploadLoading,
  setActionLoading: state.setActionLoading,
  setUploadLoading: state.setUploadLoading,
}));