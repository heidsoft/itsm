'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Card,
  Input,
  Button,
  List,
  Avatar,
  Badge,
  Space,
  Typography,
  message,
  App,
  Tooltip,
} from 'antd';
import { SendOutlined, UserOutlined, EyeOutlined, MessageOutlined } from '@ant-design/icons';
import { Ticket } from '@/lib/services/ticket-service';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

const { TextArea } = Input;
const { Text } = Typography;

interface Collaborator {
  id: number;
  name: string;
  avatar?: string;
  isOnline: boolean;
  lastSeen?: string;
}

interface CollaborationMessage {
  id: string;
  user_id: number;
  user_name: string;
  user_avatar?: string;
  content: string;
  type: 'comment' | 'status_change' | 'assignment' | 'system';
  created_at: string;
}

interface TicketCollaborationProps {
  ticket: Ticket;
  collaborators?: Collaborator[];
  messages?: CollaborationMessage[];
  onSendMessage?: (content: string) => Promise<void>;
  onTyping?: (isTyping: boolean) => void;
  canCollaborate?: boolean;
}

export const TicketCollaboration: React.FC<TicketCollaborationProps> = ({
  ticket,
  collaborators = [],
  messages = [],
  onSendMessage,
  onTyping,
  canCollaborate = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [newMessage, setNewMessage] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const [onlineUsers, setOnlineUsers] = useState<Collaborator[]>([]);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 模拟在线用户（实际应该从WebSocket获取）
  useEffect(() => {
    // 注意：WebSocket实时用户功能尚未实现
    // 未来可通过 WebSocket 连接实时获取在线用户列表
    setOnlineUsers(collaborators.filter(c => c.isOnline));
  }, [collaborators]);

  // 滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // 处理输入变化（打字指示）
  const handleInputChange = useCallback(
    (value: string) => {
      setNewMessage(value);

      if (!isTyping && value.length > 0) {
        setIsTyping(true);
        onTyping?.(true);
      }

      // 清除之前的定时器
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }

      // 设置新的定时器
      typingTimeoutRef.current = setTimeout(() => {
        setIsTyping(false);
        onTyping?.(false);
        if (typingTimeoutRef.current) {
          clearTimeout(typingTimeoutRef.current);
        }
      }, 2000);
    },
    [isTyping, onTyping]
  );

  // 发送消息
  const handleSendMessage = useCallback(async () => {
    if (!newMessage.trim() || !canCollaborate) {
      return;
    }

    try {
      await onSendMessage?.(newMessage);
      setNewMessage('');
      setIsTyping(false);
      onTyping?.(false);
    } catch (error) {
      antMessage.error('发送消息失败');
    }
  }, [newMessage, onSendMessage, onTyping, canCollaborate, antMessage]);

  // 获取消息图标
  const getMessageIcon = (type: string) => {
    switch (type) {
      case 'status_change':
        return '🔄';
      case 'assignment':
        return '👤';
      case 'system':
        return 'ℹ️';
      default:
        return '💬';
    }
  };

  return (
    <div className="space-y-4">
      {/* 在线协作者 */}
      {onlineUsers.length > 0 && (
        <Card size="small" title="在线协作者">
          <Space wrap>
            {onlineUsers.map(user => (
              <Tooltip key={user.id} title={user.name}>
                <Badge dot status="success" offset={[-2, 2]}>
                  <Avatar size="small" src={user.avatar} icon={<UserOutlined />}>
                    {user.name?.[0]}
                  </Avatar>
                </Badge>
              </Tooltip>
            ))}
          </Space>
        </Card>
      )}

      {/* 协作消息列表 */}
      <Card
        title={
          <div className="flex items-center gap-2">
            <MessageOutlined />
            <span>实时协作</span>
            {messages.length > 0 && <Badge count={messages.length} showZero className="ml-2" />}
          </div>
        }
        className="flex-1"
        bodyStyle={{ maxHeight: '500px', overflowY: 'auto' }}
      >
        {messages.length > 0 ? (
          <List
            dataSource={messages}
            renderItem={message => (
              <List.Item className="!px-0 !py-2 border-b border-gray-100 last:border-0">
                <div className="flex items-start gap-3 w-full">
                  <Avatar size="small" src={message.user_avatar} icon={<UserOutlined />}>
                    {message.user_name?.[0]}
                  </Avatar>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <Text strong className="text-sm">
                        {message.user_name}
                      </Text>
                      <Text type="secondary" className="text-xs">
                        {getMessageIcon(message.type)}
                      </Text>
                      <Text type="secondary" className="text-xs">
                        {formatDistanceToNow(new Date(message.created_at), {
                          addSuffix: true,
                          locale: zhCN,
                        })}
                      </Text>
                    </div>
                    <div className="text-sm text-gray-700 whitespace-pre-wrap">
                      {message.content}
                    </div>
                  </div>
                </div>
              </List.Item>
            )}
          />
        ) : (
          <div className="text-center py-8 text-gray-500">
            <MessageOutlined className="text-4xl mb-2 text-gray-300" />
            <div>暂无协作消息</div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </Card>

      {/* 消息输入框 */}
      {canCollaborate && (
        <Card size="small">
          <Space orientation="vertical" className="w-full" size="small">
            <TextArea
              value={newMessage}
              onChange={e => handleInputChange(e.target.value)}
              onPressEnter={e => {
                if (e.shiftKey) {
                  return; // Shift+Enter换行
                }
                e.preventDefault();
                handleSendMessage();
              }}
              placeholder="输入协作消息...（Enter发送，Shift+Enter换行）"
              rows={3}
              maxLength={500}
              showCount
            />
            <div className="flex items-center justify-between">
              <Text type="secondary" className="text-xs">
                {isTyping && '正在输入...'}
              </Text>
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSendMessage}
                disabled={!newMessage.trim()}
              >
                发送
              </Button>
            </div>
          </Space>
        </Card>
      )}
    </div>
  );
};
