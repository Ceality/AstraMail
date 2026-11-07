import React, { createContext, useCallback, useContext, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { cn } from '../../utils/cn';
import { IconCheckCircle, IconInfo, IconAlertCircle } from './Icons';

export type ToastType = 'success' | 'info' | 'warning' | 'error';

interface ToastItem {
  id: number;
  type: ToastType;
  content: string;
}

interface ToastContextValue {
  toast: (type: ToastType, content: string, duration?: number) => number;
  /** 按 id 提前关闭指定 Toast（用于加载类提示） */
  dismiss: (id: number) => void;
  success: (content: string, duration?: number) => number;
  info: (content: string, duration?: number) => number;
  warning: (content: string, duration?: number) => number;
  error: (content: string, duration?: number) => number;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}

const typeConfig: Record<ToastType, { icon: React.ReactNode; color: string }> = {
  success: { icon: <IconCheckCircle size={15} />, color: 'text-emerald-600' },
  info: { icon: <IconInfo size={15} />, color: 'text-ink-500' },
  warning: { icon: <IconAlertCircle size={15} />, color: 'text-amber-600' },
  error: { icon: <IconAlertCircle size={15} />, color: 'text-red-600' },
};

let globalToastFn: ((type: ToastType, content: string, duration?: number) => number) | null = null;

export function toast(type: ToastType, content: string, duration?: number): number | undefined {
  return globalToastFn?.(type, content, duration);
}

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const idRef = useRef(0);

  const push = useCallback((type: ToastType, content: string, duration = 2600) => {
    const id = ++idRef.current;
    setToasts((prev) => {
      const next = [...prev, { id, type, content }];
      return next.length > 3 ? next.slice(next.length - 3) : next;
    });
    window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, duration);
    return id;
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  globalToastFn = push;

  const value: ToastContextValue = {
    toast: push,
    dismiss,
    success: (c, d) => push('success', c, d ?? 1800),
    info: (c, d) => push('info', c, d ?? 2200),
    warning: (c, d) => push('warning', c, d ?? 2400),
    error: (c, d) => push('error', c, d ?? 2800),
  };

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed inset-x-0 top-4 z-[100] flex flex-col items-center gap-1.5 px-4">
        <AnimatePresence initial={false}>
          {toasts.map((t) => {
            const cfg = typeConfig[t.type];
            return (
              <motion.div
                key={t.id}
                initial={{ opacity: 0, y: -12, scale: 0.97 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -8, scale: 0.97 }}
                transition={{ duration: 0.18, ease: [0.25, 0.46, 0.45, 0.94] }}
                className={cn(
                  'pointer-events-auto flex max-w-sm items-center gap-2',
                  // 圆角 0.75rem；Toast 本身只有约 34px 高，用更大的圆角会变成胶囊形
                  'rounded-[0.75rem] border border-cold-200/75 bg-white/75 py-2 pl-3 pr-3.5 shadow-pop',
                  'backdrop-blur-md backdrop-saturate-150',
                  'text-[13px] font-medium text-ink-800'
                )}
              >
                <span className={cn('shrink-0', cfg.color)}>{cfg.icon}</span>
                <span className="leading-snug">{t.content}</span>
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>
    </ToastContext.Provider>
  );
};
