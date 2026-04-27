import { useEffect, useMemo, useState } from 'react';
import { Bot, Brain, Library, MessagesSquare, Plus, Send, Sparkles } from 'lucide-react';
import { Button } from './ui/Button';
import { useWorkspaceAgentStore } from '../store/workspaceAgentStore';
import type { ProviderConfig } from '../types/config';
import type { Workspace } from '../types/workspace';
import type { WorkspaceAgentExecutedSkill, WorkspaceAgentSkillDefinition } from '../types/workspaceAgent';

interface WorkspaceAgentShellProps {
  workspace: Workspace | null;
  llmProviderConfigs: ProviderConfig[];
  wikiScanProviderId: number;
  wikiScanModelId: number;
  onSwitchMode: () => void;
}

function formatSkillLabel(skillName: string, skills: WorkspaceAgentSkillDefinition[]): string {
  const matchedSkill = skills.find((skill) => skill.name === skillName);
  if (matchedSkill) {
    return matchedSkill.label;
  }

  return skillName
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function formatExecutedSkillDetail(executedSkill: WorkspaceAgentExecutedSkill): string {
  return executedSkill.routedBy === 'manual'
    ? 'Manual skill'
    : executedSkill.reason
      ? `Auto routed: ${executedSkill.reason.replaceAll('_', ' ')}`
      : 'Auto routed';
}

export function WorkspaceAgentShell({ workspace, llmProviderConfigs, wikiScanProviderId, wikiScanModelId, onSwitchMode }: WorkspaceAgentShellProps) {
  const [draft, setDraft] = useState('');
  const ensureWorkspace = useWorkspaceAgentStore((state) => state.ensureWorkspace);
  const createSession = useWorkspaceAgentStore((state) => state.createSession);
  const loadMessages = useWorkspaceAgentStore((state) => state.loadMessages);
  const ask = useWorkspaceAgentStore((state) => state.ask);
  const setSelectedSkill = useWorkspaceAgentStore((state) => state.setSelectedSkill);
  const pane = useWorkspaceAgentStore((state) => (workspace ? state.panes[workspace.id] : undefined));

  useEffect(() => {
    if (workspace?.id) {
      void ensureWorkspace(workspace.id);
    }
  }, [ensureWorkspace, workspace?.id]);

  const availableLLMProviderConfigs = useMemo(
    () => llmProviderConfigs.filter((item) => item.provider.type === 'llm' && item.models.length > 0),
    [llmProviderConfigs],
  );
  const defaultProviderConfig = useMemo(
    () => availableLLMProviderConfigs[0] ?? null,
    [availableLLMProviderConfigs],
  );
  const defaultModel = defaultProviderConfig?.models[0] ?? null;
  const manualSkills = (pane?.skills ?? []).filter((skill) => skill.manualEnabled);
  const selectedSkillName = pane?.selectedSkillName ?? null;
  const selectedSkill = manualSkills.find((skill) => skill.name === selectedSkillName) ?? null;
  const usesWikiScanModel = selectedSkillName === 'build_memory';
  const wikiProviderConfig = useMemo(
    () => availableLLMProviderConfigs.find((item) => item.provider.id === wikiScanProviderId) ?? null,
    [availableLLMProviderConfigs, wikiScanProviderId],
  );
  const wikiModel = useMemo(
    () => wikiProviderConfig?.models.find((model) => model.id === wikiScanModelId) ?? null,
    [wikiProviderConfig, wikiScanModelId],
  );
  const activeProviderConfig = usesWikiScanModel ? wikiProviderConfig : defaultProviderConfig;
  const activeModel = usesWikiScanModel ? wikiModel : defaultModel;
  const canAsk = Boolean(workspace?.id && activeProviderConfig && activeModel);
  const canSwitchSkills = Boolean(workspace?.id) && !pane?.asking;
  const disableReason = !workspace
    ? 'Select a workspace to start a session.'
    : usesWikiScanModel && (!wikiProviderConfig || !wikiModel)
      ? 'Build memory requires a workspace wiki-scan provider and model. Select them in Knowledge mode.'
      : !defaultProviderConfig || !defaultModel
      ? 'Configure at least one LLM provider with a model to use workspace sessions.'
      : null;
  const sessions = pane?.sessions ?? [];
  const activeSessionId = pane?.activeSessionId ?? null;
  const messages = activeSessionId ? (pane?.messages[activeSessionId] ?? []) : [];
  const executedSkillsByMessageId = activeSessionId ? (pane?.executedSkills[activeSessionId] ?? {}) : {};
  const composerHint = disableReason ?? (selectedSkill
    ? usesWikiScanModel
      ? `${selectedSkill.label} uses the workspace wiki-scan provider/model from Knowledge mode.`
      : `${selectedSkill.label} selected.`
    : 'Auto skill routing uses the first configured LLM provider/model for now.');

  async function handleCreateSession() {
    if (!workspace?.id) {
      return;
    }
    await createSession(workspace.id, 'New session', 'workspace');
  }

  async function handleAsk() {
    const question = draft.trim();
    if (!workspace?.id || !question || !activeProviderConfig || !activeModel) {
      return;
    }
    await ask({
      workspaceId: workspace.id,
      sessionId: activeSessionId ?? '',
      surface: 'workspace',
      skillName: pane?.selectedSkillName ?? undefined,
      providerId: activeProviderConfig.provider.id,
      modelId: activeModel.id,
      question,
    });
    setDraft('');
  }

  if (!workspace) {
    return (
      <section className="workspace-tab workspace-tab-empty">
        <div className="workspace-panel">
          <h2>No Workspace Selected</h2>
          <p className="workspace-panel-description">Choose a workspace from the home page to start an agent session.</p>
        </div>
      </section>
    );
  }

  return (
    <section className="workspace-tab workspace-agent-shell">
      <aside className="workspace-agent-rail workspace-panel">
        <div className="workspace-agent-rail-header">
          <p className="panel-kicker">Workspace Agent</p>
          <h2>{workspace.name}</h2>
          <p>{workspace.description || 'Start a workspace session, ask research questions, and keep the document tools one click away.'}</p>
        </div>

        <div className="workspace-mode-strip">
          <button type="button" className="workspace-mode-pill workspace-mode-pill-active">
            <MessagesSquare size={14} />
            Sessions
          </button>
          <button type="button" className="workspace-mode-pill" onClick={onSwitchMode}>
            <Library size={14} />
            Knowledge
          </button>
        </div>

        <div className="workspace-agent-session-head">
          <strong>Sessions</strong>
          <Button variant="secondary" size="sm" onClick={() => void handleCreateSession()} disabled={pane?.loading}>
            <Plus size={14} />
            New
          </Button>
        </div>

        <div className="workspace-agent-session-list">
          {sessions.length > 0 ? sessions.map((session) => (
            <button
              key={session.id}
              type="button"
              className={`workspace-agent-session-button ${session.id === activeSessionId ? 'workspace-agent-session-button-active' : ''}`}
              onClick={() => void loadMessages(workspace.id, session.id)}
            >
              <strong>{session.title || 'Untitled session'}</strong>
              <span>{session.updatedAt || session.createdAt}</span>
            </button>
          )) : (
            <p className="empty-inline">No sessions yet. Start with a question or create a blank thread.</p>
          )}
        </div>
      </aside>

      <div className="workspace-agent-thread workspace-panel">
        <div className="workspace-agent-thread-head">
          <div>
            <p className="panel-kicker">Thread</p>
            <h3>{sessions.find((session) => session.id === activeSessionId)?.title || 'New workspace session'}</h3>
          </div>
          {activeProviderConfig && activeModel ? (
            <small className="workspace-agent-provider-badge">{activeProviderConfig.provider.name} / {activeModel.modelId}</small>
          ) : null}
        </div>

        {pane?.error ? <div className="reader-error">{pane.error}</div> : null}
        {disableReason ? <div className="workspace-agent-disabled">{disableReason}</div> : null}

        <div className="workspace-agent-skill-strip">
          <button
            type="button"
            className={`workspace-agent-skill-pill ${!pane?.selectedSkillName ? 'workspace-agent-skill-pill-active' : ''}`}
            onClick={() => setSelectedSkill(workspace.id, null)}
            disabled={!canSwitchSkills}
          >
            Auto
          </button>
          {manualSkills.map((skill) => (
            <button
              key={skill.name}
              type="button"
              className={`workspace-agent-skill-pill ${pane?.selectedSkillName === skill.name ? 'workspace-agent-skill-pill-active' : ''}`}
              onClick={() => setSelectedSkill(workspace.id, skill.name)}
              title={skill.description}
              disabled={!canSwitchSkills}
            >
              {skill.label}
            </button>
          ))}
        </div>

        <div className="workspace-agent-message-list">
          {messages.length > 0 ? messages.map((message) => (
            <article key={message.id} className={`workspace-agent-message workspace-agent-message-${message.role}`}>
              {message.role === 'assistant' ? (() => {
                const executedSkill = executedSkillsByMessageId[message.id];
                const fallbackLabel = message.skillName ? formatSkillLabel(message.skillName, pane?.skills ?? []) : '';
                if (!executedSkill && !fallbackLabel) {
                  return null;
                }
                return (
                  <div className="workspace-agent-executed-skill">
                    <span className="workspace-agent-skill-badge">{executedSkill?.displayText || fallbackLabel}</span>
                    {executedSkill ? (
                      <span className="workspace-agent-executed-skill-detail">{formatExecutedSkillDetail(executedSkill)}</span>
                    ) : null}
                  </div>
                );
              })() : null}
              <div className="workspace-agent-message-meta">
                <span>{message.role === 'assistant' ? 'Assistant' : 'You'}</span>
                {message.evidenceCount > 0 ? <small>{message.evidenceCount} evidence</small> : null}
              </div>
              <div className="workspace-agent-message-body">{message.content || message.prompt}</div>
            </article>
          )) : (
            <div className="workspace-agent-empty-state">
              <Bot size={18} />
              <p>Ask the workspace agent for summaries, comparisons, or evidence-backed notes across this workspace.</p>
            </div>
          )}
        </div>

        <div className="workspace-agent-composer">
          <textarea
            className="workspace-agent-textarea"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="Ask about themes, methods, findings, or gaps across this workspace..."
            disabled={!canAsk || pane?.asking}
          />
          <div className="workspace-agent-composer-actions">
            <small>
              {composerHint}
            </small>
            <Button variant="secondary" onClick={() => void handleAsk()} disabled={!canAsk || pane?.asking || draft.trim() === ''}>
              <Send size={14} />
              {pane?.asking ? 'Asking...' : 'Ask'}
            </Button>
          </div>
        </div>
      </div>

      <aside className="workspace-agent-sidebar workspace-panel">
        <div className="workspace-agent-card">
          <div className="workspace-agent-card-head">
            <Sparkles size={16} />
            <strong>Suggested Skills</strong>
          </div>
          <p>Summarize a topic cluster, compare methods, or draft a literature gap note grounded in workspace evidence.</p>
        </div>

        <div className="workspace-agent-card">
          <div className="workspace-agent-card-head">
            <Brain size={16} />
            <strong>Memory</strong>
          </div>
          <p>{activeSessionId ? 'This thread keeps its own message history so follow-ups stay in context.' : 'Create a session to keep a persistent workspace thread.'}</p>
        </div>

        <div className="workspace-agent-card">
          <div className="workspace-agent-card-head">
            <Library size={16} />
            <strong>Evidence</strong>
          </div>
          <p>Knowledge mode still exposes documents, wiki pages, and source build status when you need to inspect raw workspace material.</p>
          <Button variant="secondary" size="sm" onClick={onSwitchMode}>
            Open Knowledge Mode
          </Button>
        </div>
      </aside>
    </section>
  );
}
