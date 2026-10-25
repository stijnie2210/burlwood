import { ref, watch } from 'vue';

export type DiffLayout = 'unified' | 'split';

const LAYOUT_KEY = 'burlwood:diff-layout';
const WRAP_KEY = 'burlwood:diff-wrap';

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // the choice just won't persist
  }
}

const layout = ref<DiffLayout>(read(LAYOUT_KEY) === 'split' ? 'split' : 'unified');
const wrap = ref(read(WRAP_KEY) === 'true');
watch(layout, (v) => write(LAYOUT_KEY, v));
watch(wrap, (v) => write(WRAP_KEY, String(v)));

export function useDiffLayout() {
  return { layout, wrap };
}
