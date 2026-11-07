import { test } from 'node:test';
import assert from 'node:assert/strict';
import { detectVerificationCode } from './verification.ts';

test('提取 6 位数字验证码', () => {
  const r = detectVerificationCode('您的验证码', '验证码：123456，5分钟内有效');
  assert.equal(r.code, '123456');
});

test('提取 8 位数字验证码', () => {
  const r = detectVerificationCode('通知', '您的编号 42233677 已生成');
  assert.equal(r.code, '42233677');
});

test('提取 4 位数字验证码', () => {
  const r = detectVerificationCode('GitHub', 'Verification code: 080211.');
  assert.equal(r.code, '080211');
});

test('运单号不误判', () => {
  const r = detectVerificationCode('订单发货', '运单号 SF1234567890 已发出');
  assert.equal(r.isVerification, false);
});

test('金额小数不误判', () => {
  const r = detectVerificationCode('月度账单', '消费 13800.00 元');
  assert.equal(r.isVerification, false);
});

test('手机号不误判', () => {
  const r = detectVerificationCode('通知', '手机 13800138000');
  assert.equal(r.isVerification, false);
});

test('全同数字拒绝', () => {
  const r = detectVerificationCode('验证码', '验证码：000000');
  assert.equal(r.code, null);
});
