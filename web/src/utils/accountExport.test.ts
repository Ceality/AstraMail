import test from 'node:test';
import assert from 'node:assert/strict';
import { storage } from '../services/storage.ts';
import type { EmailAccount } from '../types/account.ts';

function account(patch: Partial<EmailAccount>): EmailAccount {
  return {
    id: 'acc_test',
    email: 'user@example.com',
    protocol: 'imap',
    createdAt: 0,
    ...patch,
  };
}

test('IMAP 导出不追加空字段分隔符', () => {
  const line = storage.exportToLineFormat([
    account({ email: '13459256373@189.cn', password: 'ok337491' }),
  ]);
  assert.equal(line, '13459256373@189.cn----ok337491');
});

test('Graph 导出保留 password 空位以对齐列', () => {
  const line = storage.exportToLineFormat([
    account({
      email: 'user@example.com',
      protocol: 'graph',
      clientId: 'client-id',
      refreshToken: 'refresh-token',
    }),
  ]);
  assert.equal(line, 'user@example.com--------client-id----refresh-token');
});

test('双协议导出完整四列', () => {
  const line = storage.exportToLineFormat([
    account({
      email: 'user@example.com',
      password: 'pass',
      protocol: 'both',
      clientId: 'client-id',
      refreshToken: 'refresh-token',
    }),
  ]);
  assert.equal(line, 'user@example.com----pass----client-id----refresh-token');
});

test('导出结果可被重新解析', () => {
  const source = account({ email: '13459256373@189.cn', password: 'ok337491' });
  const line = storage.exportToLineFormat([source]);
  const { accounts } = storage.parseAccountLines(line);
  assert.equal(accounts.length, 1);
  assert.equal(accounts[0].email, source.email);
  assert.equal(accounts[0].password, source.password);
  assert.equal(accounts[0].protocol, 'imap');
});
