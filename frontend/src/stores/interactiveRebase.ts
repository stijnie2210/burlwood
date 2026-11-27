import { acceptHMRUpdate, defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { GetRebaseTodo, StartInteractiveRebase } from '../../bindings/burlwood/app';
import {
  firstChangedIndex,
  planProblem,
  rewrittenPushedCount,
  type RebaseRow,
} from '../utils/rebasePlan';

export const useInteractiveRebaseStore = defineStore('interactiveRebase', () => {
  const open = ref(false);
  const repoPath = ref('');
  const fromHash = ref('');
  const branch = ref('');
  const upstream = ref('');
  const rows = ref<RebaseRow[]>([]);
  const originalOrder = ref<string[]>([]);
  const starting = ref(false);

  const problem = computed(() => planProblem(rows.value));
  const changed = computed(
    () => firstChangedIndex(rows.value, originalOrder.value) < rows.value.length
  );
  const pushedRewritten = computed(() => rewrittenPushedCount(rows.value, originalOrder.value));

  async function show(repo: string, hash: string, branchName: string) {
    const todo = await GetRebaseTodo(repo, hash);
    repoPath.value = repo;
    fromHash.value = hash;
    branch.value = branchName;
    upstream.value = todo.upstream;
    rows.value = (todo.commits ?? []).map((c) => ({ ...c, action: 'pick', newMessage: c.message }));
    originalOrder.value = rows.value.map((r) => r.hash);
    starting.value = false;
    open.value = true;
  }

  function close() {
    open.value = false;
  }

  function move(index: number, by: 1 | -1) {
    const target = index + by;
    if (target < 0 || target >= rows.value.length) {
      return;
    }
    const next = [...rows.value];
    [next[index], next[target]] = [next[target], next[index]];
    rows.value = next;
  }

  async function start() {
    starting.value = true;
    try {
      return await StartInteractiveRebase(
        repoPath.value,
        fromHash.value,
        rows.value.map((r) => ({
          hash: r.hash,
          action: r.action,
          message: r.action === 'reword' ? r.newMessage : '',
        }))
      );
    } finally {
      starting.value = false;
    }
  }

  return {
    open,
    repoPath,
    branch,
    upstream,
    rows,
    starting,
    problem,
    changed,
    pushedRewritten,
    show,
    close,
    move,
    start,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useInteractiveRebaseStore, import.meta.hot));
}
