import { PushBranch } from '../../bindings/burlwood/app';
import { useToastStore } from '../stores/toast';
import { useBranchesStore } from '../stores/branches';
import { useCommitsStore } from '../stores/commits';
import { useReposStore } from '../stores/repos';

async function reload(repoPath: string) {
  if (useReposStore().activeRepo?.path !== repoPath) {
    return;
  }
  await Promise.all([useBranchesStore().load(repoPath), useCommitsStore().load(repoPath)]);
}

// A rejected push after an amend or rebase is expected, so it becomes an explicit force-push
// step instead of an error; --force-with-lease still refuses remote work this repo hasn't seen.
export async function pushWithFeedback(repoPath: string, branch = '') {
  const toast = useToastStore();
  const res = await PushBranch(repoPath, branch, false);
  if (!res.rejected) {
    toast.success(
      res.setUpstream
        ? `Pushed ${res.branch} → ${res.upstream} (now tracking it)`
        : `Pushed ${res.branch} → ${res.upstream}`
    );
    await reload(repoPath);
    return;
  }
  toast.push(
    `${res.upstream} has commits your ${res.branch} doesn't, as after an amend or rebase. Force pushing replaces them.`,
    'error',
    0,
    {
      label: 'Force push (with lease)',
      danger: true,
      run: async () => {
        try {
          const forced = await PushBranch(repoPath, res.branch, true);
          toast.success(`Force pushed ${forced.branch} → ${forced.upstream}`);
          await reload(repoPath);
        } catch (e) {
          toast.error(e);
        }
      },
    }
  );
}
