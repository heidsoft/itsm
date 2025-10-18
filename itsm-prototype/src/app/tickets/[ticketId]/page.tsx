"use client";

import React, { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { TicketApi } from "../../lib/ticket-api";
import { Ticket, ApiResponse } from "../../lib/api-config";
import { TicketDetail } from "../../components/TicketDetail";
import {
  ArrowLeft,
  AlertCircle} from 'lucide-react';
import Link from "next/link";
import {
  Button,
  Card,
  Typography,
  App,
  Badge,
  Tag,
} from "antd";

const { Title, Text } = Typography;

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const ticketId = parseInt(params.ticketId as string);

  // Get ticket details
  const fetchTicket = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await TicketApi.getTicket(ticketId) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        setTicket(response.data);
      } else {
        setError(response.message || "Failed to load ticket");
      }
    } catch (error) {
      setError(error instanceof Error ? error.message : "Network error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId]);

  // Handle approval
  const handleApprove = async () => {
    try {
      const response = await TicketApi.updateTicket(ticketId, {
        status: "approved",
      }) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success("Approved successfully");
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || "Approval failed");
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : "Network error");
    }
  };

  // Handle rejection
  const handleReject = async () => {
    try {
      const response = await TicketApi.updateTicket(ticketId, {
        status: "rejected",
      }) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success("Rejected successfully");
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || "Rejection failed");
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : "Network error");
    }
  };

  // Handle assignment
  const handleAssign = async (assignee: string) => {
    try {
      const response = await TicketApi.updateTicket(ticketId, {
        assignee_id: parseInt(assignee), // This needs to be handled according to actual situation
      }) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success("Assigned successfully");
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || "Assignment failed");
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : "Network error");
    }
  };

  // Handle update
  const handleUpdate = async (updates: unknown) => {
    try {
      const response = await TicketApi.updateTicket(ticketId, updates as Partial<Ticket>) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success("Updated successfully");
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || "Update failed");
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : "Network error");
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <Text className="mt-4 block">Loading...</Text>
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
            <Title level={4} className="text-red-600 mb-2">
              Load Failed
            </Title>
            <Text type="secondary">{error}</Text>
            <div className="mt-4">
              <Button type="primary" onClick={fetchTicket}>
                Retry
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <XCircle className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <Title level={4} className="text-gray-600 mb-2">
              Ticket Not Found
            </Title>
            <Text type="secondary">The specified ticket was not found</Text>
            <div className="mt-4">
              <Link href="/tickets">
                <Button type="primary">Back to Ticket List</Button>
              </Link>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Page header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <Link href="/tickets">
              <Button icon={<ArrowLeft />} type="text">
                Back to List
              </Button>
            </Link>
            <div>
              <Title level={2} className="mb-1">
                Ticket Details #{ticket.id}
              </Title>
              <Text type="secondary">{ticket.title}</Text>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Badge
              status={
                ticket.status === "open"
                  ? "processing"
                  : ticket.status === "closed"
                  ? "success"
                  : "warning"
              }
              text={
                ticket.status === "open"
                  ? "In Progress"
                  : ticket.status === "closed"
                  ? "Closed"
                  : "Pending"
              }
            />
            <Tag
              color={
                ticket.priority === "high"
                  ? "red"
                  : ticket.priority === "medium"
                  ? "orange"
                  : "green"
              }
            >
              {ticket.priority === "high"
                ? "High Priority"
                : ticket.priority === "medium"
                ? "Medium Priority"
                : "Low Priority"}
            </Tag>
          </div>
        </div>
      </div>

      {/* Ticket detail component */}
      <Card>
        <TicketDetail
          ticket={ticket}
          onApprove={handleApprove}
          onReject={handleReject}
          onAssign={handleAssign}
          onUpdate={handleUpdate}
        />
      </Card>
    </div>
  );
};

export default TicketDetailPage;
