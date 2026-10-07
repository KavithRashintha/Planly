export type Persona = 'student' | 'undergraduate' | 'employee';

export interface User {
  id: string;
  email: string;
  full_name: string;
  persona: Persona;
  timezone: string;
  work_start: string;
  work_end: string;
  created_at: string;
  updated_at: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export type TaskStatus = 'todo' | 'in_progress' | 'done' | 'cancelled';

export interface Tag {
  id: string;
  user_id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface Subtask {
  id: string;
  task_id: string;
  title: string;
  done: boolean;
  position: number;
  created_at: string;
}

export interface Project {
  id: string;
  user_id: string;
  name: string;
  color: string;
  created_at: string;
  updated_at: string;
  task_count?: number;
  completed_count?: number;
}

export interface Task {
  id: string;
  user_id: string;
  project_id?: string | null;
  title: string;
  description?: string;
  priority: number; // 1: Low, 2: Medium, 3: High, 4: Urgent
  status: TaskStatus;
  due_date?: string | null;
  due_at?: string | null;
  estimated_minutes?: number | null;
  estimate_minutes?: number | null;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
  project?: Project | null;
  subtasks?: Subtask[];
  tags?: Tag[];
}

export interface TimeBlock {
  id: string;
  user_id: string;
  task_id?: string | null;
  title: string;
  starts_at: string;
  ends_at: string;
  created_at: string;
  task?: Task | null;
}

export interface Reminder {
  id: string;
  user_id: string;
  task_id?: string | null;
  message: string;
  remind_at: string;
  fired: boolean;
  created_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  kind: 'reminder' | 'briefing' | 'agent' | 'system';
  title: string;
  body: string;
  read: boolean;
  created_at: string;
}

export interface WorkloadDay {
  date: string;
  planned_minutes?: number;
  total_minutes?: number;
  capacity_minutes?: number;
  over_capacity?: boolean;
  task_count: number;
}

export interface WorkloadResponse {
  from: string;
  to: string;
  days: WorkloadDay[];
}

export interface DashboardMetrics {
  total_tasks: number;
  completed_tasks: number;
  overdue_tasks: number;
  today_tasks: number;
}

// ----------------- Agent Models -----------------

export interface Conversation {
  id: string;
  user_id: string;
  title: string;
  created_at: string;
}

export interface MessageContentBlock {
  type: 'text' | 'tool_use' | 'tool_result';
  text?: string;
  id?: string;
  name?: string;
  input?: any;
  tool_use_id?: string;
  content?: string;
  is_error?: boolean;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant';
  content: MessageContentBlock[] | string;
  created_at: string;
}

export interface ProposalAction {
  id: string;
  tool: string;
  input: any;
  status: 'pending' | 'applied' | 'failed' | 'skipped';
  error?: string;
}

export interface Proposal {
  id: string;
  user_id: string;
  run_id?: string | null;
  summary: string;
  actions: ProposalAction[];
  status: 'pending' | 'approved' | 'rejected' | 'partial' | 'failed';
  created_at: string;
  decided_at?: string | null;
}

export interface ChatResponse {
  conversation_id: string;
  message_id: string;
  reply: string;
  proposals?: Proposal[];
  run_id: string;
}

export interface ToolCall {
  id: string;
  run_id: string;
  tool_name: string;
  input: any;
  output?: any;
  is_error: boolean;
  created_at: string;
}

export interface AgentRun {
  id: string;
  conversation_id?: string | null;
  user_id: string;
  trigger: 'chat' | 'briefing' | 'plan_goal' | 'reschedule';
  status: 'running' | 'completed' | 'failed';
  iterations: number;
  input_tokens: number;
  output_tokens: number;
  error?: string | null;
  created_at: string;
  tool_calls?: ToolCall[];
}

