import React, { useEffect, useMemo, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import type { EmailAccount } from '../types/account';
import type { EmailDetail, EmailSummary } from '../types/email';
import { api, apiErrorMessage } from '../services/api';
import { AnimatedDrawer } from './ui/AnimatedDrawer';
import { useToast } from './ui/Toast';
import {
  IconAlignLeft, IconCode, IconInbox, IconPaperclip,
} from './ui/Icons';

/* ============ HTML 邮件安全渲染 ============ */
const EmailHtmlViewer: React.FC<{ html: string }> = ({ html }) => {
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const safeContent = useMemo(() => {
    if (!html) return '';
    const scrollbarCss = `
      ::-webkit-scrollbar { width: 6px; height: 6px; }
      ::-webkit-scrollbar-thumb { background-color: #d3d9de; border-radius: 3px; }
      ::-webkit-scrollbar-thumb:hover { background-color: #9aa5ad; }
      ::-webkit-scrollbar-track { background-color: transparent; }
      body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; }
    `;
    if (!html.includes('<base')) {
      return `<base target="_blank">\n<style>${scrollbarCss}</style>\n${html}`;
    }
    return `<style>${scrollbarCss}</style>\n${html}`;
  }, [html]);

  const handleIframeLoad = () => {
    try {
      const iframe = iframeRef.current;
      if (!iframe) return;
      const doc = iframe.contentDocument || iframe.contentWindow?.document;
      if (doc?.body) {
        const height = Math.max(doc.body.scrollHeight, doc.documentElement.scrollHeight);
        if (height > 200) {
          iframe.style.height = `${Math.min(height + 20, 800)}px`;
        }
      }
    } catch {
      // ignore
    }
  };

  return (
    <iframe
      ref={iframeRef}
      srcDoc={safeContent}
      title="Email HTML"
      onLoad={handleIframeLoad}
      sandbox="allow-popups allow-popups-to-escape-sandbox allow-same-origin"
      style={{
        width: '100%',
        minHeight: 420,
        height: 600,
        border: 'none',
        display: 'block',
        backgroundColor: '#FFFFFF',
        borderRadius: 8,
      }}
    />
  );
};

/* ============ 详情抽屉 ============ */
interface MailDetailDrawerProps {
  open: boolean;
  mailSummary: EmailSummary | null;
  account: EmailAccount | null;
  onClose: () => void;
}

export const MailDetailDrawer: React.FC<MailDetailDrawerProps> = ({
  open,
  mailSummary,
  account,
  onClose,
}) => {
  const toast = useToast();
  const [detail, setDetail] = useState<EmailDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [viewFormat, setViewFormat] = useState<'html' | 'text'>('html');

  useEffect(() => {
    let cancelled = false;
    if (!open || !mailSummary || !account) return;

    queueMicrotask(() => { if (!cancelled) setLoading(true); });
    api.fetchMailDetail(account, 'INBOX', mailSummary.id, mailSummary.protocol)
      .then((data) => { if (!cancelled) setDetail(data); })
      .catch((err) => { if (!cancelled) toast.error(`获取邮件详情失败: ${apiErrorMessage(err)}`); })
      .finally(() => { if (!cancelled) setLoading(false); });

    return () => { cancelled = true; };
  }, [open, mailSummary, account]);

  const handleClose = () => {
    setDetail(null);
    onClose();
  };

  return (
    <AnimatedDrawer
      open={open}
      onOpenChange={(o) => { if (!o) handleClose(); }}
      side="right"
      widthClass="w-full sm:w-[720px]"
      title={mailSummary?.subject || '邮件详情'}
    >
      <AnimatePresence initial={false}>
        {loading ? (
          <motion.div
            key="detail-loading"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="flex flex-col gap-4 p-5"
          >
            <div className="space-y-2.5">
              {[0.55, 0.8, 0.45, 0.65].map((w, i) => (
                <div key={i} className="shimmer h-3 rounded-md" style={{ width: `${w * 100}%` }} />
              ))}
            </div>
            <div className="shimmer h-56 rounded-2xl" />
            <div className="flex items-center justify-center gap-2 pt-2 text-[12px] font-medium text-ink-400">
              <motion.div
                animate={{ rotate: 360 }}
                transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                className="rounded-full border-2 border-cold-200 border-t-ink-500"
                style={{ width: 13, height: 13 }}
              />
              正在拉取邮件正文与附件…
            </div>
          </motion.div>
        ) : detail ? (
          <motion.div
            key="detail-content"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0, transition: { duration: 0.12 } }}
            transition={{ duration: 0.18 }}
            className="flex flex-col gap-4 p-5"
          >
            {/* 元信息 */}
            <div className="rounded-2xl border border-cold-200 bg-cold-50 p-3.5">
              <div className="flex flex-col gap-2.5">
                <MetaRow label="发件人" value={detail.from} highlight />
                <MetaRow label="收件人" value={detail.to.join(', ') || '-'} />
                {detail.cc.length > 0 && <MetaRow label="抄送" value={detail.cc.join(', ')} />}
                <MetaRow label="日期" value={detail.date} />
                <div className="flex items-center gap-2">
                  <span className="w-12 shrink-0 text-[11px] font-semibold uppercase tracking-wider text-ink-400">
                    协议
                  </span>
                  <span
                    className={`rounded-md px-1.5 py-px text-[10px] font-semibold uppercase tracking-wider ${
                      detail.protocol === 'graph' ? 'badge-graph' : 'badge-imap'
                    }`}
                  >
                    {detail.protocol}
                  </span>
                </div>
              </div>
            </div>

            {/* 附件 */}
            {detail.attachments.length > 0 && (
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-1.5 text-[12.5px] font-semibold text-ink-800">
                  <IconPaperclip size={12.5} className="text-ink-400" />
                  附件
                  <span className="rounded-md bg-cold-100 px-1.5 py-px text-[10px] font-semibold text-ink-600">
                    {detail.attachments.length}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {detail.attachments.map((att, idx) => (
                    <span
                      key={idx}
                      className="flex items-center gap-1.5 rounded-xl border border-cold-200 bg-white px-2.5 py-1 text-[11.5px] font-medium text-ink-700"
                    >
                      <IconPaperclip size={10} strokeWidth={2.2} />
                      {att.filename}
                      <span className="text-[10px] text-ink-400">
                        {(att.size / 1024).toFixed(1)} KB
                      </span>
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* 正文头 + 视图切换 */}
            <div className="flex items-center justify-between">
              <span className="text-[12.5px] font-semibold text-ink-800">邮件正文</span>
              <div className="flex items-center gap-0.5 rounded-xl border border-cold-200 bg-cold-50 p-0.5">
                {([
                  { key: 'html' as const, icon: <IconCode size={12} />, disabled: !detail.bodyHtml, tip: 'HTML 视图' },
                  { key: 'text' as const, icon: <IconAlignLeft size={12} />, disabled: false, tip: '纯文本视图' },
                ]).map((v) => (
                  <button
                    key={v.key}
                    disabled={v.disabled}
                    onClick={() => setViewFormat(v.key)}
                    title={v.tip}
                    className={`relative cursor-pointer rounded-md p-1.5 outline-none transition-colors duration-200 disabled:cursor-not-allowed disabled:opacity-35 ${
                      viewFormat === v.key ? 'text-white' : 'text-ink-400 hover:text-ink-600'
                    }`}
                  >
                    {/* 纯 CSS 指示器：只动 opacity/transform，不触发布局 */}
                    <span
                      className={`pointer-events-none absolute inset-0 rounded-md bg-ink-900 transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.25,0.46,0.45,0.94)] ${
                        viewFormat === v.key ? 'scale-100 opacity-100' : 'scale-90 opacity-0'
                      }`}
                    />
                    <span className="relative z-10">{v.icon}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* 正文内容 */}
            <AnimatePresence mode="wait">
              {viewFormat === 'html' && detail.bodyHtml ? (
                <motion.div
                  key="html-view"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0, transition: { duration: 0.08 } }}
                  transition={{ duration: 0.12 }}
                  className="overflow-hidden rounded-2xl border border-cold-200 bg-white"
                >
                  <EmailHtmlViewer html={detail.bodyHtml} />
                </motion.div>
              ) : (
                <motion.div
                  key="text-view"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0, transition: { duration: 0.08 } }}
                  transition={{ duration: 0.12 }}
                  className="scroll-slim max-h-[620px] min-h-[300px] overflow-y-auto whitespace-pre-wrap rounded-2xl border border-cold-200 bg-white p-4 font-mono text-[12px] leading-relaxed text-ink-700"
                >
                  {detail.bodyText || detail.bodyHtml || '(正文无内容)'}
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        ) : (
          <motion.div
            key="detail-empty"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="flex flex-col items-center gap-2.5 py-20 text-center"
          >
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-cold-100 text-ink-400">
              <IconInbox size={20} strokeWidth={1.5} />
            </div>
            <p className="text-[12.5px] font-medium text-ink-400">未选定邮件</p>
          </motion.div>
        )}
      </AnimatePresence>
    </AnimatedDrawer>
  );
};

function MetaRow({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-start gap-2">
      <span className="w-12 shrink-0 pt-px text-[11px] font-semibold uppercase tracking-wider text-ink-400">
        {label}
      </span>
      <span className={`min-w-0 flex-1 break-all text-[12.5px] leading-relaxed ${highlight ? 'font-medium text-ink-900' : 'text-ink-600'}`}>
        {value}
      </span>
    </div>
  );
}
