import React, { useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import type { EmailAccount } from '../types/account';
import { storage } from '../services/storage';
import { copyTextToClipboard } from '../utils/clipboard';
import { resolveSelectedAccounts, selectAllState } from '../utils/selection';
import { AnimatedCheckbox } from './ui/AnimatedCheckbox';
import { AnimatedDialog } from './ui/AnimatedDialog';
import { useToast } from './ui/Toast';
import {
  IconCopy, IconDownload, IconInbox, IconSearch,
  IconTrash, IconUpload,
} from './ui/Icons';

interface AccountSidebarProps {
  accounts: EmailAccount[];
  selectedAccountIds: string[];
  activeAccountId: string | null;
  onToggleSelectAccount: (id: string) => void;
  onSelectAllAccounts: (checked: boolean) => void;
  onSetActiveAccount: (acc: EmailAccount) => void;
  onOpenImportModal: () => void;
  onDeleteAccount: (id: string) => void;
  onClearAccounts: () => void;
}

/**
 * 激活账号的滑动指示条。
 *
 * 用一次性的 offsetTop 测量 + CSS transform 位移，而不是 Framer Motion 的
 * layoutId：后者会在每次渲染时对列表内所有元素做 getBoundingClientRect，
 * 账号一多就明显掉帧。这里只在激活项/列表真正变化时测一次。
 */
const ActiveIndicator: React.FC<{
  activeAccountId: string | null;
  itemCount: number;
  containerRef: React.RefObject<HTMLDivElement | null>;
}> = ({ activeAccountId, itemCount, containerRef }) => {
  const barRef = useRef<HTMLSpanElement>(null);
  const [top, setTop] = useState<number>(-1);
  const [height, setHeight] = useState<number>(0);

  const measure = useCallback(() => {
    const container = containerRef.current;
    const bar = barRef.current;
    if (!container || !bar) return;
    const rows = container.querySelectorAll<HTMLElement>('[data-account-row]');
    for (const row of rows) {
      if (row.dataset.accountId === activeAccountId) {
        // 用 rect 相减得到容器内偏移，避免 offsetParent 层级不同带来的误差
        const cRect = container.getBoundingClientRect();
        const rRect = row.getBoundingClientRect();
        setTop(rRect.top - cRect.top);
        setHeight(row.offsetHeight);
        return;
      }
    }
    setTop(-1);
  }, [activeAccountId, containerRef]);

  useLayoutEffect(() => {
    measure();
  }, [measure, itemCount]);

  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container || typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(() => measure());
    ro.observe(container);
    return () => ro.disconnect();
  }, [measure]);

  if (top < 0) return null;

  return (
    <span
      ref={barRef}
      className="pointer-events-none absolute left-0 w-[2.5px] rounded-full bg-ink-900"
      style={{
        top: 0,
        // 列表行上下各 6px 内边距，指示条略短于行高以对齐内容区
        height: Math.max(0, height - 12),
        transform: `translateY(${top + 6}px)`,
        transition: 'transform 240ms cubic-bezier(0.25,0.46,0.45,0.94), height 240ms cubic-bezier(0.25,0.46,0.45,0.94)',
      }}
    />
  );
};

/**
 * 单个账号行。
 *
 * 用 React.memo 隔离：勾选其中一个账号时，其它行不会重新渲染，
 * 配合纯 CSS 过渡可让长列表下的连续勾选保持流畅。
 */
const AccountRow = React.memo(function AccountRow({
  account,
  isChecked,
  isActive,
  onToggle,
  onActivate,
  onRequestDelete,
}: {
  account: EmailAccount;
  isChecked: boolean;
  isActive: boolean;
  onToggle: (id: string) => void;
  onActivate: (acc: EmailAccount) => void;
  onRequestDelete: (acc: EmailAccount) => void;
}) {
  const hasStatus = account.imapStatus || account.graphStatus;
  const ok =
    (account.imapStatus?.success ?? false) ||
    (account.graphStatus?.success ?? false);

  return (
    <motion.div
      data-account-row
      data-account-id={account.id}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0, transition: { duration: 0.12 } }}
      onClick={() => onActivate(account)}
      className={[
        'group relative flex cursor-pointer items-center gap-2 rounded-xl px-2 py-1.5 transition-colors duration-200',
        isActive
          ? 'bg-cold-50'
          : 'hover:bg-cold-50/70',
      ].join(' ')}
    >
      <div onClick={(e) => e.stopPropagation()} className="shrink-0">
        <AnimatedCheckbox
          checked={isChecked}
          onCheckedChange={() => onToggle(account.id)}
        />
      </div>

      {/* 邮箱地址 + 协议标记：地址优先占满宽度 */}
      <div className="min-w-0 flex-1">
        <div
          className={`truncate text-[12.5px] leading-5 ${
            isChecked ? 'font-medium text-ink-900' : 'font-medium text-ink-700'
          }`}
        >
          {account.email}
        </div>
        <div className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wider text-ink-400">
          {account.protocol === 'both' ? (
            <>
              <span className="h-1 w-1 rounded-full bg-ink-400" />
              <span className="h-1 w-1 rounded-full bg-ink-400" />
            </>
          ) : (
            account.protocol
          )}
        </div>
      </div>

      {hasStatus && (
        <span
          className={`h-1.5 w-1.5 shrink-0 rounded-full ${ok ? 'bg-emerald-500' : 'bg-red-500'}`}
        />
      )}

      {/* 行内操作：常驻显示，hover 时加深，避免隐藏导致不易发现 */}
      <div
        className="flex shrink-0 items-center gap-0.5"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={() => copyTextToClipboard(account.email, `已复制: ${account.email}`)}
          className="flex cursor-pointer items-center justify-center rounded-md p-1.5 text-ink-400 opacity-70 transition-all duration-150 hover:bg-cold-200 hover:text-ink-700 hover:opacity-100"
          title="复制账号"
        >
          <IconCopy size={14} strokeWidth={2} />
        </button>
        <button
          onClick={() => onRequestDelete(account)}
          className="flex cursor-pointer items-center justify-center rounded-md p-1.5 text-ink-400 opacity-70 transition-all duration-150 hover:bg-red-50 hover:text-red-600 hover:opacity-100"
          title="删除账号"
        >
          <IconTrash size={14} strokeWidth={2} />
        </button>
      </div>
    </motion.div>
  );
});

