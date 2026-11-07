import type { AccountImportRow, EmailAccount, ProtocolType } from '../types/account';

const ACCOUNTS_STORAGE_KEY = 'astramail_accounts_v1';

export const storage = {
  getAccounts(): EmailAccount[] {
    try {
      const data = localStorage.getItem(ACCOUNTS_STORAGE_KEY);
      if (!data) return [];
      const parsed = JSON.parse(data);
      if (!Array.isArray(parsed)) return [];

      return parsed.map((acc: any) => ({
        ...acc,
        clientId: acc.clientId || acc.client_id || undefined,
        refreshToken: acc.refreshToken || acc.refresh_token || undefined,
        imapHost: acc.imapHost || acc.imap_host || undefined,
        imapPort: acc.imapPort || acc.imap_port ? Number(acc.imapPort || acc.imap_port) : undefined,
      }));
    } catch (e) {
      console.error('Failed to load accounts from localStorage', e);
      return [];
    }
  },

  saveAccounts(accounts: EmailAccount[]): void {
    try {
      localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(accounts));
    } catch (e) {
      console.error('Failed to save accounts to localStorage', e);
    }
  },

  parseAccountLines(text: string): { rows: AccountImportRow[]; accounts: EmailAccount[] } {
    const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);
    const rows: AccountImportRow[] = [];
    const accounts: EmailAccount[] = [];

    for (const line of lines) {
      const parts = line.split('----').map((p) => p.trim());
      const email = parts[0] || '';
      const password = parts[1] || '';
      const clientId = parts[2] || '';
      const refreshToken = parts[3] || '';

      const isEmailLike = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);

      if (!email || !isEmailLike) {
        rows.push({
          email,
          password,
          clientId,
          refreshToken,
          rawLine: line,
          isValid: false,
          error: '邮箱格式无效',
        });
        continue;
      }

      let protocol: ProtocolType = 'imap';
      const hasImap = Boolean(password);
      const hasGraph = Boolean(clientId && refreshToken);

      if (hasImap && hasGraph) {
        protocol = 'both';
      } else if (hasGraph) {
        protocol = 'graph';
      } else {
        protocol = 'imap';
      }

      const account: EmailAccount = {
        id: 'acc_' + Math.random().toString(36).substring(2, 9) + '_' + Date.now(),
        email,
        password: password || undefined,
        clientId: clientId || undefined,
        refreshToken: refreshToken || undefined,
        protocol,
        createdAt: Date.now(),
      };

      rows.push({
        email,
        password,
        clientId,
        refreshToken,
        rawLine: line,
        isValid: true,
      });

      accounts.push(account);
    }

    return { rows, accounts };
  },

  exportToLineFormat(accounts: EmailAccount[]): string {
    return accounts
      .map((acc) => {
        const parts = [
          acc.email || '',
          acc.password || '',
          acc.clientId || '',
          acc.refreshToken || '',
        ];
        // 只裁掉末尾的空字段：IMAP 账号不再导出多余的 ----，
        // Graph 账号仍需保留 password 的空位，导入时才能按列对齐。
        while (parts.length > 1 && parts[parts.length - 1] === '') {
          parts.pop();
        }
        return parts.join('----');
      })
      .join('\n');
  },
};
