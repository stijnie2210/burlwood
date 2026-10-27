import { acceptHMRUpdate, defineStore } from 'pinia';
import { ref } from 'vue';

export const useLogModalStore = defineStore('logModal', () => {
  const open = ref(false);
  const title = ref('');
  const content = ref('');

  function show(t: string, c: string) {
    title.value = t;
    content.value = c;
    open.value = true;
  }

  function close() {
    open.value = false;
  }

  return { open, title, content, show, close };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useLogModalStore, import.meta.hot));
}
