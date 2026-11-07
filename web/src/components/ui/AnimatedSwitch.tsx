import React from 'react';
import { Switch } from '@base-ui/react/switch';
import { cn } from '../../utils/cn';

interface AnimatedSwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  label?: string;
  /** 无可见标签时提供无障碍名称 */
  disabled?: boolean;
  ariaLabel?: string;
}

export const AnimatedSwitch: React.FC<AnimatedSwitchProps> = ({
  checked,
  onCheckedChange,
  label,
  ariaLabel,
  disabled,
}) => {
  return (
    <label
      className={cn(
        'flex cursor-pointer items-center gap-2 select-none',
        disabled && 'cursor-not-allowed opacity-45'
      )}
    >
      <Switch.Root
        checked={checked}
        disabled={disabled}
        onCheckedChange={onCheckedChange}
        aria-label={ariaLabel ?? label}
        className={cn(
          'relative h-[20px] w-[36px] rounded-full p-[2px] transition-colors duration-200 outline-none',
          'focus-visible:ring-2 focus-visible:ring-ink-200',
          checked
            ? 'bg-ink-900'
            : 'bg-cold-200 hover:bg-cold-300'
        )}
      >
        <Switch.Thumb
          className={cn(
            'block h-[16px] w-[16px] rounded-full bg-white shadow-xs',
            'transition-transform duration-200 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]',
            'data-[checked]:translate-x-4'
          )}
        />
      </Switch.Root>
      {label && (
        <span className="text-[12.5px] font-medium text-ink-600">{label}</span>
      )}
    </label>
  );
};
