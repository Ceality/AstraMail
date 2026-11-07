import axios from 'axios';
import type { EmailAccount, ProtocolType } from '../types/account';
import type { AccountStatus, ApiError, HealthStatus, MessageListResponse, MessageQueryResponse } from '../types/api';
import type { EmailDetail, FolderInfo } from '../types/email';

const API_BASE = '/api';

const apiClient = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

function toCredentialsPayload(account: EmailAccount) {
  const acc = account as Record<string, any>;
  const clientId = acc.clientId || acc.client_id || '';
  const refreshToken = acc.refreshToken || acc.refresh_token || '';
  const imapHost = acc.imapHost || acc.imap_host || '';
  const imapPort = acc.imapPort || acc.imap_port;

  return {
    email: acc.email || '',
    password: acc.password || '',
    clientId: clientId || undefined,
    refreshToken: refreshToken || undefined,
    client_id: clientId || undefined,
    refresh_token: refreshToken || undefined,
    protocol: acc.protocol || 'imap',
    imapHost: imapHost || undefined,
    imapPort: imapPort ? Number(imapPort) : undefined,
    imap_host: imapHost || undefined,
    imap_port: imapPort ? Number(imapPort) : undefined,
  };
}

function base64Encode(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function authHeaders(account: EmailAccount): Record<string, string> {
  return {
    Authorization: `AstraMail ${base64Encode(JSON.stringify(toCredentialsPayload(account)))}`,
  };
}

export function apiErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as ApiError | undefined;
    if (data?.message) {
      return data.message;
    }
    return error.message;
  }
  return error instanceof Error ? error.message : String(error);
}

export const api = {
  async healthCheck(): Promise<HealthStatus> {
    const res = await apiClient.get<HealthStatus>('/health');
    return res.data;
  },

  async getAccount(account: EmailAccount, checkProtocol?: ProtocolType): Promise<AccountStatus> {
    const res = await apiClient.get<AccountStatus>('/account', {
      params: { checkProtocol },
      headers: authHeaders(account),
    });
    return res.data;
  },

  async fetchFolders(account: EmailAccount, protocol?: string): Promise<FolderInfo[]> {
    const res = await apiClient.get<FolderInfo[]>('/folders', {
      params: { protocol },
      headers: authHeaders(account),
    });
    return res.data;
  },

  async fetchMailList(
    account: EmailAccount,
    folder: string = 'INBOX',
    page: number = 1,
    limit: number = 20,
    query?: string,
    protocol?: string
  ): Promise<MessageListResponse> {
    const res = await apiClient.get<MessageListResponse>('/messages', {
      params: { folder, page, limit, q: query, protocol },
      headers: authHeaders(account),
    });
    return res.data;
  },

  async fetchMailDetail(
    account: EmailAccount,
    folder: string,
    mailId: string,
    protocol?: string
  ): Promise<EmailDetail> {
    const res = await apiClient.get<EmailDetail>(`/messages/${encodeURIComponent(mailId)}`, {
      params: { folder, protocol },
      headers: authHeaders(account),
    });
    return res.data;
  },

  async queryMessages(
    accounts: EmailAccount[],
    limit: number = 20,
    query?: string,
    senderFilter?: string,
    protocol?: string
  ): Promise<MessageQueryResponse> {
    const res = await apiClient.post<MessageQueryResponse>('/messageQueries', {
      accounts: accounts.map(toCredentialsPayload),
      limit,
      query: query || undefined,
      senderFilter: senderFilter || undefined,
      protocol: protocol || undefined,
    });
    return res.data;
  },
};
