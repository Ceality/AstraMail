import { toast } from '../components/ui/Toast';

export async function copyTextToClipboard(
  text: string,
  successMsg?: string,
  errorMsg = '复制失败'
): Promise<boolean> {
  if (!text) {
    toast('warning', '没有可复制的内容');
    return false;
  }

  let success = false;

  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      success = true;
    }
  } catch {
    // ignore
  }

  if (success) {
    toast('success', successMsg || '已复制到剪贴板');
  } else {
    toast('error', errorMsg);
  }

  return success;
}
