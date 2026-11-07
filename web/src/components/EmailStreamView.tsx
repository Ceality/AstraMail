import React, { useCallback, useState } from 'react';
import { motion } from 'framer-motion';
import type { EmailAccount } from '../types/account';
import type { EmailSummary } from '../types/email';
import { api, apiErrorMessage } from '../services/api';
import { resolveSelectedAccounts } from '../utils/selection';
import { detectVerificationCode } from '../utils/verification';
import { copyTextToClipboard } from '../utils/clipboard';
import { cn } from '../utils/cn';
import { useAutoFetch } from '../hooks/useAutoFetch';
import { useToast } from './ui/Toast';
import { AnimatedSwitch } from './ui/AnimatedSwitch';
import { MailDetailDrawer } from './MailDetailDrawer';
import {
  IconClock, IconCopy, IconInbox, IconKey, IconMail, IconMenu, IconPaperclip,
  IconRotate, IconSearch, IconSend, IconTimer, IconUser,
} from './ui/Icons';

interface EmailStreamViewProps {
  accounts: EmailAccount[];
  selectedAccounts: EmailAccount[];
  selectedAccountIds: string[];
  activeAccount: EmailAccount | null;
  isMobile?: boolean;
  onOpenMobileSidebar?: () => void;
}

function formatShortDate(dateStr: string): string {
  if (!dateStr) return '';
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const min = String(d.getMinutes()).padStart(2, '0');
    return `${mm}/${dd} ${hh}:${min}`;
  } catch {
    return dateStr;
  }
}

function senderName(from: string): string {
  const m = from.match(/^"?([^"<]+)"?\s*</);
  return (m ? m[1] : from.split('@')[0]).trim() || from;
}

