package main

import (
	"context"
	"fmt"
	"strings"

	"OpenSciReader/internal/translator"
)

type PDFTranslateStartInput struct {
	PDFPath            string `json:"pdfPath"`
	PageCount          int    `json:"pageCount"`
	ItemID             string `json:"itemId"`
	ItemTitle          string `json:"itemTitle"`
	SourceLang         string `json:"sourceLang"`
	TargetLang         string `json:"targetLang"`
	Mode               string `json:"mode"`
	PreviewChunkPages  int    `json:"previewChunkPages"`
	MaxPagesPerPart    int    `json:"maxPagesPerPart"`
	QPS                int    `json:"qps"`
	PoolMaxWorkers     int    `json:"poolMaxWorkers"`
	TermPoolMaxWorkers int    `json:"termPoolMaxWorkers"`
	RetryJobID         string `json:"retryJobId"`
	ReusePreviewJobID  string `json:"reusePreviewJobId"`
	LLMProviderID      int64  `json:"llmProviderId"`
	LLMModelID         int64  `json:"llmModelId"`
}

func (a *App) StartPDFTranslate(input PDFTranslateStartInput) (translator.JobSnapshot, error) {
	return a.startPDFTranslate(a.ctx, input)
}

func (a *App) CancelPDFTranslate(jobID string) (translator.JobSnapshot, error) {
	if a.translator == nil {
		return translator.JobSnapshot{}, fmt.Errorf("pdf translate service is unavailable")
	}
	return a.translator.Cancel(strings.TrimSpace(jobID))
}

func (a *App) GetPDFTranslateStatus(jobID string) (translator.JobSnapshot, error) {
	if a.translator == nil {
		return translator.JobSnapshot{}, fmt.Errorf("pdf translate service is unavailable")
	}
	return a.translator.GetStatus(strings.TrimSpace(jobID))
}

func (a *App) ListPDFTranslateJobs() ([]translator.JobSnapshot, error) {
	if a.translator == nil {
		return nil, fmt.Errorf("pdf translate service is unavailable")
	}
	return a.translator.ListJobs()
}

func (a *App) DeletePDFTranslateJob(jobID string) error {
	if a.translator == nil {
		return fmt.Errorf("pdf translate service is unavailable")
	}
	return a.translator.DeleteJob(strings.TrimSpace(jobID))
}

func (a *App) startPDFTranslate(ctx context.Context, input PDFTranslateStartInput) (translator.JobSnapshot, error) {
	if a == nil || a.translator == nil || a.store == nil {
		return translator.JobSnapshot{}, fmt.Errorf("pdf translate service is unavailable")
	}

	if runtime, err := a.store.GetPDFTranslateRuntimeConfig(ctx); err == nil {
		if runtime.Status != PDFTranslateRuntimeStatusValid {
			if strings.TrimSpace(runtime.LastValidationError) != "" {
				return translator.JobSnapshot{}, fmt.Errorf("pdf translate runtime unavailable: %s", runtime.LastValidationError)
			}
			return translator.JobSnapshot{}, fmt.Errorf("pdf translate runtime is not installed")
		}
	}

	provider, err := a.store.GetProviderSecret(ctx, input.LLMProviderID)
	if err != nil {
		return translator.JobSnapshot{}, err
	}
	model, err := a.store.GetModel(ctx, input.LLMModelID)
	if err != nil {
		return translator.JobSnapshot{}, err
	}
	if model.ProviderID != provider.ID {
		return translator.JobSnapshot{}, fmt.Errorf("model %d does not belong to provider %d", model.ID, provider.ID)
	}
	if provider.Type != ProviderTypeLLM {
		return translator.JobSnapshot{}, fmt.Errorf("provider %s is not an LLM provider", provider.Name)
	}
	if !provider.IsActive {
		return translator.JobSnapshot{}, fmt.Errorf("provider %s is inactive", provider.Name)
	}
	if strings.TrimSpace(provider.APIKey) == "" {
		return translator.JobSnapshot{}, fmt.Errorf("provider %s has no api key", provider.Name)
	}

	return a.translator.Start(ctx, translator.StartRequest{
		PDFPath:            normalizeAttachmentPath(input.PDFPath),
		PageCount:          input.PageCount,
		ItemID:             strings.TrimSpace(input.ItemID),
		ItemTitle:          strings.TrimSpace(input.ItemTitle),
		SourceLang:         strings.TrimSpace(input.SourceLang),
		TargetLang:         strings.TrimSpace(input.TargetLang),
		Mode:               translator.Mode(strings.TrimSpace(input.Mode)),
		PreviewChunkPages:  input.PreviewChunkPages,
		MaxPagesPerPart:    input.MaxPagesPerPart,
		QPS:                input.QPS,
		PoolMaxWorkers:     input.PoolMaxWorkers,
		TermPoolMaxWorkers: input.TermPoolMaxWorkers,
		RetryOfJobID:       strings.TrimSpace(input.RetryJobID),
		ReusePreviewJobID:  strings.TrimSpace(input.ReusePreviewJobID),
		Provider: translator.ProviderConfig{
			ProviderID:   provider.ID,
			ProviderName: provider.Name,
			BaseURL:      provider.BaseURL,
			APIKey:       provider.APIKey,
			ModelID:      model.ModelID,
		},
	})
}
