import { useEffect, useRef } from 'react';

interface UseAutoFetchOptions {
  /** 是否启用 */
  enabled: boolean;
  /** 间隔（秒） */
  intervalSeconds: number;
  /** 每次触发的回调 */
  onTick: () => void | Promise<void>;
  /** 回调执行中时跳过本次触发，避免请求堆积 */
  skipIfBusy?: boolean;
}

/**
 * 定时自动收件。
 *
 * 用 setTimeout 递归而非 setInterval，避免上一轮请求未完成时堆积触发。
 * 间隔变化时重建定时器，保证新间隔立即生效。卸载时清理定时器，
 * 避免内存泄漏和幽灵请求。
 */
export function useAutoFetch({
  enabled,
  intervalSeconds,
  onTick,
  skipIfBusy = true,
}: UseAutoFetchOptions) {
  const busyRef = useRef(false);
  const onTickRef = useRef(onTick);

  // 始终指向最新的回调，避免因回调重建而重置定时器
  useEffect(() => {
    onTickRef.current = onTick;
  }, [onTick]);

  useEffect(() => {
    if (!enabled) return;

    const seconds = Math.max(1, intervalSeconds);
    let timer: number | undefined;
    let cancelled = false;

    const schedule = () => {
      if (cancelled) return;
      timer = window.setTimeout(tick, seconds * 1000);
    };

    const tick = async () => {
      if (cancelled) return;

      if (skipIfBusy && busyRef.current) {
        schedule();
        return;
      }

      busyRef.current = true;
      try {
        await onTickRef.current();
      } finally {
        busyRef.current = false;
        schedule();
      }
    };

    schedule();

    return () => {
      cancelled = true;
      if (timer !== undefined) window.clearTimeout(timer);
    };
  }, [enabled, intervalSeconds, skipIfBusy]);
}
