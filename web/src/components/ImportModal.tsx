import React, { useMemo, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import type { EmailAccount } from '../types/account';
import { storage } from '../services/storage';
import { AnimatedDialog } from './ui/AnimatedDialog';
import { useToast } from './ui/Toast';
import {
  IconCheck, IconCheckCircle, IconCopy, IconInfo, IconUpload, IconX,
} from './ui/Icons';
import { copyTextToClipboard } from '../utils/clipboard';

interface ImportModalProps {
  open: boolean;
  onClose: () => void;
  onImportSuccess: (imported: EmailAccount[]) => void;
}

const FORMAT_HINT = '账号----密码----clientid----刷新令牌';

export const ImportModal: React.FC<ImportModalProps> = ({
  open,
  onClose,
  onImportSuccess,
}) => {
  const toast = useToast();
  const [inputText, setInputText] = useState('');

  const parseResult = useMemo(() => {
    if (!inputText.trim()) return { rows: [], accounts: [] };
    return storage.parseAccountLines(inputText);
  }, [inputText]);

  const handleConfirmImport = () => {
    if (parseResult.accounts.length === 0) {
      toast.warning('未能识别到有效的账号数据，请检查输入格式');
      return;
    }
    onImportSuccess(parseResult.accounts);
    toast.success(`成功导入 ${parseResult.accounts.length} 个邮箱账号`);
    setInputText('');
    onClose();
  };

  const validCount = parseResult.accounts.length;
  const invalidCount = parseResult.rows.length - validCount;

  return (
    <AnimatedDialog
      open={open}
      onOpenChange={(o) => { if (!o) onClose(); }}
      title={
        <span className="flex items-center gap-2">
          <IconUpload size={14} strokeWidth={2.2} className="text-ink-500" />
          批量导入邮箱账号
        </span>
      }
      width="max-w-xl"
      footer={
        <>
          <button
            onClick={onClose}
            className="cursor-pointer rounded-xl border border-cold-200 bg-white px-3.5 py-1.5 text-[12.5px] font-medium text-ink-600 transition-colors hover:bg-cold-50"
          >
            取消
          </button>
          <button
            onClick={handleConfirmImport}
            disabled={validCount === 0}
            className="flex cursor-pointer items-center gap-1.5 rounded-xl bg-ink-900 px-4 py-1.5 text-[12.5px] font-medium text-white transition-colors hover:bg-ink-700 disabled:cursor-not-allowed disabled:opacity-45"
          >
            <IconCheck size={12.5} strokeWidth={2.4} />
            确认导入
            <span className="rounded-md bg-white/20 px-1.5 py-px text-[10.5px] font-semibold">
              {validCount}
            </span>
          </button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {/* 格式说明 */}
        <div className="flex gap-3 rounded-2xl border border-cold-200 bg-cold-50 p-3.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-xl bg-white text-ink-400 border border-cold-200">
            <IconInfo size={13} />
          </div>
          <div className="min-w-0">
            <div className="mb-1 text-[12.5px] font-semibold text-ink-800">标准输入格式</div>
            <button
              onClick={() => copyTextToClipboard(FORMAT_HINT, '已复制标准导入格式')}
              className="group flex cursor-pointer items-center gap-1.5 rounded-md border border-cold-200 bg-white px-2 py-0.5 font-mono text-[11px] font-medium text-ink-700 transition-colors hover:border-cold-300"
            >
              {FORMAT_HINT}
              <IconCopy size={10} className="text-ink-300 transition-colors group-hover:text-ink-500" />
            </button>
            <p className="mt-1.5 text-[11.5px] leading-relaxed text-ink-500">
              每行一个账号。同时填写密码与刷新令牌开启 IMAP + Graph 双协议；仅有密码为 IMAP；仅有 Client ID 与刷新令牌为 Graph。
            </p>
          </div>
        </div>

        {/* 输入区 */}
        <div>
          <label className="mb-1.5 block text-[12.5px] font-semibold text-ink-800">
            粘贴账号数据（支持多行批量输入）
          </label>
          <textarea
            rows={5}
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            placeholder={'示例：\nuser1@outlook.com----Password123----d1234567-89ab-cdef----0.ARAA_token_xxx\nuser2@gmail.com----AppPassword456--------\nuser3@hotmail.com--------client_id_here----refresh_token_here'}
            className="scroll-slim w-full resize-none rounded-2xl border border-cold-200 bg-white p-3 font-mono text-[11.5px] leading-relaxed text-ink-800 placeholder-ink-300 outline-none transition-colors duration-150 focus:border-cold-400"
          />
        </div>

        {/* 实时解析预览 */}
        <AnimatePresence>
          {parseResult.rows.length > 0 && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.22, ease: [0.25, 0.46, 0.45, 0.94] }}
              className="overflow-hidden"
            >
              <div className="mb-1.5 flex items-center gap-2">
                <span className="text-[12.5px] font-semibold text-ink-800">实时解析预览</span>
                <span className="flex items-center gap-1 rounded-md bg-emerald-50 px-1.5 py-px text-[10px] font-semibold text-emerald-700">
                  <IconCheckCircle size={10} />
                  {validCount} 有效
                </span>
                {invalidCount > 0 && (
                  <span className="flex items-center gap-1 rounded-md bg-red-50 px-1.5 py-px text-[10px] font-semibold text-red-600">
                    <IconX size={10} strokeWidth={3} />
                    {invalidCount} 无效
                  </span>
                )}
              </div>

              <div className="scroll-slim max-h-44 overflow-y-auto rounded-2xl border border-cold-200">
                <table className="w-full text-left text-[11.5px]">
                  <thead className="sticky top-0 bg-cold-50">
                    <tr>
                      {['状态', '账号', '密码', 'Client ID', '刷新令牌'].map((h) => (
                        <th key={h} className="border-b border-cold-200 px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-ink-400">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {parseResult.rows.map((row, i) => (
                      <tr
                        key={`${row.rawLine}_${i}`}
                        className="border-b border-cold-100 last:border-0 transition-colors hover:bg-cold-50"
                      >
                        <td className="px-3 py-1.5">
                          {row.isValid ? (
                            <span className="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-1.5 py-px text-[10px] font-semibold text-emerald-700">
                              有效
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 rounded-md bg-red-50 px-1.5 py-px text-[10px] font-semibold text-red-600">
                              {row.error || '无效'}
                            </span>
                          )}
                        </td>
                        <td className="max-w-[140px] truncate px-3 py-1.5 font-medium text-ink-800">
                          {row.email || '-'}
                        </td>
                        <td className="px-3 py-1.5">
                          {row.password ? (
                            <span className="rounded-md bg-cold-100 px-1.5 py-px text-[10px] font-medium text-ink-600">已设置</span>
                          ) : (
                            <span className="text-ink-300">无</span>
                          )}
                        </td>
                        <td className="px-3 py-1.5 font-mono text-[10.5px] text-ink-500">
                          {row.clientId ? `${row.clientId.slice(0, 8)}…` : <span className="text-ink-300">无</span>}
                        </td>
                        <td className="px-3 py-1.5 font-mono text-[10.5px] text-ink-500">
                          {row.refreshToken ? `${row.refreshToken.slice(0, 10)}…` : <span className="text-ink-300">无</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </AnimatedDialog>
  );
};
