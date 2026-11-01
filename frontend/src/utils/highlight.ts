import Prism from 'prismjs';
import 'prismjs/components/prism-markup';
import 'prismjs/components/prism-css';
import 'prismjs/components/prism-clike';
import 'prismjs/components/prism-javascript';
import 'prismjs/components/prism-typescript';
import 'prismjs/components/prism-jsx';
import 'prismjs/components/prism-tsx';
import 'prismjs/components/prism-json';
import 'prismjs/components/prism-yaml';
import 'prismjs/components/prism-markdown';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-go';
import 'prismjs/components/prism-rust';
import 'prismjs/components/prism-python';
import 'prismjs/components/prism-java';
import 'prismjs/components/prism-c';
import 'prismjs/components/prism-cpp';
import 'prismjs/components/prism-csharp';
import 'prismjs/components/prism-ruby';
import 'prismjs/components/prism-markup-templating';
import 'prismjs/components/prism-php';
import 'prismjs/components/prism-sql';
import 'prismjs/components/prism-docker';
import 'prismjs/components/prism-toml';

const EXT_TO_LANG: Record<string, string> = {
  ts: 'typescript',
  tsx: 'tsx',
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  jsx: 'jsx',
  vue: 'markup',
  json: 'json',
  yml: 'yaml',
  yaml: 'yaml',
  md: 'markdown',
  markdown: 'markdown',
  sh: 'bash',
  bash: 'bash',
  zsh: 'bash',
  go: 'go',
  rs: 'rust',
  py: 'python',
  java: 'java',
  c: 'c',
  h: 'c',
  cpp: 'cpp',
  cc: 'cpp',
  cxx: 'cpp',
  hpp: 'cpp',
  cs: 'csharp',
  rb: 'ruby',
  php: 'php',
  sql: 'sql',
  html: 'markup',
  htm: 'markup',
  xml: 'markup',
  svg: 'markup',
  css: 'css',
  scss: 'css',
  toml: 'toml',
  dockerfile: 'docker',
};

export function languageForPath(path: string): string | null {
  const base = path.split('/').pop() ?? path;
  if (base.toLowerCase() === 'dockerfile') {
    return 'docker';
  }
  const dot = base.lastIndexOf('.');
  if (dot === -1) {
    return null;
  }
  const ext = base.slice(dot + 1).toLowerCase();
  const lang = EXT_TO_LANG[ext];
  return lang && Prism.languages[lang] ? lang : null;
}

// A run of text with the classes of every Prism token around it, outermost first. Nested
// tokens stay nested when rendered, so the theme's innermost colour wins as it would in HTML.
export interface CodeSegment {
  text: string;
  tokens: string[][];
}

// Consecutive segments inside or outside the changed range; a changed run renders as one mark.
export interface CodeRun {
  changed: boolean;
  segments: CodeSegment[];
}

export function highlightSegments(content: string, lang: string | null): CodeSegment[] {
  const grammar = lang ? Prism.languages[lang] : undefined;
  if (!lang || !grammar || content === '') {
    return content === '' ? [] : [{ text: content, tokens: [] }];
  }
  // The same steps as Prism.highlight: markup-templating languages (PHP) tokenize in these hooks.
  const env: {
    code: string;
    grammar: Prism.Grammar;
    language: string;
    tokens?: Prism.TokenStream;
  } = { code: content, grammar, language: lang };
  Prism.hooks.run('before-tokenize', env);
  env.tokens = Prism.tokenize(env.code, env.grammar);
  Prism.hooks.run('after-tokenize', env);
  const out: CodeSegment[] = [];
  flatten(env.tokens, [], out);
  return out;
}

function flatten(stream: Prism.TokenStream, tokens: string[][], out: CodeSegment[]) {
  if (typeof stream === 'string') {
    if (stream !== '') {
      out.push({ text: stream, tokens });
    }
    return;
  }
  if (Array.isArray(stream)) {
    for (const part of stream) {
      flatten(part, tokens, out);
    }
    return;
  }
  const alias = stream.alias ? ([] as string[]).concat(stream.alias) : [];
  flatten(stream.content, [...tokens, ['token', stream.type, ...alias]], out);
}

// The whole line is tokenized once and the segments are cut at the change, so a change inside
// a string or comment keeps that token's colour on both sides.
export function highlightChange(
  content: string,
  lang: string | null,
  change?: { start: number; end: number }
): CodeRun[] {
  const segments = highlightSegments(content, lang);
  if (!change || change.start === change.end) {
    return segments.length ? [{ changed: false, segments }] : [];
  }
  const runs: CodeRun[] = [];
  const push = (changed: boolean, segment: CodeSegment) => {
    if (segment.text === '') {
      return;
    }
    const last = runs[runs.length - 1];
    if (last && last.changed === changed) {
      last.segments.push(segment);
    } else {
      runs.push({ changed, segments: [segment] });
    }
  };
  let offset = 0;
  for (const seg of segments) {
    const end = offset + seg.text.length;
    const cuts = [
      offset,
      Math.min(Math.max(change.start, offset), end),
      Math.min(Math.max(change.end, offset), end),
      end,
    ];
    for (let i = 0; i < 3; i++) {
      push(i === 1, {
        text: seg.text.slice(cuts[i] - offset, cuts[i + 1] - offset),
        tokens: seg.tokens,
      });
    }
    offset = end;
  }
  return runs;
}