export const EmailStreamView: React.FC<EmailStreamViewProps> = ({
  accounts,
  selectedAccounts,
  selectedAccountIds,
  activeAccount,
  isMobile = false,
  onOpenMobileSidebar,
}) => {
  const toast = useToast();

  const [drawerVisible, setDrawerVisible] = useState(false);
  const [activeMail, setActiveMail] = useState<EmailSummary | null>(null);
  const [activeMailAccount, setActiveMailAccount] = useState<EmailAccount | null>(null);

  const [searchQuery, setSearchQuery] = useState('');
  const [senderFilter, setSenderFilter] = useState('');
  const [fetchLimit, setFetchLimit] = useState<number>(20);

  const [imapActive, setImapActive] = useState(true);
  const [graphActive, setGraphActive] = useState(true);

  // 自动收件
  const [autoFetchEnabled, setAutoFetchEnabled] = useState(false);
  const [autoFetchInterval, setAutoFetchInterval] = useState(5);

  // 验证码模式
  const [verificationMode, setVerificationMode] = useState(false);

  /** 取件拿到的原始邮件，不受显示模式影响 */
  const [rawMails, setRawMails] = useState<EmailSummary[]>([]);
  const [loading, setLoading] = useState(false);

  // 验证码开关只影响标注，不改变列表内容
  const mails = rawMails;

  const getProtocolParam = (): string | undefined => {
    if (imapActive && graphActive) return 'both';
    if (imapActive) return 'imap';
    if (graphActive) return 'graph';
    return undefined;
  };

  const executeFetch = useCallback(
    async (targetAccounts: EmailAccount[], label: string, silent = false) => {
      setLoading(true);
      let loadingToastId: number | null = null;
      if (!silent) {
        loadingToastId = toast.info(
          `正在对 ${targetAccounts.length} 个账号执行${label}取件…`,
          60000
        );
      }

      try {
        const res = await api.queryMessages(
          targetAccounts,
          fetchLimit,
          searchQuery || undefined,
          senderFilter || undefined,
          getProtocolParam()
        );

        const items = res.items || [];
        setRawMails(items);

        // 验证码模式下，静默收件只有真的出现验证码邮件才提示
        const codeCount = items.filter(
          (m) => detectVerificationCode(m.subject, m.snippet).isVerification
        ).length;

        if (silent) {
          if (verificationMode && codeCount > 0) {
            toast.success(`收到 ${codeCount} 封验证码邮件`);
          }
          return;
        }

        if (res.errors && res.errors.length > 0) {
          toast.error(`部分账号异常: ${res.errors.join('；')}`);
        }
        if (items.length > 0) {
          toast.success(`取件成功，获取到 ${items.length} 封邮件`);
        } else if (!res.errors || res.errors.length === 0) {
          toast.warning('取件完成，未检索到新邮件');
        }
      } catch (err) {
        if (!silent) {
          toast.error(`取件失败: ${apiErrorMessage(err)}`);
        }
      } finally {
        if (loadingToastId !== null) {
          toast.dismiss(loadingToastId);
        }
        setLoading(false);
      }
    },
    [fetchLimit, searchQuery, senderFilter, imapActive, graphActive, verificationMode]
  );

  const handleFetchSelected = async () => {
    const selection = resolveSelectedAccounts(accounts, selectedAccountIds);
    if (selection.hasNoSelection) {
      if (selection.hasNoAccounts) {
        toast.warning('请先导入邮箱账号');
      } else {
        toast.warning('请先勾选要取件的账号，或点击「全部取件」');
      }
      return;
    }
    if (selection.accounts.length === 0) {
      toast.warning('勾选的账号已不存在，请重新勾选');
      return;
    }
    if (!imapActive && !graphActive) {
      toast.warning('请至少开启 IMAP 或 Graph 协议');
      return;
    }
    await executeFetch(selection.accounts, '自动化');
  };

  const handleFetchAll = async () => {
    if (accounts.length === 0) {
      toast.warning('尚未导入任何账号');
      return;
    }
    if (!imapActive && !graphActive) {
      toast.warning('请至少开启 IMAP 或 Graph 协议');
      return;
    }
    await executeFetch(accounts, '全量');
  };

  /**
   * 自动收件：优先用已勾选的账号，没勾选就用全部账号。
   * 静默执行，不打断用户当前操作。
   */
  const runAutoFetch = useCallback(async () => {
    if (accounts.length === 0) return;
    if (!imapActive && !graphActive) return;

    const selection = resolveSelectedAccounts(accounts, selectedAccountIds);
    const targets = selection.accounts.length > 0 ? selection.accounts : accounts;
    await executeFetch(targets, '自动', true);
  }, [accounts, selectedAccountIds, imapActive, graphActive, executeFetch]);

  useAutoFetch({
    enabled: autoFetchEnabled,
    intervalSeconds: autoFetchInterval,
    onTick: runAutoFetch,
  });

  const findAccountForMail = (mail: EmailSummary): EmailAccount | undefined => {
    if (mail.accountEmail) {
      return accounts.find((a) => a.email.toLowerCase() === mail.accountEmail!.toLowerCase());
    }
    return activeAccount || accounts[0];
  };

  return (
    <div className="flex h-full flex-1 flex-col overflow-hidden">
      {/* ===== 顶部工具栏 ===== */}
      <div className="shrink-0 bg-white px-5 py-3.5 sm:px-6">
        {isMobile ? (
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <button
                onClick={onOpenMobileSidebar}
                className="flex cursor-pointer items-center gap-1.5 rounded-xl border border-cold-200 bg-white px-3 py-1.5 text-[12.5px] font-medium text-ink-700 transition-colors hover:bg-cold-50"
              >
                <IconMenu size={13} />
                账号
                <span className="rounded-md bg-cold-100 px-1.5 py-0.5 text-[10.5px] font-semibold text-ink-600">
                  {selectedAccounts.length}/{accounts.length}
                </span>
              </button>
              <ProtocolToggle
                imapActive={imapActive}
                graphActive={graphActive}
                onImapChange={setImapActive}
                onGraphChange={setGraphActive}
              />
            </div>

            <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
              <AnimatedSwitch
                checked={verificationMode}
                onCheckedChange={setVerificationMode}
                label="验证码"
              />
              <AutoFetchControl
                enabled={autoFetchEnabled}
                onEnabledChange={setAutoFetchEnabled}
                interval={autoFetchInterval}
                onIntervalChange={setAutoFetchInterval}
                loading={loading}
                onManualTrigger={runAutoFetch}
              />
            </div>

            <div className="flex gap-2">
              <SearchInput
                value={searchQuery}
                onChange={setSearchQuery}
                onEnter={handleFetchSelected}
                placeholder="搜索主题 / 正文…"
                icon={<IconSearch size={12.5} />}
                className="flex-1"
              />
              <SearchInput
                value={senderFilter}
                onChange={setSenderFilter}
                onEnter={handleFetchSelected}
                placeholder="发件人"
                icon={<IconUser size={12.5} />}
                className="w-28"
              />
            </div>

            <LimitControl value={fetchLimit} onChange={setFetchLimit} />

            <div className="flex gap-2">
              <FetchButton primary loading={loading} onClick={handleFetchSelected} icon={<IconSend size={12.5} strokeWidth={2.2} />} label="选中取件" className="flex-1" />
              <FetchButton loading={loading} onClick={handleFetchAll} icon={<IconMail size={12.5} strokeWidth={2.2} />} label="全部取件" className="flex-1" />
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-2.5">
            <div className="flex items-center gap-2.5">
              <SearchInput
                value={searchQuery}
                onChange={setSearchQuery}
                onEnter={handleFetchSelected}
                placeholder="搜索主题 / 正文…"
                icon={<IconSearch size={12.5} />}
                className="flex-[2]"
              />
              <SearchInput
                value={senderFilter}
                onChange={setSenderFilter}
                onEnter={handleFetchSelected}
                placeholder="发件人过滤"
                icon={<IconUser size={12.5} />}
                className="flex-[1.1]"
              />
              <LimitControl value={fetchLimit} onChange={setFetchLimit} />
            </div>

            <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
              <div className="flex items-center gap-3">
                <ProtocolToggle
                  imapActive={imapActive}
                  graphActive={graphActive}
                  onImapChange={setImapActive}
                  onGraphChange={setGraphActive}
                />
                <AnimatedSwitch
                  checked={verificationMode}
                  onCheckedChange={setVerificationMode}
                  label="验证码"
                />
                <AutoFetchControl
                  enabled={autoFetchEnabled}
                  onEnabledChange={setAutoFetchEnabled}
                  interval={autoFetchInterval}
                  onIntervalChange={setAutoFetchInterval}
                  loading={loading}
                  onManualTrigger={runAutoFetch}
                />
              </div>
              <div className="flex items-center gap-2">
                <FetchButton primary loading={loading} onClick={handleFetchSelected} icon={<IconSend size={12.5} strokeWidth={2.2} />} label="选中取件" />
                <FetchButton loading={loading} onClick={handleFetchAll} icon={<IconMail size={12.5} strokeWidth={2.2} />} label="全部取件" />
              </div>
            </div>
          </div>
        )}
      </div>

      {/* ===== 邮件流列表 ===== */}
      <div className="scroll-slim flex-1 overflow-y-auto px-4 py-3 sm:px-6">
        {/* 三种状态互斥渲染：避免骨架屏与邮件列表同时出现在 DOM 里造成重叠 */}
        {loading ? (
          <LoadingState />
        ) : mails.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="flex flex-col gap-1.5">
              {mails.map((mail, idx) => {
                const matchedAccount = findAccountForMail(mail);
                const name = senderName(mail.from);

                // 验证码模式：检测该邮件是否含验证码
                const detected = verificationMode
                  ? detectVerificationCode(mail.subject, mail.snippet)
                  : null;
                // 只有真正提取到码值才算命中，避免空结果也套上验证码样式
                const match = detected?.code ? detected : null;

                return (
                  <div
                    key={`${mail.id}_${idx}`}
                    onClick={() => {
                      if (matchedAccount) {
                        setActiveMail(mail);
                        setActiveMailAccount(matchedAccount);
                        setDrawerVisible(true);
                      }
                    }}
                    className={[
                      'group rounded-2xl border bg-white p-3.5 transition-colors duration-150',
                      'cursor-pointer border-cold-200 hover:border-cold-300',
                    ].join(' ')}
                  >
                    <div className="flex gap-3">
                      {/* 发件人头像 */}
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-ink-100 text-[11.5px] font-bold text-ink-600">
                        {name.charAt(0).toUpperCase()}
                      </div>

                      <div className="min-w-0 flex-1">
                        {/* 主题 + 验证码标签 + 时间 */}
                        <div className="flex h-[22px] items-center justify-between gap-3">
                          <h3 className="min-w-0 flex-1 truncate text-[13.5px] font-semibold tracking-tight text-ink-900">
                            {mail.subject || '(无主题)'}
                          </h3>

                          <div className="flex shrink-0 items-center gap-2">
                            {/* 验证码标签 + 复制图标：固定行高，避免开关时撑高卡片 */}
                            {match?.code && (
                              <div className="flex h-[22px] items-center gap-1.5">
                                <span className="flex h-[22px] items-center rounded-[0.75rem] bg-emerald-600 px-2.5 font-mono text-[12px] font-bold leading-none tracking-wider text-white">
                                  {match.code}
                                </span>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    copyTextToClipboard(match.code!, `已复制验证码: ${match.code}`);
                                  }}
                                  className="flex h-[22px] w-[22px] cursor-pointer items-center justify-center rounded-[0.75rem] text-ink-400 transition-colors hover:bg-cold-100 hover:text-ink-700"
                                  title="复制验证码"
                                >
                                  <IconCopy size={14} strokeWidth={2.2} />
                                </button>
                              </div>
                            )}

                            <span className="flex h-[22px] items-center gap-1 text-[12px] font-medium leading-none tabular-nums text-ink-400">
                              <IconClock size={11} />
                              {formatShortDate(mail.date)}
                            </span>
                          </div>
                        </div>

                        {/* 发件人 + 徽标 */}
                        <div className="mt-1 flex flex-wrap items-center gap-1.5">
                          <span className="max-w-[55%] truncate text-[12px] font-medium text-ink-500">
                            {mail.from}
                          </span>
                          {mail.accountEmail && (
                            <span className="rounded-md bg-cold-100 px-1.5 py-px text-[10px] font-medium text-ink-500">
                              {mail.accountEmail}
                            </span>
                          )}
                          <span
                            className={`rounded-md px-1.5 py-px text-[10px] font-semibold uppercase tracking-wider ${
                              mail.protocol === 'graph' ? 'badge-graph' : 'badge-imap'
                            }`}
                          >
                            {mail.protocol}
                          </span>
                          {mail.hasAttachments && (
                            <span className="flex items-center gap-0.5 rounded-md bg-amber-50 px-1.5 py-px text-[10px] font-medium text-amber-600">
                              <IconPaperclip size={9} strokeWidth={2.2} />
                              附件
                            </span>
                          )}
                        </div>

                        <p className="line-clamp-2 mt-1.5 text-[12.5px] leading-relaxed text-ink-500">
                          {mail.snippet || '(正文无摘要内容)'}
                        </p>
                      </div>
                    </div>
                  </div>
                );
              })}
          </div>
        )}
      </div>

      {/* ===== 邮件详情抽屉 ===== */}
      <MailDetailDrawer
        open={drawerVisible}
        mailSummary={activeMail}
        account={activeMailAccount}
        onClose={() => setDrawerVisible(false)}
      />
    </div>
  );
};

