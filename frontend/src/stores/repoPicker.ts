import { defineStore } from 'pinia';
import { ref } from 'vue';
import { useReposStore } from './repos';

export const useRepoPickerStore = defineStore('repoPicker', () => {
  const open = ref(false);
  const x = ref(0);
  const y = ref(0);

  async function toggle(anchor: HTMLElement) {
    if (open.value) {
      open.value = false;
      return;
    }
    await useReposStore().loadRecents();
    const rect = anchor.getBoundingClientRect();
    x.value = rect.left;
    y.value = rect.bottom + 4;
    open.value = true;
  }

  function close() {
    open.value = false;
  }

  return { open, x, y, toggle, close };
});
