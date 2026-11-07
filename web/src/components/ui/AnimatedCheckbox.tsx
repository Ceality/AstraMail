import React from 'react';
import { Checkbox } from '@base-ui/react/checkbox';
import { cn } from '../../utils/cn';

interface AnimatedCheckboxProps {
  checked: boolean;
  onCheckedChange?: (checked: boolean) => void;
  indeterminate?: boolean;
  disabled?: boolean;
  label?: React.ReactNode;
  className?: string;
}

/**
 * 复选框。
 *
 * 刻意不使用 Framer Motion 的 layout / layoutId：那类动画需要在每次渲染时强制
 * 读取 getBoundingClientRect 触发重排，在长列表里逐个勾选会造成明显卡顿。
 * 这里全部用 CSS 过渡（background-color / stroke-dashoffset），不触发布局。
 */
export const AnimatedCheckbox: React.FC<AnimatedCheckboxProps> = ({
  checked,
  onCheckedChange,
  indeterminate = false,
  disabled = false,
  label,
  className,
}) => {
  const isOn = checked || indeterminate;
  const isIndeterminate = indeterminate && !checked;
  const dash = 'cubic-bezier(0.25,0.46,0.45,0.94)';

  const box = (
    <Checkbox.Root
      checked={checked}
      indeterminate={indeterminate}
      disabled={disabled}
      onCheckedChange={(c) => onCheckedChange?.(c === true)}
      className={cn(
        'relative flex h-[16px] w-[16px] shrink-0 items-center justify-center overflow-hidden rounded-[6px]',
        'border cursor-pointer outline-none',
        'focus-visible:ring-2 focus-visible:ring-ink-200',
        'transition-[background-color,border-color] duration-200',
        isOn
          ? 'border-ink-900 bg-ink-900'
          : 'border-cold-300 bg-white hover:border-cold-400',
        disabled && 'opacity-40 cursor-not-allowed',
        className
      )}
      style={{ transitionTimingFunction: dash }}
    >
      <svg
        width="10"
        height="10"
        viewBox="0 0 24 24"
        fill="none"
        stroke="white"
        strokeWidth="3.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden
        className={cn(
          'pointer-events-none absolute transition-[stroke-dashoffset,opacity] duration-200',
          checked ? 'opacity-100' : 'opacity-0'
        )}
        style={{ strokeDasharray: 24, strokeDashoffset: checked ? 0 : 24, transitionTimingFunction: dash }}
      >
        <path d="M20 6 9 17l-5-5" />
      </svg>
      <svg
        width="10"
        height="10"
        viewBox="0 0 24 24"
        fill="none"
        stroke="white"
        strokeWidth="3.5"
        strokeLinecap="round"
        aria-hidden
        className={cn(
          'pointer-events-none absolute transition-[stroke-dashoffset,opacity] duration-200',
          isIndeterminate ? 'opacity-100' : 'opacity-0'
        )}
        style={{ strokeDasharray: 14, strokeDashoffset: isIndeterminate ? 0 : 14, transitionTimingFunction: dash }}
      >
        <line x1="5" x2="19" y1="12" y2="12" />
      </svg>
    </Checkbox.Root>
  );

  if (label === undefined) return box;

  return (
    <label className={cn('flex cursor-pointer items-center gap-2 select-none', disabled && 'cursor-not-allowed opacity-50')}>
      {box}
      <span className="text-[12.5px] font-medium text-ink-600">{label}</span>
    </label>
  );
};
