export type ProtocolType = 'imap' | 'graph' | 'both';

export interface EmailAccount {
  id: string;
  email: string;
  password?: string;
  clientId?: string;
  refreshToken?: string;
  protocol: ProtocolType;
  imapHost?: string;
  imapPort?: number;
  createdAt: number;
  lastCheckedAt?: number;
  imapStatus?: {
    success: boolean;
    message: string;
    messageCount?: number;
  };
  graphStatus?: {
    success: boolean;
    message: string;
    messageCount?: number;
  };
}

export interface AccountImportRow {
  email: string;
  password?: string;
  clientId?: string;
  refreshToken?: string;
  rawLine: string;
  isValid: boolean;
  error?: string;
}
