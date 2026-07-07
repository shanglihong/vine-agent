export interface TimelineItem {
  kind: 'reasoning' | 'tool_call';
  text?: string;
  toolCallId?: string;
  toolName?: string;
  toolArgs?: string;
  output?: string;
  error?: string;
}

export interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  reasoning_content?: string;
  tool_calls?: any[];
  timeline?: TimelineItem[];
}

export interface Session {
  id: string;
  user_id: string;
  updated_at: string;
  status?: string;
}

export interface Profile {
  user_id: string;
  preferences: string[];
  facts: string[];
}

export interface PendingTool {
  id: string;
  function: {
    name: string;
    arguments: string;
  };
}

export interface UserInfo {
  id: string;
  username: string;
  email: string;
}
