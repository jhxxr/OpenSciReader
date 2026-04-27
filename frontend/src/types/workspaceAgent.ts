import type { WorkspaceKnowledgeSourceRef } from './workspaceKnowledge';

export interface WorkspaceAgentSession {
  id: string;
  workspaceId: string;
  title: string;
  surface: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceAgentMessage {
  id: number;
  sessionId: string;
  workspaceId: string;
  surface: string;
  role: string;
  kind: string;
  prompt: string;
  content: string;
  skillName: string;
  executedSkill?: WorkspaceAgentExecutedSkill | null;
  evidenceCount: number;
  createdAt: string;
}

export interface WorkspaceAgentSkillDefinition {
  name: string;
  label: string;
  description: string;
  manualEnabled: boolean;
  autoEnabled: boolean;
  readerEnabled: boolean;
  workspaceOnly: boolean;
}

export interface WorkspaceAgentExecutedSkill {
  name: string;
  label: string;
  routedBy: 'manual' | 'auto';
  reason: string;
  displayText: string;
}

export interface WorkspaceAgentAskInput {
  workspaceId: string;
  documentId?: string;
  sessionId?: string;
  surface: string;
  skillName?: string;
  includeDocumentContext?: boolean;
  includeWorkspaceContext?: boolean;
  selection?: string;
  currentPage?: number;
  providerId: number;
  modelId: number;
  question: string;
}

export interface WorkspaceAgentAskResult {
  session: WorkspaceAgentSession;
  userMessage: WorkspaceAgentMessage;
  assistantMessage: WorkspaceAgentMessage;
  executedSkill: WorkspaceAgentExecutedSkill;
  query: {
    answer: string;
    evidence: Array<{
      kind: string;
      id: string;
      title: string;
      summary: string;
      excerpt: string;
      sourceRefs: WorkspaceKnowledgeSourceRef[];
      confidence?: number;
    }>;
    candidates: Array<{
      id: string;
      title: string;
      type: string;
      summary: string;
      aliases: string[];
      entityIds: string[];
      priority: string;
      sourceId: string;
      pageStart: number;
      pageEnd: number;
      excerpt: string;
      sourceRefs: WorkspaceKnowledgeSourceRef[];
      confidence?: number;
    }>;
  };
}
