import { ref, watch } from 'vue';

const KEY = 'burlwood:detailWidth';
const stored = localStorage.getItem(KEY);
const detailWidth = ref(stored ? parseInt(stored, 10) : 380);
watch(detailWidth, (v) => localStorage.setItem(KEY, String(v)));

export function useDetailWidth() {
  return detailWidth;
}
