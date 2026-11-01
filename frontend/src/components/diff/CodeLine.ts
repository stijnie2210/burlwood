import { h, type FunctionalComponent, type VNode } from 'vue';
import { highlightChange, type CodeSegment } from '../../utils/highlight';

interface Props {
  content: string;
  lang: string | null;
  change?: { start: number; end: number };
}

function segmentNode(segment: CodeSegment): VNode | string {
  return segment.tokens.reduceRight<VNode | string>(
    (child, classes) => h('span', { class: classes }, [child]),
    segment.text
  );
}

// Renders a syntax-highlighted line as elements, never as an HTML string: file contents only
// ever become text nodes.
const CodeLine: FunctionalComponent<Props> = (props) =>
  highlightChange(props.content, props.lang, props.change)
    .map((run) =>
      run.changed
        ? h('mark', { class: 'diff-change' }, run.segments.map(segmentNode))
        : run.segments.map(segmentNode)
    )
    .flat();

CodeLine.props = ['content', 'lang', 'change'];

export default CodeLine;
