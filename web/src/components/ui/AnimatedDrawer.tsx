import React from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { AnimatePresence, motion } from 'framer-motion';
import { cn } from '../../utils/cn';
import { IconX } from './Icons';

interface AnimatedDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: React.ReactNode;
  children: React.ReactNode;
  side?: 'left' | 'right';
  widthClass?: string;
  headerExtra?: React.ReactNode;
}

export const AnimatedDrawer: React.FC<AnimatedDrawerProps> = ({
  open,
  onOpenChange,
  title,
  children,
  side = 'right',
  widthClass = 'w-full sm:w-[640px]',
  headerExtra,
}) => {
  const isRight = side === 'right';

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open && (
          <Dialog.Portal keepMounted>
            <Dialog.Backdrop
              render={
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.18 }}
                />
              }
              className="fixed inset-0 z-[70] bg-ink-950/18"
            />
            <Dialog.Popup
              render={
                <motion.div
                  initial={{ x: isRight ? '100%' : '-100%' }}
                  animate={{ x: 0 }}
                  exit={{ x: isRight ? '100%' : '-100%' }}
                  transition={{ duration: 0.22, ease: [0.25, 0.46, 0.45, 0.94] }}
                />
              }
              className={cn(
                'fixed inset-y-0 z-[75] flex flex-col bg-white outline-none',
                isRight
                  ? 'right-0 border-l border-cold-200'
                  : 'left-0 border-r border-cold-200',
                widthClass
              )}
            >
              {(title !== undefined || headerExtra) && (
                <div className="flex shrink-0 items-center justify-between gap-3 border-b border-cold-100 px-5 py-3.5">
                  <Dialog.Title className="min-w-0 flex-1 truncate text-[14px] font-semibold tracking-tight text-ink-900">
                    {title}
                  </Dialog.Title>
                  <div className="flex shrink-0 items-center gap-1.5">
                    {headerExtra}
                    <Dialog.Close
                      className={cn(
                        'rounded-xl p-1 text-ink-400 transition-colors duration-150',
                        'hover:bg-cold-100 hover:text-ink-700',
                        'cursor-pointer outline-none'
                      )}
                      aria-label="关闭"
                    >
                      <IconX size={15} />
                    </Dialog.Close>
                  </div>
                </div>
              )}
              <div className="scroll-slim flex-1 overflow-y-auto">
                <Dialog.Description className="contents" render={<div />}>
                  {children}
                </Dialog.Description>
              </div>
            </Dialog.Popup>
          </Dialog.Portal>
        )}
      </AnimatePresence>
    </Dialog.Root>
  );
};
