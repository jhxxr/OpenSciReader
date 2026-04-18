import type { ModelRecord, ProviderRecord } from "../types/config";

export interface PDFTranslatePerformanceSettings {
  qps: number;
  poolMaxWorkers: number;
  termPoolMaxWorkers: number;
}

export interface PDFTranslatePerformancePreset
  extends PDFTranslatePerformanceSettings {
  label: string;
  reason: string;
}

const DEFAULT_PRESET: PDFTranslatePerformancePreset = {
  qps: 6,
  poolMaxWorkers: 6,
  termPoolMaxWorkers: 3,
  label: "通用默认",
  reason: "适合作为大多数在线模型接口的起始值。",
};

function normalizeText(value: string | null | undefined) {
  return (value || "").trim().toLowerCase();
}

function includesAny(value: string, keywords: string[]) {
  return keywords.some((keyword) => value.includes(keyword));
}

export function coercePerformanceValue(
  value: number | null | undefined,
  fallback: number,
) {
  const normalized = Math.floor(Number(value) || 0);
  return normalized > 0 ? normalized : fallback;
}

export function getPDFTranslatePerformancePreset(
  provider: Pick<ProviderRecord, "name" | "baseUrl"> | null | undefined,
  model: Pick<ModelRecord, "modelId"> | null | undefined,
): PDFTranslatePerformancePreset {
  const providerText = `${normalizeText(provider?.name)} ${normalizeText(provider?.baseUrl)}`;
  const modelText = normalizeText(model?.modelId);
  const combined = `${providerText} ${modelText}`;

  if (
    includesAny(combined, [
      "localhost",
      "127.0.0.1",
      "0.0.0.0",
      "ollama",
      "lmstudio",
      "xinference",
      "vllm",
    ])
  ) {
    return {
      qps: 12,
      poolMaxWorkers: 12,
      termPoolMaxWorkers: 6,
      label: "本地或自建服务",
      reason: "本地或自建接口通常能承受更高并发，可以先从更积极的值开始。",
    };
  }

  if (
    includesAny(combined, [
      ":free",
      "openrouter",
      "oneapi",
      "newapi",
      "xfree",
    ])
  ) {
    return {
      qps: 6,
      poolMaxWorkers: 6,
      termPoolMaxWorkers: 3,
      label: "公共网关或免费线路",
      reason: "这类线路波动和限流更常见，建议先用中等并发。",
    };
  }

  if (
    includesAny(modelText, [
      "gpt-5",
      "o1",
      "o3",
      "r1",
      "reason",
      "reasoning",
      "sonnet",
      "opus",
      "32b",
      "70b",
      "72b",
      "110b",
      "120b",
    ])
  ) {
    return {
      qps: 3,
      poolMaxWorkers: 3,
      termPoolMaxWorkers: 2,
      label: "重推理模型",
      reason: "慢模型或重推理模型单次延迟更高，先保守更稳。",
    };
  }

  if (
    includesAny(modelText, [
      "mini",
      "flash",
      "haiku",
      "nano",
      "8b",
      "7b",
      "gpt-oss-20b",
    ])
  ) {
    return {
      qps: 8,
      poolMaxWorkers: 8,
      termPoolMaxWorkers: 4,
      label: "轻量快模型",
      reason: "快模型通常能承受更高吞吐，适合先从较高并发开始。",
    };
  }

  return DEFAULT_PRESET;
}
