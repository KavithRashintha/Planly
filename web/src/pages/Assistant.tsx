import React, { useState, useEffect, useRef } from 'react';
import { 
  Sparkles, 
  Send, 
  Target, 
  CalendarClock, 
  Plus, 
  Trash2, 
  Bot, 
  User as UserIcon, 
  Wrench, 
  Loader2, 
  Clock,
  ArrowRight,
  MessageSquare
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { 
  Conversation, 
  Message, 
  Proposal, 
  ChatResponse, 
  MessageContentBlock 
} from '../types';
import { agentApi } from '../api/agent';
import { ProposalCard } from '../components/ProposalCard';
import { PlanGoalModal } from '../components/PlanGoalModal';
import { RunHistoryModal } from '../components/RunHistoryModal';
import { SuggestedPrompts } from '../components/SuggestedPrompts';

export const Assistant: React.FC = () => {
  const { user } = useAuth();

  // State
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConvId, setActiveConvId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [inputText, setInputText] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isRescheduling, setIsRescheduling] = useState(false);
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [isRunModalOpen, setIsRunModalOpen] = useState(false);
  const [isPlanModalOpen, setIsPlanModalOpen] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom of chat
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  // Load conversations list on mount
  useEffect(() => {
    loadConversations();
  }, []);

  const loadConversations = async () => {
    try {
      const convList = await agentApi.getConversations();
      setConversations(convList);
      if (convList.length > 0 && !activeConvId) {
        selectConversation(convList[0].id);
      }
    } catch {
      // Ignore initial fetch errors
    }
  };

  // Load conversation details
  const selectConversation = async (convId: string) => {
    setActiveConvId(convId);
    setErrorMsg(null);
    try {
      const data = await agentApi.getConversation(convId);
      setMessages(data.messages || []);
      
      // Also load user proposals to match with conversation
      const allProps = await agentApi.getProposals();
      setProposals(allProps);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to load conversation');
    }
  };

  const handleNewChat = () => {
    setActiveConvId(null);
    setMessages([]);
    setErrorMsg(null);
  };

  const handleDeleteConversation = async (e: React.MouseEvent, convId: string) => {
    e.stopPropagation();
    try {
      await agentApi.deleteConversation(convId);
      const remaining = conversations.filter((c) => c.id !== convId);
      setConversations(remaining);
      if (activeConvId === convId) {
        if (remaining.length > 0) {
          selectConversation(remaining[0].id);
        } else {
          handleNewChat();
        }
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to delete conversation');
    }
  };

  const handleSendMessage = async (textToSend?: string) => {
    const text = (textToSend || inputText).trim();
    if (!text || isLoading) return;

    setInputText('');
    setErrorMsg(null);

    // Optimistically add user message
    const tempUserMsg: Message = {
      id: 'temp-' + Date.now(),
      conversation_id: activeConvId || '',
      role: 'user',
      content: [{ type: 'text', text }],
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, tempUserMsg]);
    setIsLoading(true);

    try {
      const res: ChatResponse = await agentApi.sendMessage(text, activeConvId || undefined);

      if (!activeConvId) {
        setActiveConvId(res.conversation_id);
        await loadConversations();
      }

      // Append assistant message
      const assistantMsg: Message = {
        id: res.message_id,
        conversation_id: res.conversation_id,
        role: 'assistant',
        content: [{ type: 'text', text: res.reply }],
        created_at: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, assistantMsg]);

      // If new proposals created, append to proposal state
      if (res.proposals && res.proposals.length > 0) {
        setProposals((prev) => [...res.proposals!, ...prev]);
      }
      
      setActiveRunId(res.run_id);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to send message');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRescheduleAction = async () => {
    setIsRescheduling(true);
    setErrorMsg(null);

    try {
      const res = await agentApi.reschedule(undefined, activeConvId || undefined);
      if (!activeConvId) {
        setActiveConvId(res.conversation_id);
        await loadConversations();
      }

      const assistantMsg: Message = {
        id: res.message_id,
        conversation_id: res.conversation_id,
        role: 'assistant',
        content: [{ type: 'text', text: res.reply }],
        created_at: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, assistantMsg]);

      if (res.proposals && res.proposals.length > 0) {
        setProposals((prev) => [...res.proposals!, ...prev]);
      }
      setActiveRunId(res.run_id);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to reschedule tasks');
    } finally {
      setIsRescheduling(false);
    }
  };

  const handlePlanGoalSuccess = (res: ChatResponse) => {
    if (!activeConvId) {
      setActiveConvId(res.conversation_id);
      loadConversations();
    }
    const assistantMsg: Message = {
      id: res.message_id,
      conversation_id: res.conversation_id,
      role: 'assistant',
      content: [{ type: 'text', text: res.reply }],
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, assistantMsg]);
    if (res.proposals && res.proposals.length > 0) {
      setProposals((prev) => [...res.proposals!, ...prev]);
    }
    setActiveRunId(res.run_id);
  };

  const renderMessageContent = (msg: Message) => {
    if (typeof msg.content === 'string') {
      return <p className="whitespace-pre-wrap leading-relaxed">{msg.content}</p>;
    }

    if (Array.isArray(msg.content)) {
      return (
        <div className="space-y-2">
          {msg.content.map((block: MessageContentBlock, idx: number) => {
            if (block.type === 'text') {
              return (
                <p key={idx} className="whitespace-pre-wrap leading-relaxed">
                  {block.text}
                </p>
              );
            }
            return null;
          })}
        </div>
      );
    }

    return null;
  };

  return (
    <div className="flex h-[calc(100vh-6rem)] -m-6 md:-m-8 bg-[#191919] overflow-hidden text-[#e6e6e6]">
      {/* Left Chat Sessions Sidebar */}
      <div className="w-64 bg-[#1e1e1e] border-r border-[#2b2b2b] flex flex-col shrink-0">
        <div className="p-3 border-b border-[#2b2b2b]">
          <button
            onClick={handleNewChat}
            className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-md bg-[#282828] hover:bg-[#323232] text-xs font-medium text-[#f0f0f0] transition-colors border border-[#363636] cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5 text-[#2383e2]" />
            New Chat
          </button>
        </div>

        {/* Conversations List */}
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          <div className="px-2 py-1 text-[10px] font-semibold uppercase tracking-wider text-[#666666]">
            Recent Chats
          </div>

          {conversations.length === 0 ? (
            <p className="px-2 py-3 text-xs text-[#737373] italic">No saved conversations</p>
          ) : (
            conversations.map((conv) => (
              <div
                key={conv.id}
                onClick={() => selectConversation(conv.id)}
                className={`group flex items-center justify-between px-2.5 py-2 rounded-md text-xs cursor-pointer transition-colors ${
                  activeConvId === conv.id
                    ? 'bg-[#2b2b2b] text-[#ffffff] font-medium'
                    : 'text-[#9b9b9b] hover:text-[#ebebeb] hover:bg-[#252525]'
                }`}
              >
                <div className="flex items-center gap-2 min-w-0">
                  <MessageSquare className="w-3.5 h-3.5 shrink-0 text-[#777777]" />
                  <span className="truncate">{conv.title}</span>
                </div>
                <button
                  onClick={(e) => handleDeleteConversation(e, conv.id)}
                  title="Delete chat"
                  className="opacity-0 group-hover:opacity-100 p-1 text-[#666666] hover:text-[#f87171] rounded transition-opacity"
                >
                  <Trash2 className="w-3 h-3" />
                </button>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col min-w-0 bg-[#191919]">
        {/* Top Chat Bar with Quick Actions */}
        <div className="px-6 py-3.5 border-b border-[#2b2b2b] flex items-center justify-between bg-[#191919]/90 backdrop-blur-xs shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-[#2383e2]/10 border border-[#2383e2]/20 flex items-center justify-center text-[#2383e2]">
              <Sparkles className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-[#f0f0f0] leading-none">
                Planly Assistant
              </h2>
              <span className="text-[10px] text-[#737373] capitalize mt-0.5 inline-block">
                AI Work Planner • {user?.persona || 'personal'} mode
              </span>
            </div>
          </div>

          {/* Quick Action Buttons */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => setIsPlanModalOpen(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-[#242424] hover:bg-[#2e2e2e] text-[#cccccc] hover:text-white border border-[#333333] transition-colors cursor-pointer"
            >
              <Target className="w-3.5 h-3.5 text-[#2383e2]" />
              Plan a Goal
            </button>
            <button
              onClick={handleRescheduleAction}
              disabled={isRescheduling}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-[#242424] hover:bg-[#2e2e2e] disabled:opacity-50 text-[#cccccc] hover:text-white border border-[#333333] transition-colors cursor-pointer"
            >
              <CalendarClock className="w-3.5 h-3.5 text-[#fbbf24]" />
              {isRescheduling ? 'Rescheduling...' : 'Reschedule Overdue'}
            </button>
          </div>
        </div>

        {/* Message Thread */}
        <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
          {errorMsg && (
            <div className="p-3 rounded-lg bg-[#2e1717] border border-[#4a2222] text-xs text-[#f87171] max-w-2xl mx-auto">
              {errorMsg}
            </div>
          )}

          {/* Empty State */}
          {messages.length === 0 && (
            <div className="py-12 flex flex-col items-center justify-center text-center">
              <div className="w-12 h-12 rounded-2xl bg-[#222222] border border-[#2e2e2e] flex items-center justify-center mb-3 shadow-sm">
                <Sparkles className="w-6 h-6 text-[#2383e2]" />
              </div>
              <h3 className="text-base font-semibold text-[#f0f0f0]">
                How can I help you plan today?
              </h3>
              <p className="text-xs text-[#888888] max-w-md mt-1 leading-relaxed">
                I can schedule time blocks, break down goals, inspect your workload, and reschedule overdue tasks with your approval.
              </p>

              {/* Persona-specific Suggested Prompts */}
              <SuggestedPrompts 
                persona={user?.persona} 
                onSelectPrompt={(p) => handleSendMessage(p)} 
              />
            </div>
          )}

          {/* Message List */}
          {messages.map((msg) => {
            const isUser = msg.role === 'user';
            return (
              <div
                key={msg.id}
                className={`flex gap-3 max-w-3xl ${isUser ? 'ml-auto justify-end' : 'mr-auto justify-start'}`}
              >
                {!isUser && (
                  <div className="w-7 h-7 rounded-lg bg-[#242424] border border-[#333333] flex items-center justify-center shrink-0 mt-0.5 text-[#2383e2]">
                    <Bot className="w-4 h-4" />
                  </div>
                )}

                <div className={`space-y-2 ${isUser ? 'items-end' : 'items-start'}`}>
                  <div
                    className={`p-3.5 rounded-xl text-xs ${
                      isUser
                        ? 'bg-[#2383e2] text-white rounded-tr-xs'
                        : 'bg-[#212121] text-[#e6e6e6] border border-[#2e2e2e] rounded-tl-xs shadow-xs'
                    }`}
                  >
                    {renderMessageContent(msg)}
                  </div>

                  {/* Render any pending proposal cards linked to this assistant turn */}
                  {!isUser && proposals.length > 0 && (
                    <div className="space-y-2">
                      {proposals.map((prop) => (
                        <ProposalCard 
                          key={prop.id} 
                          proposal={prop} 
                          onDecided={(updated) => {
                            setProposals((prev) => 
                              prev.map((p) => (p.id === updated.id ? updated : p))
                            );
                          }}
                        />
                      ))}
                    </div>
                  )}

                  {/* Execution Trace Badge */}
                  {!isUser && activeRunId && (
                    <div className="pt-0.5">
                      <button
                        onClick={() => setIsRunModalOpen(true)}
                        className="inline-flex items-center gap-1 text-[11px] text-[#737373] hover:text-[#2383e2] transition-colors"
                      >
                        <Wrench className="w-3 h-3" />
                        <span>Inspect tool calls</span>
                      </button>
                    </div>
                  )}
                </div>

                {isUser && (
                  <div className="w-7 h-7 rounded-full bg-[#333333] flex items-center justify-center shrink-0 mt-0.5 text-[11px] font-bold text-[#e6e6e6]">
                    {user?.full_name ? user.full_name.charAt(0).toUpperCase() : 'U'}
                  </div>
                )}
              </div>
            );
          })}

          {/* Thinking / Loading Animation */}
          {isLoading && (
            <div className="flex gap-3 max-w-3xl mr-auto justify-start">
              <div className="w-7 h-7 rounded-lg bg-[#242424] border border-[#333333] flex items-center justify-center shrink-0 mt-0.5 text-[#2383e2]">
                <Bot className="w-4 h-4 animate-pulse" />
              </div>
              <div className="p-3.5 rounded-xl bg-[#212121] border border-[#2e2e2e] text-xs text-[#888888] flex items-center gap-2">
                <Loader2 className="w-3.5 h-3.5 animate-spin text-[#2383e2]" />
                <span>Thinking & inspecting schedule...</span>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Input Bar */}
        <div className="p-4 border-t border-[#2b2b2b] bg-[#191919]">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleSendMessage();
            }}
            className="max-w-3xl mx-auto flex items-center gap-2 bg-[#202020] border border-[#2e2e2e] rounded-xl px-3 py-2 focus-within:border-[#2383e2] transition-colors shadow-sm"
          >
            <input
              type="text"
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              placeholder="Ask Planly to organize tasks, balance workload, or schedule blocks..."
              disabled={isLoading}
              className="flex-1 bg-transparent text-xs text-[#e6e6e6] placeholder-[#666666] focus:outline-none"
            />
            <button
              type="submit"
              disabled={!inputText.trim() || isLoading}
              className="p-1.5 rounded-lg bg-[#2383e2] hover:bg-[#1d6fc2] disabled:opacity-40 disabled:hover:bg-[#2383e2] text-white transition-colors cursor-pointer shrink-0"
            >
              <Send className="w-3.5 h-3.5" />
            </button>
          </form>
          <p className="text-[10px] text-center text-[#555555] mt-2">
            Planly writes changes as proposals requiring your approval before affecting planner data.
          </p>
        </div>
      </div>

      {/* Modals */}
      <PlanGoalModal
        isOpen={isPlanModalOpen}
        conversationId={activeConvId || undefined}
        onClose={() => setIsPlanModalOpen(false)}
        onSuccess={handlePlanGoalSuccess}
      />

      {activeRunId && (
        <RunHistoryModal
          runId={activeRunId}
          isOpen={isRunModalOpen}
          onClose={() => setIsRunModalOpen(false)}
        />
      )}
    </div>
  );
};
