import type { PDFTranslateChunkStatus, PDFTranslateJobSnapshot } from "../types/pdfTranslate";

export function getJobProgressPercent(job: PDFTranslateJobSnapshot | null | undefined) {
  if (!job) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round((job.overallProgress ?? 0) * 1000) / 10));
}

export function formatRelativeTimestamp(value: string) {
  const ts = Date.parse(value);
  if (Number.isNaN(ts)) {
    return value;
  }
  const deltaSeconds = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (deltaSeconds < 5) {
    return "刚刚";
  }
  if (deltaSeconds < 60) {
    return `${deltaSeconds} 秒前`;
  }
  const deltaMinutes = Math.round(deltaSeconds / 60);
  if (deltaMinutes < 60) {
    return `${deltaMinutes} 分钟前`;
  }
  const deltaHours = Math.round(deltaMinutes / 60);
  if (deltaHours < 24) {
    return `${deltaHours} 小时前`;
  }
  return new Date(ts).toLocaleString();
}

export function getJobStatusLabel(status: PDFTranslateJobSnapshot["status"]) {
  switch (status) {
    case "queued":
      return "等待启动";
    case "running":
      return "运行中";
    case "completed":
      return "已完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    default:
      return status;
  }
}

export function getChunkStatusLabel(status: PDFTranslateChunkStatus["status"]) {
  switch (status) {
    case "queued":
      return "等待";
    case "running":
      return "进行中";
    case "completed":
      return "完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "取消";
    default:
      return status;
  }
}

export function getJobStageLabel(stage: string | undefined) {
  const normalized = normalizeStage(stage);
  if (!normalized) {
    return "";
  }
  const mappings: Array<[string, string]> = [
    ["parse paragraphs", "解析段落"],
    ["paragraph", "解析段落"],
    ["layout parser", "解析版面"],
    ["layout", "解析版面"],
    ["detect scanned", "检测扫描页"],
    ["styles and formulas", "识别样式与公式"],
    ["formula", "识别样式与公式"],
    ["font", "准备字体"],
    ["translate", "请求模型翻译"],
    ["term", "提取术语"],
    ["typesetting", "重新排版"],
    ["create pdf", "生成 PDF"],
    ["pdf", "生成 PDF"],
  ];
  for (const [pattern, label] of mappings) {
    if (normalized.includes(pattern)) {
      return label;
    }
  }
  return stage || "";
}

export function getJobPrimaryStatus(job: PDFTranslateJobSnapshot) {
  const stageLabel = getJobStageLabel(job.currentStage);
  if (job.status === "running" && stageLabel) {
    return stageLabel;
  }
  return getJobStatusLabel(job.status);
}

export function getJobSecondaryStatus(job: PDFTranslateJobSnapshot) {
  if (job.status === "failed") {
    return job.error || "任务失败";
  }
  if (job.status === "cancelled") {
    return "任务已取消，可在首页清理旧任务或重新发起。";
  }
  if (job.status === "completed") {
    return job.mode === "preview"
      ? "本次预览 chunk 已完成，右侧译文页会按块替换。"
      : "整本导出已完成，可以直接下载译文 / 混排 / 对照 PDF。";
  }
  const stageLabel = getJobStageLabel(job.currentStage);
  const stageCurrent = job.stageCurrent ?? 0;
  const stageTotal = job.stageTotal ?? 0;
  if (stageLabel && stageTotal > 0) {
    return `${stageLabel} · ${stageCurrent}/${stageTotal}`;
  }
  if (job.status === "running" && getJobProgressPercent(job) <= 0) {
    if (job.mode === "preview" && job.chunks.length <= 1) {
      return "当前文档只有 1 个 chunk，需要等本块完成后右侧译文页才会整体替换。";
    }
    return "首次运行可能会下载字体、OCR 模型并建立缓存，短时间没有明显进度变化是正常的。";
  }
  return "";
}

export function getJobLiveHint(job: PDFTranslateJobSnapshot) {
  const recent = formatRelativeTimestamp(job.updatedAt || job.startedAt || job.createdAt);
  if (job.status === "running") {
    return `任务仍在运行，最近一次状态更新时间：${recent}`;
  }
  if (job.finishedAt) {
    return `任务结束时间：${formatRelativeTimestamp(job.finishedAt)}`;
  }
  return `最近一次状态更新时间：${recent}`;
}

export function getJobSummaryLine(job: PDFTranslateJobSnapshot) {
  const completedChunks = job.chunks.filter((chunk) => chunk.status === "completed").length;
  return `${getJobStatusLabel(job.status)} · ${getJobProgressPercent(job).toFixed(1)}% · ${completedChunks}/${job.chunks.length} 个 chunk`;
}

export function shouldUseIndeterminateProgress(job: PDFTranslateJobSnapshot) {
  return job.status === "running" && getJobProgressPercent(job) <= 0;
}

export function getChunkCaption(chunk: PDFTranslateChunkStatus) {
  return `P${chunk.startPage}-${chunk.endPage}`;
}

function normalizeStage(stage: string | undefined) {
  return (stage || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, " ");
}
