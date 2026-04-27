package main

import "testing"

func TestAppListWorkspaceAgentSkillsReturnsAgentSkills(t *testing.T) {
	t.Parallel()

	app := &App{
		agent: newWorkspaceAgentService(nil, nil, nil),
	}

	skills := app.ListWorkspaceAgentSkills()
	if len(skills) == 0 {
		t.Fatal("ListWorkspaceAgentSkills() returned no skills")
	}
	if skills[0].Name != string(WorkspaceAgentSkillAskWithEvidence) {
		t.Fatalf("skills[0].Name = %q, want %q", skills[0].Name, WorkspaceAgentSkillAskWithEvidence)
	}
}

func TestAppListWorkspaceAgentSkillsReturnsStaticCatalogWithoutAgent(t *testing.T) {
	t.Parallel()

	app := &App{}

	skills := app.ListWorkspaceAgentSkills()
	if len(skills) == 0 {
		t.Fatal("ListWorkspaceAgentSkills() returned no skills")
	}
	if skills[0].Name != string(WorkspaceAgentSkillAskWithEvidence) {
		t.Fatalf("skills[0].Name = %q, want %q", skills[0].Name, WorkspaceAgentSkillAskWithEvidence)
	}
}
