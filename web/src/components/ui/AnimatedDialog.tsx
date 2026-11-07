import React from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { AnimatePresence, motion } from 'framer-motion';
import { cn } from '../../utils/cn';
import { IconX } from './Icons';

interface AnimatedDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  width?: string;
}

export const AnimatedDialog: React.FC<AnimatedDialogProps> = ({
  open,
  onOpenChange,
  title,
  children,
  footer,
  className,
  width = 'max-w-lg',
}) => {
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
                  transition={{ duration: 0.16 }}
                />
              }
              className="fixed inset-0 z-[80] bg-ink-950/20"
            />
            <Dialog.Popup
              render={
                <motion.div
                  initial={{ opacity: 0, scale: 0.97, y: 8 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.97, y: 6 }}
                  transition={{ duration: 0.18, ease: [0.25, 0.46, 0.45, 0.94] }}
                />
              }
              className={cn(
                'fixed left-1/2 top-1/2 z-[90] w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2',
                'rounded-3xl border border-cold-200 bg-white shadow-modal outline-none',
                'max-h-[85vh] flex flex-col overflow-hidden',
                width,
                className
              )}
            >
              {title !== undefined && (
                <div className="flex shrink-0 items-center justify-between border-b border-cold-100 px-5 py-3.5">
                  <Dialog.Title className="text-[14px] font-semibold tracking-tight text-ink-900">
                    {title}
                  </Dialog.Title>
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
              )}

              <div className="scroll-slim flex-1 overflow-y-auto px-5 py-4">
                <Dialog.Description className="contents" render={<div />}>
                  {children}
                </Dialog.Description>
              </div>

              {footer && (
                <div className="flex shrink-0 items-center justify-end gap-2 border-t border-cold-100 px-5 py-3.5">
                  {footer}
                </div>
              )}
            </Dialog.Popup>
          </Dialog.Portal>
        )}
      </AnimatePresence>
    </Dialog.Root>
  );
};
