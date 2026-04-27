import { WorkspaceAgentShell } from './WorkspaceAgentShell';
import { WorkspaceKnowledgeMode, type WorkspaceKnowledgeModeProps } from './WorkspaceKnowledgeMode';

type WorkspaceTabMode = 'sessions' | 'knowledge';
type WorkspaceTabProps = Omit<WorkspaceKnowledgeModeProps, 'onSwitchMode'> & {
  mode: WorkspaceTabMode;
  onChangeMode: (mode: WorkspaceTabMode) => void;
};

export function WorkspaceTab(props: WorkspaceTabProps) {
  if (props.mode === 'sessions') {
    return (
        <WorkspaceAgentShell
          workspace={props.workspace}
          llmProviderConfigs={props.llmProviderConfigs}
          wikiScanProviderId={props.wikiScanProviderId}
          wikiScanModelId={props.wikiScanModelId}
          onSwitchMode={() => props.onChangeMode('knowledge')}
        />
      );
  }

  return (
    <WorkspaceKnowledgeMode
      {...props}
      onSwitchMode={() => props.onChangeMode('sessions')}
    />
  );
}
