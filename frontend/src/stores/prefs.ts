import { acceptHMRUpdate, defineStore } from 'pinia';
import { ref } from 'vue';
import { GetPrefs, SavePrefs } from '../../wailsjs/go/main/App';
import type { repo } from '../../wailsjs/go/models';

export type AppPrefs = repo.AppPrefs;

export const usePrefsStore = defineStore('prefs', () => {
  const prefs = ref<AppPrefs>({
    editor: '',
    autoFetchEnabled: true,
    autoFetchIntervalSecs: 60,
    commitGraphLimit: 2000,
    diffContextLines: 3,
    dateFormat: 'relative',
    pullMode: 'ff',
  });
  const loaded = ref(false);
  const modalOpen = ref(false);
  const now = ref(Date.now());

  setInterval(() => {
    now.value = Date.now();
  }, 60_000);

  async function load() {
    try {
      prefs.value = await GetPrefs();
    } finally {
      loaded.value = true;
    }
  }

  async function save(updates: Partial<AppPrefs>) {
    const next = { ...prefs.value, ...updates };
    await SavePrefs(next);
    prefs.value = next;
  }

  function openModal() {
    modalOpen.value = true;
  }

  function closeModal() {
    modalOpen.value = false;
  }

  return { prefs, loaded, modalOpen, now, load, save, openModal, closeModal };
});

// Without this, editing a store during `wails dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(usePrefsStore, import.meta.hot));
}
