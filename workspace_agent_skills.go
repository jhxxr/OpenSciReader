package main

import (
	"fmt"
	"strings"
)

var workspaceAgentSkillCatalog = []WorkspaceAgentSkillDefinition{
	{Name: string(WorkspaceAgentSkillAskWithEvidence), Label: "Ask with evidence", Description: "Ground an answer in workspace memory and evidence.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillReadingOutputs), Label: "Reading outputs", Description: "Generate structured reading notes and summaries.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillTaskPlanning), Label: "Task planning", Description: "Turn a research goal into concrete next steps.", ManualEnabled: true, AutoEnabled: true},
	{Name: string(WorkspaceAgentSkillBuildMemory), Label: "Build memory", Description: "Run the workspace memory build pipeline.", ManualEnabled: true, WorkspaceOnly: true},
	{Name: string(WorkspaceAgentSkillCrossSource), Label: "Cross-source synthesis", Description: "Compare and synthesize across sources.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillPromoteToWiki), Label: "Promote to wiki", Description: "File useful output back into workspace memory.", ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillToolExecution), Label: "Tool execution", Description: "Run non-LLM tools when needed."},
}

func listWorkspaceAgentSkills() []WorkspaceAgentSkillDefinition {
	result := make([]WorkspaceAgentSkillDefinition, len(workspaceAgentSkillCatalog))
	copy(result, workspaceAgentSkillCatalog)
	return result
}

func getWorkspaceAgentSkillDefinition(name string) (WorkspaceAgentSkillDefinition, bool) {
	trimmedName := strings.TrimSpace(name)
	for _, skill := range workspaceAgentSkillCatalog {
		if skill.Name == trimmedName {
			return skill, true
		}
	}
	return WorkspaceAgentSkillDefinition{}, false
}

func newAutoWorkspaceAgentSkill(name WorkspaceAgentSkillName, reason string) WorkspaceAgentExecutedSkill {
	definition, ok := getWorkspaceAgentSkillDefinition(string(name))
	if !ok {
		return WorkspaceAgentExecutedSkill{}
	}
	return WorkspaceAgentExecutedSkill{
		Name:        definition.Name,
		Label:       definition.Label,
		RoutedBy:    "auto",
		Reason:      reason,
		DisplayText: definition.Label,
	}
}

func isWorkspaceAgentSkillAvailableOnSurface(definition WorkspaceAgentSkillDefinition, surface string) bool {
	switch strings.TrimSpace(surface) {
	case string(WorkspaceAgentSurfaceReader):
		return definition.ReaderEnabled && !definition.WorkspaceOnly
	case "", string(WorkspaceAgentSurfaceWorkspace):
		return true
	default:
		return false
	}
}

func newSurfaceSkillUnavailableError(skillName string, surface string) error {
	return fmt.Errorf("workspace agent skill %s is not available on %s surface", skillName, strings.TrimSpace(surface))
}

func tryAutoWorkspaceAgentSkill(input WorkspaceAgentAskInput, name WorkspaceAgentSkillName, reason string) (WorkspaceAgentExecutedSkill, bool) {
	definition, ok := getWorkspaceAgentSkillDefinition(string(name))
	if !ok || !definition.AutoEnabled || !isWorkspaceAgentSkillAvailableOnSurface(definition, input.Surface) {
		return WorkspaceAgentExecutedSkill{}, false
	}
	return newAutoWorkspaceAgentSkill(name, reason), true
}

func resolveWorkspaceAgentSkill(input WorkspaceAgentAskInput) (WorkspaceAgentExecutedSkill, error) {
	explicit := strings.TrimSpace(input.SkillName)
	if explicit != "" {
		definition, ok := getWorkspaceAgentSkillDefinition(explicit)
		if !ok {
			return WorkspaceAgentExecutedSkill{}, fmt.Errorf("unsupported workspace agent skill: %s", explicit)
		}
		if !definition.ManualEnabled {
			return WorkspaceAgentExecutedSkill{}, fmt.Errorf("unsupported workspace agent skill: %s", explicit)
		}
		if !isWorkspaceAgentSkillAvailableOnSurface(definition, input.Surface) {
			return WorkspaceAgentExecutedSkill{}, newSurfaceSkillUnavailableError(explicit, input.Surface)
		}
		return WorkspaceAgentExecutedSkill{
			Name:        definition.Name,
			Label:       definition.Label,
			RoutedBy:    "manual",
			Reason:      "user_selected",
			DisplayText: definition.Label,
		}, nil
	}

	return routeWorkspaceAgentSkill(input), nil
}

func routeWorkspaceAgentSkill(input WorkspaceAgentAskInput) WorkspaceAgentExecutedSkill {
	question := strings.ToLower(strings.TrimSpace(input.Question))
	if input.Surface == string(WorkspaceAgentSurfaceReader) && (strings.Contains(question, "summary") || strings.Contains(question, "summarize") || strings.Contains(question, "notes")) {
		if skill, ok := tryAutoWorkspaceAgentSkill(input, WorkspaceAgentSkillReadingOutputs, "reader_summary_request"); ok {
			return skill
		}
	}
	if strings.Contains(question, "plan") || strings.Contains(question, "next step") || strings.Contains(question, "next steps") {
		if skill, ok := tryAutoWorkspaceAgentSkill(input, WorkspaceAgentSkillTaskPlanning, "planning_language"); ok {
			return skill
		}
	}
	if strings.Contains(question, "compare") || strings.Contains(question, "across") || strings.Contains(question, "synthesis") {
		if skill, ok := tryAutoWorkspaceAgentSkill(input, WorkspaceAgentSkillCrossSource, "cross_source_language"); ok {
			return skill
		}
	}
	return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillAskWithEvidence, "default_grounded_answer")
}

func buildWorkspaceAgentSkillPrompt(mode string, input WorkspaceAgentAskInput, recentMessages []WorkspaceAgentMessage) string {
	baseQuestion := buildWorkspaceAgentQueryQuestion(strings.TrimSpace(input.Question), input, recentMessages)
	return strings.TrimSpace(mode) + ":\n\n" + baseQuestion
}
