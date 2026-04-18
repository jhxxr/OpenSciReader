interface WailsPhase6App {
  ProxyTranslation: (providerId: number, modelId: number, text: string, sourceLang: string, targetLang: string) => Promise<string>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsPhase6App } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsPhase6App | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const phase6Api = {
  async proxyTranslation(providerId: number, modelId: number, text: string, sourceLang: string, targetLang: string): Promise<string> {
    const app = getApp();
    if (!app) {
      return `[Mock ${sourceLang}->${targetLang}] ${text}`;
    }
    return app.ProxyTranslation(providerId, modelId, text, sourceLang, targetLang);
  },
};
