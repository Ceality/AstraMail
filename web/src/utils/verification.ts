/**
 * 验证码提取：匹配 4-8 位独立的纯数字串（如 42233677）。
 *
 * 边界约束用于排除误报：
 * - 前方不是数字/连字符/点/斜杠（排除日期 2024-11-15、运单号、IP）
 * - 后方不是数字，且不是「点+数字」的小数形态（排除金额 13800.00）
 */

export interface VerificationMatch {
  /** 是否判定为验证码邮件 */
  isVerification: boolean;
  /** 提取到的验证码；判定为验证码但提取失败时为 null */
  code: string | null;
}

/** 全角转半角，中文邮件里的全角数字不漏检 */
function normalize(text: string): string {
  return text
    .replace(/[\uFF01-\uFF5E]/g, (c) => String.fromCharCode(c.charCodeAt(0) - 0xfee0))
    .replace(/\u3000/g, ' ');
}

/**
 * 候选模式，按可靠性从高到低。
 *
 * 每个模式都要求码值是「独立的一小段」：两侧不能是字母数字（排除长单词、
 * 长数字串如订单号/运单号）、也不能是分隔符连接的号码（排除日期、手机号）。
 */
const EXTRACT_PATTERNS: RegExp[] = [
  // 4-8 位纯数字
  /(?<![\d\-\.\/])(\d{4,8})(?!\d)(?!\.\d)/,
];

/** 过滤明显不是验证码的值 */
function isPlausibleCode(code: string): boolean {
  if (code.length < 4 || code.length > 8) return false;
  if (/^(\d)\1+$/.test(code)) return false;  // 000000
  return true;
}

/**
 * 检测并提取验证码。
 *
 * @param subject 邮件主题
 * @param snippet 邮件摘要 / 正文片段
 */
export function detectVerificationCode(
  subject: string,
  snippet: string
): VerificationMatch {
  const text = normalize(`${subject || ''} ${snippet || ''}`);

  for (const pattern of EXTRACT_PATTERNS) {
    const m = pattern.exec(text);
    if (m) {
      const code = m[1].trim();
      if (isPlausibleCode(code)) {
        return { isVerification: true, code };
      }
    }
  }

  return { isVerification: false, code: null };
}
