import { useEffect, useMemo, useState } from 'react';
import { Search, Sparkles, X } from 'lucide-react';
import { findModelPreset } from '../lib/aiCatalog';
import type { DiscoveredModel } from '../types/config';
import { Button } from './ui/Button';

interface ModelDiscoveryModalProps {
  open: boolean;
  providerName: string;
  existingModelIds: string[];
  models: DiscoveredModel[];
  isLoading: boolean;
  error: string | null;
  onFetch: () => void;
  onClose: () => void;
  onApply: (modelIds: string[]) => void;
}

function normalizeModelID(value: string): string {
  return value.trim().toLowerCase();
}

export function ModelDiscoveryModal({
  open,
  providerName,
  existingModelIds,
  models,
  isLoading,
  error,
  onFetch,
  onClose,
  onApply,
}: ModelDiscoveryModalProps) {
  const [searchText, setSearchText] = useState('');
  const [selectedModelIds, setSelectedModelIds] = useState<string[]>([]);

  useEffect(() => {
    if (!open) {
      return;
    }
    setSearchText('');
    setSelectedModelIds([]);
  }, [open, providerName]);

  const existingModelIdSet = useMemo(
    () => new Set(existingModelIds.map((item) => normalizeModelID(item))),
    [existingModelIds],
  );

  const filteredModels = useMemo(() => {
    const normalizedSearch = searchText.trim().toLowerCase();
    if (!normalizedSearch) {
      return models;
    }

    return models.filter((item) => {
      const haystacks = [item.id, item.name, item.ownedBy].map((value) => value.toLowerCase());
      return haystacks.some((value) => value.includes(normalizedSearch));
    });
  }, [models, searchText]);

  if (!open) {
    return null;
  }

  const toggleModel = (modelId: string) => {
    setSelectedModelIds((current) =>
      current.includes(modelId) ? current.filter((item) => item !== modelId) : [...current, modelId],
    );
  };

  return (
    <div className="modal-backdrop" role="presentation" onClick={onClose}>
      <section className="modal discovery-modal" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <p className="panel-kicker">Model Discovery</p>
            <h2>{providerName} /models</h2>
          </div>
          <Button variant="ghost" onClick={onClose}>
            <X size={16} />
            关闭
          </Button>
        </div>

        <div className="discovery-toolbar">
          <Button variant="secondary" onClick={onFetch} disabled={isLoading}>
            <Sparkles size={14} />
            {isLoading ? '拉取中...' : models.length > 0 ? '重新拉取' : '开始拉取'}
          </Button>
          <label className="discovery-search">
            <Search size={14} />
            <input
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              placeholder="搜索 modelId / 名称"
            />
          </label>
          <div className="discovery-summary">
            <strong>{models.length}</strong>
            <span>已发现</span>
            <strong>{selectedModelIds.length}</strong>
            <span>待添加</span>
          </div>
        </div>

        {error ? <div className="reader-error">{error}</div> : null}

        <div className="discovery-list">
          {filteredModels.length > 0 ? (
            filteredModels.map((item) => {
              const exists = existingModelIdSet.has(normalizeModelID(item.id));
              const matchedPreset = findModelPreset(item.id);
              return (
                <label
                  key={item.id}
                  className={`discovery-card ${exists ? 'discovery-card-disabled' : ''}`}
                >
                  <input
                    type="checkbox"
                    checked={selectedModelIds.includes(item.id)}
                    disabled={exists}
                    onChange={() => toggleModel(item.id)}
                  />
                  <div className="discovery-card-main">
                    <div className="discovery-card-title">
                      <strong>{item.id}</strong>
                      {exists ? <span className="inline-badge inline-badge-muted">已添加</span> : null}
                      {matchedPreset ? <span className="inline-badge">已匹配预设</span> : null}
                    </div>
                    <p>{item.name || item.id}</p>
                    <div className="meta-row">
                      {item.ownedBy ? <small>{item.ownedBy}</small> : null}
                      {matchedPreset?.contextWindow ? <small>上下文 {matchedPreset.contextWindow.toLocaleString()}</small> : null}
                    </div>
                  </div>
                </label>
              );
            })
          ) : (
            <p className="empty-inline">
              {models.length > 0 ? '没有匹配当前搜索的模型。' : '还没有拉取到模型列表。'}
            </p>
          )}
        </div>

        <div className="modal-actions">
          <Button variant="ghost" onClick={onClose}>
            取消
          </Button>
          <Button onClick={() => onApply(selectedModelIds)} disabled={selectedModelIds.length === 0}>
            添加 {selectedModelIds.length} 个模型
          </Button>
        </div>
      </section>
    </div>
  );
}
