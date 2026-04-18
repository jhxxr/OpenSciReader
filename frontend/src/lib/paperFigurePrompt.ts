// Adapted from zotero-AI-Butler:
// https://github.com/steven-jianhao-li/zotero-AI-Butler

export const PAPER_IMAGE_SUMMARY_PROMPT = `请阅读我提供的论文内容，提取用于生成"学术概念海报"的关键视觉信息。

请确保描述具体、形象，适合画面呈现。
请输出如下内容（只输出内容，不要废话），使用\${language}：
1. 研究问题：提到的核心问题
2. 创新方法：论文提出的主要方法或技术，要找到Aha！的那个点。
3. 工作流程：从输入到输出的处理流程
4. 关键结果：主要实验发现或性能提升
5. 应用价值：该研究的实际意义
---
论文内容如下：
\${context}`;

export const PAPER_IMAGE_GENERATION_PROMPT = `根据"\${summaryForImage}"，生成一张学术论文概念图，清晰展示以下内容：

研究问题：提到的核心问题
创新方法：论文提出的主要方法或技术
工作流程：从输入到输出的处理流程
关键结果：主要实验发现或性能提升
应用价值：该研究的实际意义
论文标题：\${title}
要求：
**设计要求 (Design Guidelines - STRICTLY FOLLOW):**
1.  **艺术风格 (Style):**
    *   Modern Minimalist Tech Infographic (现代极简科技信息图).
    *   Flat vector illustration with subtle isometric elements (带有微妙等距元素的扁平矢量插画).
    *   High-quality corporate Memphis design style (高质量企业级孟菲斯设计风格).
    *   Clean lines, geometric shapes (线条干净，几何形状).
2.  **构图 (Composition):**
    *   **Layout:** Central composition or Left-to-Right Process Flow (居中构图或从左到右的流程).
    *   **Background:** Clean, solid off-white or very light grey background (#F5F5F7). No clutter. (干净的米白或浅灰背景，无杂乱).
    *   **Structure:** Organize elements logically like a presentation slide or a academic poster.
3.  **配色方案 (Color Palette):**
    *   Primary: Deep Academic Blue (深学术蓝) & Slate Grey (板岩灰).
    *   Accent: Vibrant Orange or Teal for highlights (活力橙或青色用于高亮).
    *   High contrast, professional color grading (高对比度，专业调色).
4.  **文字渲染 (Text Rendering):**
    *   Use Times New Roman font for English.
    *   Use SimSun font for Chinese.
    *   Main text language: \${language} (User defined language).
    *   The title does not need to be reflected in the figure.
    *   The text, especially Chinese, needs to be clear and free of garbled characters.
5.  **负面提示 (Negative Prompt - Avoid these):**
    *   No photorealism (不要照片写实风格).
    *   No messy sketches (不要草图).
    *   No blurry text (不要模糊文字).
    *   No chaotic background (不要混乱背景).
**Generation Instructions:**
Generate an academic infographic poster.`;

export function buildPaperImageSummaryPrompt(
  context: string,
  language: string,
): string {
  return PAPER_IMAGE_SUMMARY_PROMPT.replace(/\$\{context\}/g, context).replace(
    /\$\{language\}/g,
    language,
  );
}

export function buildPaperImageSummaryInstruction(language: string): string {
  return `请阅读上面提供的论文内容，提取用于生成"学术概念海报"的关键视觉信息。

请确保描述具体、形象，适合画面呈现。
请输出如下内容（只输出内容，不要废话），使用${language}：
1. 研究问题：提到的核心问题
2. 创新方法：论文提出的主要方法或技术，要找到Aha！的那个点。
3. 工作流程：从输入到输出的处理流程
4. 关键结果：主要实验发现或性能提升
5. 应用价值：该研究的实际意义`;
}

export function buildPaperImageGenerationPrompt(
  summaryForImage: string,
  title: string,
  language: string,
  extraInstructions: string,
): string {
  const prompt = PAPER_IMAGE_GENERATION_PROMPT.replace(
    /\$\{summaryForImage\}/g,
    summaryForImage,
  )
    .replace(/\$\{title\}/g, title)
    .replace(/\$\{language\}/g, language);

  if (!extraInstructions.trim()) {
    return prompt;
  }

  return `${prompt}\n\n附加要求：\n${extraInstructions.trim()}`;
}
