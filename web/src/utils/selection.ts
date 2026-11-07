import type { EmailAccount } from '../types/account';

export interface SelectionResult {
  accounts: EmailAccount[];
  hasNoSelection: boolean;
  hasNoAccounts: boolean;
}

export function resolveSelectedAccounts(
  accounts: EmailAccount[],
  selectedIds: string[]
): SelectionResult {
  const selected = accounts.filter((a) => selectedIds.includes(a.id));

  return {
    accounts: selected,
    hasNoSelection: selectedIds.length === 0,
    hasNoAccounts: accounts.length === 0,
  };
}

export function selectAllState(
  accounts: EmailAccount[],
  selectedIds: string[]
): { checked: boolean; indeterminate: boolean } {
  const checked =
    accounts.length > 0 && accounts.every((a) => selectedIds.includes(a.id));

  return {
    checked,
    indeterminate: selectedIds.length > 0 && !checked,
  };
}

export function selectAllAccountIds(
  accounts: EmailAccount[],
  selectedIds: string[],
  checked: boolean
): string[] {
  if (!checked) {
    return [];
  }

  const known = new Set(accounts.map((a) => a.id));
  const kept = selectedIds.filter((id) => !known.has(id));

  return [...kept, ...accounts.map((a) => a.id)];
}
