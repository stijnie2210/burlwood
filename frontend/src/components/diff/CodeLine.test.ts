import { describe, it, expect } from 'vitest';
import { h } from 'vue';
import { renderToString } from 'vue/server-renderer';
import CodeLine from './CodeLine';

const render = (content: string, lang: string | null, change?: { start: number; end: number }) =>
  renderToString(h(CodeLine, { content, lang, change }));

describe('CodeLine', () => {
  it('renders file contents as text, never as markup', async () => {
    const html = await render('<img src=x onerror="alert(1)">', null);
    expect(html).not.toContain('<img');
    expect(html).toContain('&lt;img');
  });

  it('nests token spans and wraps the change in one mark', async () => {
    const line = 'x = "ab cd";';
    const start = line.indexOf('cd');
    const html = await render(line, 'javascript', { start, end: start + 2 });
    expect(html).toContain('<mark class="diff-change"><span class="token string">cd</span></mark>');
    expect(html.match(/<mark/g)).toHaveLength(1);
  });
});
