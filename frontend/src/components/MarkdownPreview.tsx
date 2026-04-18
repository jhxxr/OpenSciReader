import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';

export function MarkdownPreview({ content, placeholder = '暂无内容。' }: { content: string; placeholder?: string }) {
  if (!content.trim()) {
    return <div className="markdown-preview markdown-preview-empty">{placeholder}</div>;
  }

  return (
    <div className="markdown-preview">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
        components={{
          a: ({ node: _node, ...props }) => <a {...props} rel="noreferrer" target="_blank" />,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
