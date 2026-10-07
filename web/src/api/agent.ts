import { api } from './client';
import { 
  Conversation, 
  Message, 
  Proposal, 
  ChatResponse, 
  AgentRun 
} from '../types';

export const agentApi = {
  // Chat
  sendMessage: async (message: string, conversation_id?: string): Promise<ChatResponse> => {
    const res = await api.post<ChatResponse>('/agent/chat', {
      message,
      conversation_id,
    });
    return res.data;
  },

  // Plan Goal
  planGoal: async (goal: string, deadline: string, conversation_id?: string): Promise<ChatResponse> => {
    const res = await api.post<ChatResponse>('/agent/plan-goal', {
      goal,
      deadline,
      conversation_id,
    });
    return res.data;
  },

  // Reschedule
  reschedule: async (task_ids?: string[], conversation_id?: string): Promise<ChatResponse> => {
    const res = await api.post<ChatResponse>('/agent/reschedule', {
      task_ids,
      conversation_id,
    });
    return res.data;
  },

  // Conversations
  getConversations: async (limit = 30, offset = 0): Promise<Conversation[]> => {
    const res = await api.get<Conversation[]>(`/agent/conversations?limit=${limit}&offset=${offset}`);
    return res.data || [];
  },

  getConversation: async (id: string): Promise<{ conversation: Conversation; messages: Message[] }> => {
    const res = await api.get<{ conversation: Conversation; messages: Message[] }>(`/agent/conversations/${id}`);
    return res.data;
  },

  deleteConversation: async (id: string): Promise<void> => {
    await api.delete<void>(`/agent/conversations/${id}`);
  },

  // Proposals
  getProposals: async (limit = 20, offset = 0): Promise<Proposal[]> => {
    const res = await api.get<Proposal[]>(`/agent/proposals?limit=${limit}&offset=${offset}`);
    return res.data || [];
  },

  getProposal: async (id: string): Promise<Proposal> => {
    const res = await api.get<Proposal>(`/agent/proposals/${id}`);
    return res.data;
  },

  approveProposal: async (id: string, action_ids?: string[]): Promise<Proposal> => {
    const res = await api.post<Proposal>(`/agent/proposals/${id}/approve`, {
      action_ids,
    });
    return res.data;
  },

  rejectProposal: async (id: string): Promise<Proposal> => {
    const res = await api.post<Proposal>(`/agent/proposals/${id}/reject`, {});
    return res.data;
  },

  // Run history & Tool calls
  getAgentRun: async (id: string): Promise<AgentRun> => {
    const res = await api.get<AgentRun>(`/agent/runs/${id}`);
    return res.data;
  },
};
