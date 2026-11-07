export interface FolderInfo {
  name: string;
  displayName: string;
  totalCount: number;
}

export interface AttachmentInfo {
  filename: string;
  mimeType: string;
  size: number;
}

export interface EmailSummary {
  id: string;
  subject: string;
  from: string;
  to: string[];
  date: string;
  hasAttachments: boolean;
  snippet: string;
  protocol: string;
  accountEmail?: string;
}

export interface EmailDetail {
  id: string;
  subject: string;
  from: string;
  to: string[];
  cc: string[];
  date: string;
  bodyText?: string;
  bodyHtml?: string;
  attachments: AttachmentInfo[];
  protocol: string;
}