/* ================= 子组件 ================= */

/**
 * 自动收件内联控件。
 * 关闭时只占一个开关；开启后间隔输入与立即执行以展开动画出现。
 */
function AutoFetchControl({
  enabled,
  onEnabledChange,
  interval,
  onIntervalChange,
  loading,
  onManualTrigger,
}: {
  enabled: boolean;
  onEnabledChange: (v: boolean) => void;
  interval: number;
  onIntervalChange: (v: number) => void;
  loading: boolean;
  onManualTrigger: () => void;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <div className="flex shrink-0 items-center gap-2 select-none">
        <IconTimer size={13} className="text-ink-400" />
        <span className="text-[12px] font-medium text-ink-600">自动收件</span>
        <AnimatedSwitch
          checked={enabled}
          onCheckedChange={onEnabledChange}
          ariaLabel="自动收件"
        />
      </div>

      {/* 0fr → 1fr 的网格展开：只影响这一小段宽度，避免整块界面跳变 */}
      <div
        aria-hidden={!enabled}
        inert={!enabled}
        className={cn(
          'grid transition-[grid-template-columns,opacity] duration-300 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]',
          enabled ? 'grid-cols-[1fr] opacity-100' : 'grid-cols-[0fr] opacity-0'
        )}
      >
        <div className="min-w-0 overflow-hidden">
          <div
            className={cn(
              'flex items-center gap-1.5 pl-1 transition-transform duration-300 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]',
              enabled ? 'translate-x-0' : '-translate-x-1'
            )}
          >
            <span className="text-[11.5px] font-medium text-ink-400">每</span>
            <input
              type="number"
              min={1}
              max={3600}
              value={interval}
              onChange={(e) =>
                onIntervalChange(Math.min(3600, Math.max(1, Number(e.target.value) || 5)))
              }
              aria-label="自动收件间隔（秒）"
              className="w-12 rounded-lg bg-cold-50 px-1.5 py-1 text-center text-[11.5px] font-semibold tabular-nums text-ink-700 outline-none transition-colors duration-150 focus:bg-white focus:ring-1 focus:ring-cold-300 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
            />
            <span className="text-[11.5px] font-medium text-ink-400">秒</span>
            <button
              onClick={onManualTrigger}
              disabled={loading}
              title="立即执行一次"
              aria-label="立即执行一次"
              className="ml-0.5 flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-lg text-ink-400 transition-colors duration-150 hover:bg-cold-100 hover:text-ink-700 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {loading ? (
                <span
                  className="h-3 w-3 shrink-0 animate-spin rounded-full border-[1.5px] border-current border-t-transparent"
                  aria-hidden
                />
              ) : (
                <IconRotate size={12.5} />
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function SearchInput({
  value, onChange, onEnter, placeholder, icon, className,
}: {
  value: string; onChange: (v: string) => void; onEnter?: () => void;
  placeholder: string; icon: React.ReactNode; className?: string;
}) {
  return (
    <div className={`group relative ${className ?? ''}`}>
      <span className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-400">
        {icon}
      </span>
      <input
        type="text"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && onEnter?.()}
        className="w-full rounded-xl border border-cold-200 bg-white py-2 pl-8 pr-7 text-[12.5px] text-ink-800 placeholder-ink-400 outline-none transition-colors duration-150 focus:border-cold-400"
      />
      {value && (
        <button
          onClick={() => onChange('')}
          className="absolute right-2 top-1/2 flex -translate-y-1/2 cursor-pointer items-center justify-center rounded-md text-ink-400 transition-colors hover:text-ink-700"
          aria-label="清除"
          style={{ width: 16, height: 16 }}
        >
          <svg width="7.5" height="7.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.2" strokeLinecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>
        </button>
      )}
    </div>
  );
}

function ProtocolToggle({
  imapActive, graphActive, onImapChange, onGraphChange,
}: {
  imapActive: boolean; graphActive: boolean;
  onImapChange: (v: boolean) => void; onGraphChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-center gap-0.5 rounded-xl border border-cold-200 bg-cold-50 p-0.5">
      {([
        { label: 'IMAP', active: imapActive, toggle: onImapChange },
        { label: 'Graph', active: graphActive, toggle: onGraphChange },
      ] as const).map((p) => (
        <button
          key={p.label}
          onClick={() => p.toggle(!p.active)}
          className={`relative cursor-pointer rounded-lg px-2.5 py-1 text-[11.5px] font-semibold tracking-wide outline-none transition-colors duration-200 ${
            p.active ? 'text-white' : 'text-ink-400 hover:text-ink-600'
          }`}
        >
          {/* 纯 CSS 指示器：scale/opacity 只走合成层，无 JS 动画开销 */}
          <span
            className={cn(
              'pointer-events-none absolute inset-0 rounded-lg bg-ink-900',
              'transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]',
              p.active
                ? 'scale-100 opacity-100'
                : 'scale-90 opacity-0'
            )}
          />
          <span className="relative z-10">{p.label}</span>
        </button>
      ))}
    </div>
  );
}

function LimitControl({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-[11.5px] font-medium text-ink-400">取件数</span>
      <div className="flex items-center rounded-xl border border-cold-200 bg-white">
        <button
          onClick={() => onChange(Math.max(1, value - 5))}
          className="cursor-pointer px-1.5 py-1 text-ink-400 transition-colors hover:text-ink-700"
        >
          −
        </button>
        <input
          type="number"
          min={1}
          max={100}
          value={value}
          onChange={(e) => onChange(Math.min(100, Math.max(1, Number(e.target.value) || 20)))}
          className="w-9 border-x border-cold-200 bg-transparent py-1 text-center text-[11.5px] font-semibold tabular-nums text-ink-800 outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
        />
        <button
          onClick={() => onChange(Math.min(100, value + 5))}
          className="cursor-pointer px-1.5 py-1 text-ink-400 transition-colors hover:text-ink-700"
        >
          +
        </button>
      </div>
    </div>
  );
}

function FetchButton({
  primary, loading, onClick, icon, label, className,
}: {
  primary?: boolean; loading: boolean; onClick: () => void;
  icon: React.ReactNode; label: string; className?: string;
}) {
  return (
    <button
      disabled={loading}
      onClick={onClick}
      className={`flex cursor-pointer items-center justify-center gap-1.5 rounded-xl px-4 py-2 text-[12.5px] font-medium outline-none transition-colors duration-150 disabled:cursor-not-allowed ${
        primary
          ? 'bg-ink-900 text-white hover:bg-ink-700 disabled:opacity-55'
          : 'border border-cold-200 bg-white text-ink-700 hover:bg-cold-50 hover:border-cold-300 disabled:opacity-45'
      } ${className ?? ''}`}
    >
      {/* 加载时换成专用 spinner，不要让业务图标自己转圈 */}
      {loading ? (
        <span
          className="h-[12.5px] w-[12.5px] shrink-0 animate-spin rounded-full border-[1.5px] border-current border-t-transparent opacity-80"
          aria-hidden
        />
      ) : (
        <span className="shrink-0">{icon}</span>
      )}
      {label}
    </button>
  );
}

function LoadingState() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.15 }}
      className="flex flex-col gap-1.5 pt-0.5"
    >
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex gap-3 rounded-2xl border border-cold-200 bg-white p-3.5">
          <div className="shimmer h-8 w-8 shrink-0 rounded-lg" />
          <div className="flex-1 space-y-2 pt-0.5">
            <div className="shimmer h-3 w-3/5 rounded-md" />
            <div className="shimmer h-2.5 w-2/5 rounded-md" />
            <div className="shimmer h-2.5 w-4/5 rounded-md" />
          </div>
        </div>
      ))}
      <div className="mt-4 flex items-center justify-center gap-2 text-[12px] font-medium text-ink-400">
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
          className="rounded-full border-2 border-cold-200 border-t-ink-500"
          style={{ width: 14, height: 14 }}
        />
        正在并行拉取邮件…
      </div>
    </motion.div>
  );
}

function EmptyState() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
      className="flex flex-col items-center gap-4 pb-10 pt-16 text-center sm:pt-24"
    >
      <IconInbox size={36} strokeWidth={1.4} className="text-ink-300" />

      <p className="text-[14px] font-semibold text-ink-700">暂无邮件数据</p>
    </motion.div>
  );
}
