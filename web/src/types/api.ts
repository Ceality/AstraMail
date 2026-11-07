import type { EmailSummary } from './email';

export interface ApiError {
  code: string;
  message: string;
}

export interface ProtocolStatus {
  success: boolean;
  message: string;
  folderCount: number | null;
  messageCount: number | null;
}

export interface AccountStatus {
  email: string;
  imapStatus: ProtocolStatus | null;
  graphStatus: ProtocolStatus | null;
}

export interface HealthStatus {
  status: string;
  service: string;
  version: string;
}

export interface MessageListResponse {
  items: EmailSummary[];
  total: number;
  page: number;
  limit: number;
}

export interface MessageQueryResponse {
  items: EmailSummary[];
  total: number;
  errors?: string[];
}