export const AccountSidebar: React.FC<AccountSidebarProps> = ({
  accounts,
  selectedAccountIds,
  activeAccountId,
  onToggleSelectAccount,
  onSelectAllAccounts,
  onSetActiveAccount,
  onOpenImportModal,
  onDeleteAccount,
  onClearAccounts,
}) => {
  const toast = useToast();
  const [searchKeyword, setSearchKeyword] = useState('');
  const [clearConfirmOpen, setClearConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<EmailAccount | null>(null);
  const listContainerRef = useRef<HTMLDivElement>(null);

  const filteredAccounts = useMemo(
    () =>
      accounts.filter((acc) =>
        acc.email.toLowerCase().includes(searchKeyword.toLowerCase().trim())
      ),
    [accounts, searchKeyword]
  );

  // 用 Set 做 O(1) 查找：勾选状态在长列表下每次渲染都做 Array.includes 会拖慢切换
  const selectedIdSet = useMemo(
    () => new Set(selectedAccountIds),
    [selectedAccountIds]
  );

  const { checked: isAllSelected, indeterminate: isIndeterminate } = selectAllState(
    accounts,
    selectedAccountIds
  );

  const downloadTextFile = (content: string, filename: string) => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 100);
  };

  const handleExport = () => {
    const selection = resolveSelectedAccounts(accounts, selectedAccountIds);
    if (selection.hasNoSelection) {
      toast.warning('请先勾选要导出的账号');
      return;
    }
    const targets = selection.accounts;
    if (targets.length === 0) {
      toast.warning('没有可导出的账号');
      return;
    }
    const text = storage.exportToLineFormat(targets);
    try {
      const now = new Date();
      const dateStr = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}`;
      downloadTextFile(text, `astramail_accounts_${dateStr}.txt`);
      toast.success(`已导出 ${targets.length} 个账号`);
    } catch (err) {
      console.error('Download file error:', err);
    }
  };

  return (
    <div className="flex h-full w-full flex-col gap-2.5 p-3">
      {/* ===== 头部 ===== */}
      <div className="flex shrink-0 items-center justify-between pb-3">
          <div className="flex items-center gap-2.5">
            <img
              src="/favicon.svg"
              alt=""
              width={36}
              height={36}
              className="h-9 w-9 shrink-0 rounded-full object-cover"
            />
            <div className="text-[17px] font-semibold leading-none tracking-[-0.02em] text-ink-900">
              AstraMail
            </div>
          </div>

          <button
            onClick={onOpenImportModal}
            className="flex cursor-pointer items-center gap-1.5 rounded-xl bg-ink-900 px-3 py-1.5 text-[12.5px] font-medium text-white transition-colors duration-150 hover:bg-ink-700"
          >
            <IconUpload size={12.5} strokeWidth={2.4} />
            导入
          </button>
      </div>

      {/* ===== 搜索框 ===== */}
      <div className="group relative mb-3 shrink-0">
          <IconSearch
            size={13}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-400"
          />
          <input
            type="text"
            placeholder="搜索账号…"
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            className="w-full rounded-xl border border-cold-200 bg-white py-2 pl-8.5 pr-8 text-[12.5px] text-ink-800 placeholder-ink-400 outline-none transition-colors duration-150 focus:border-cold-400"
          />
          {searchKeyword && (
            <button
              onClick={() => setSearchKeyword('')}
              className="absolute right-2.5 top-1/2 flex h-4.5 w-4.5 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md text-ink-400 transition-colors hover:text-ink-700"
              aria-label="清除搜索"
              style={{ width: 18, height: 18 }}
            >
              <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>
            </button>
          )}
      </div>

      {/* ===== 全选与批量操作 ===== */}
      <div className="shrink-0 rounded-3xl border border-cold-200 bg-white p-3 shadow-soft">
          <div className="mb-2 flex items-center justify-between">
            <AnimatedCheckbox
              checked={isAllSelected}
              indeterminate={isIndeterminate}
              onCheckedChange={onSelectAllAccounts}
              label="全选"
            />
            <span className="text-[11px] font-medium tabular-nums text-ink-400">
              {selectedAccountIds.length} / {accounts.length}
            </span>
          </div>
          <div className="flex gap-1.5">
            <button
              disabled={accounts.length === 0}
              onClick={handleExport}
              className="flex flex-1 cursor-pointer items-center justify-center gap-1 rounded-xl border border-cold-200 bg-cold-50 py-1.5 text-[11.5px] font-medium text-ink-600 transition-colors duration-150 hover:border-cold-300 hover:bg-cold-100 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <IconDownload size={11.5} strokeWidth={2.2} />
              导出
            </button>
            <button
              disabled={accounts.length === 0}
              onClick={() => setClearConfirmOpen(true)}
              className="flex flex-1 cursor-pointer items-center justify-center gap-1 rounded-xl border border-cold-200 bg-cold-50 py-1.5 text-[11.5px] font-medium text-ink-600 transition-colors duration-150 hover:border-red-200 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <IconTrash size={11.5} strokeWidth={2.2} />
              清空
            </button>
          </div>
      </div>

      {/* ===== 账号列表卡片 ===== */}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-3xl border border-cold-200 bg-white shadow-soft">
        <div className="scroll-slim flex-1 overflow-y-auto p-3">
        {filteredAccounts.length === 0 ? (
          <div className="flex flex-col items-center gap-2.5 py-14 text-center">
            <IconInbox size={28} strokeWidth={1.5} className="text-ink-300" />
            <p className="text-[12.5px] font-medium text-ink-400">
              {searchKeyword ? '暂无匹配账号' : '暂无账号，点击「导入」开始'}
            </p>
          </div>
        ) : (
          <div ref={listContainerRef} className="relative flex flex-col gap-0.5">
            <ActiveIndicator
              activeAccountId={activeAccountId}
              itemCount={filteredAccounts.length}
              containerRef={listContainerRef}
            />
            <AnimatePresence initial={false}>
              {filteredAccounts.map((account) => (
                <AccountRow
                  key={account.id}
                  account={account}
                  isChecked={selectedIdSet.has(account.id)}
                  isActive={activeAccountId === account.id}
                  onToggle={onToggleSelectAccount}
                  onActivate={onSetActiveAccount}
                  onRequestDelete={setDeleteTarget}
                />
              ))}
            </AnimatePresence>
          </div>
        )}
        </div>
      </div>

      {/* ===== 删除确认 ===== */}
      <AnimatedDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="删除账号"
        width="max-w-sm"
        footer={
          <>
            <button
              onClick={() => setDeleteTarget(null)}
              className="cursor-pointer rounded-xl border border-cold-200 bg-white px-3.5 py-1.5 text-[12.5px] font-medium text-ink-600 transition-colors hover:bg-cold-50"
            >
              取消
            </button>
            <button
              onClick={() => {
                if (deleteTarget) onDeleteAccount(deleteTarget.id);
                setDeleteTarget(null);
              }}
              className="cursor-pointer rounded-xl bg-red-600 px-3.5 py-1.5 text-[12.5px] font-medium text-white transition-colors hover:bg-red-700"
            >
              确认删除
            </button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-ink-600">
          确定删除账号 <span className="font-semibold text-ink-900">{deleteTarget?.email}</span> 吗？
          此操作将移除该账号的全部本地凭据。
        </p>
      </AnimatedDialog>

      {/* ===== 清空确认 ===== */}
      <AnimatedDialog
        open={clearConfirmOpen}
        onOpenChange={setClearConfirmOpen}
        title="清空全部账号"
        width="max-w-sm"
        footer={
          <>
            <button
              onClick={() => setClearConfirmOpen(false)}
              className="cursor-pointer rounded-xl border border-cold-200 bg-white px-3.5 py-1.5 text-[12.5px] font-medium text-ink-600 transition-colors hover:bg-cold-50"
            >
              取消
            </button>
            <button
              onClick={() => {
                onClearAccounts();
                setClearConfirmOpen(false);
              }}
              className="cursor-pointer rounded-xl bg-red-600 px-3.5 py-1.5 text-[12.5px] font-medium text-white transition-colors hover:bg-red-700"
            >
              全部清空
            </button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-ink-600">
          此操作将移除全部 <span className="font-semibold text-ink-900">{accounts.length}</span> 个账号及其本地凭据，
          且不可恢复。建议先执行「导出」备份。
        </p>
      </AnimatedDialog>
    </div>
  );
};
