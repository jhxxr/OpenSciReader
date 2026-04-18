import { EventsEmit } from '../../wailsjs/runtime/runtime';
import type { GatewayContextData, GatewayStreamEvent } from '../types/gateway';
import type { FigureGenerationResult } from '../types/drawing';

interface WailsGatewayApp {
  StreamLLMChat: (providerId: number, modelId: number, prompt: string, contextData: GatewayContextData) => Promise<string>;
  GenerateResearchFigure: (providerId: number, modelId: number, prompt: string, contextData: GatewayContextData, workspaceId: string) => Promise<FigureGenerationResult>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsGatewayApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsGatewayApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const gatewayApi = {
  async streamLLMChat(providerId: number, modelId: number, prompt: string, contextData: GatewayContextData): Promise<string> {
    const app = getApp();
    if (app) {
      return app.StreamLLMChat(providerId, modelId, prompt, contextData);
    }

    const requestId = `mock-${Date.now()}`;
    const eventName = `gateway:chat:${requestId}`;
    const chunks = ['这是本地 mock 响应。', ' ', '流式聊天 UI 已完成，', '现在只差真实 Wails Provider 请求。'];
    window.setTimeout(() => {
      chunks.forEach((chunk, index) => {
        window.setTimeout(() => {
          EventsEmit(eventName, { requestId, type: 'chunk', content: chunk } satisfies GatewayStreamEvent);
        }, index * 100);
      });
      window.setTimeout(() => {
        EventsEmit(eventName, { requestId, type: 'done' } satisfies GatewayStreamEvent);
      }, chunks.length * 100 + 50);
    }, 20);
    return requestId;
  },
  async generateResearchFigure(providerId: number, modelId: number, prompt: string, contextData: GatewayContextData, workspaceId: string): Promise<FigureGenerationResult> {
    const app = getApp();
    if (app) {
      return app.GenerateResearchFigure(providerId, modelId, prompt, contextData, workspaceId);
    }

    return {
      mimeType: 'image/png',
      dataUrl: contextData.snapshot || 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9s2l9iQAAAAASUVORK5CYII=',
      prompt,
    };
  },
};
