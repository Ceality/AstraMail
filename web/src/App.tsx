import { useMemo, useState } from 'react';
import type { EmailAccount } from './types/account';
import { storage } from './services/storage';
import { AccountSidebar } from './components/AccountSidebar';
import { EmailStreamView } from './components/EmailStreamView';
import { ImportModal } from './components/ImportModal';
import { AnimatedDrawer } from './components/ui/AnimatedDrawer';
import { ToastProvider, useToast } from './components/ui/Toast';
import { useIsMobile } from './hooks/useIsMobile';
import { selectAllAccountIds } from './utils/selection';

function AppInner() {
  const isMobile = useIsMobile();
  const toast = useToast();
  const [mobileDrawerVisible, setMobileDrawerVisible] = useState(false);

  const [accounts, setAccounts] = useState<EmailAccount[]>(() => storage.getAccounts());
  const [selectedAccountIds, setSelectedAccountIds] = useState<string[]>(() => {
    const loaded = storage.getAccounts();
    return loaded.length > 0 ? [loaded[0].id] : [];
  });
  const [activeAccountId, setActiveAccountId] = useState<string | null>(() => {
    const loaded = storage.getAccounts();
    return loaded[0]?.id || null;
  });

  const [importModalVisible, setImportModalVisible] = useState(false);

  const updateAccounts = (newAccs: EmailAccount[]) => {
    setAccounts(newAccs);
    storage.saveAccounts(newAccs);
  };

  const handleToggleSelectAccount = (id: string) => {
    setSelectedAccountIds((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    );
  };

  const handleSelectAllAccounts = (checked: boolean) => {
    setSelectedAccountIds(selectAllAccountIds(accounts, selectedAccountIds, checked));
  };

  const handleDeleteAccount = (id: string) => {
    const target = accounts.find((a) => a.id === id);
    const updated = accounts.filter((a) => a.id !== id);
    updateAccounts(updated);
    setSelectedAccountIds((prev) => prev.filter((i) => i !== id));
    if (activeAccountId === id) {
      setActiveAccountId(updated[0]?.id || null);
    }
    toast.success(target ? `已删除账号: ${target.email}` : '账号已删除');
  };

  const handleClearAccounts = () => {
    const count = accounts.length;
    updateAccounts([]);
    setSelectedAccountIds([]);
    setActiveAccountId(null);
    toast.success(`已清空全部 ${count} 个账号`);
  };

  const handleImportSuccess = (imported: EmailAccount[]) => {
    const updated = [...accounts];
    const newSelected = [...selectedAccountIds];

    for (const acc of imported) {
      const idx = updated.findIndex((a) => a.email.toLowerCase() === acc.email.toLowerCase());
      if (idx >= 0) {
        const existing = updated[idx] as Record<string, any>;
        updated[idx] = {
          ...updated[idx],
          password: acc.password || updated[idx].password,
          clientId: acc.clientId || updated[idx].clientId || existing.client_id,
          refreshToken: acc.refreshToken || updated[idx].refreshToken || existing.refresh_token,
          protocol: acc.protocol,
        };
      } else {
        updated.push(acc);
        newSelected.push(acc.id);
      }
    }

    updateAccounts(updated);
    setSelectedAccountIds(newSelected);
    if (!activeAccountId && updated.length > 0) {
      setActiveAccountId(updated[0].id);
    }
  };

  const activeAccount = accounts.find((a) => a.id === activeAccountId) || accounts[0] || null;
  const selectedAccounts = useMemo(
    () => accounts.filter((a) => selectedAccountIds.includes(a.id)),
    [accounts, selectedAccountIds]
  );

  const sidebarContent = (
    <AccountSidebar
      accounts={accounts}
      selectedAccountIds={selectedAccountIds}
      activeAccountId={activeAccountId}
      onToggleSelectAccount={handleToggleSelectAccount}
      onSelectAllAccounts={handleSelectAllAccounts}
      onSetActiveAccount={(acc) => {
        setActiveAccountId(acc.id);
        if (isMobile) setMobileDrawerVisible(false);
      }}
      onOpenImportModal={() => {
        if (isMobile) setMobileDrawerVisible(false);
        setImportModalVisible(true);
      }}
      onDeleteAccount={handleDeleteAccount}
      onClearAccounts={handleClearAccounts}
    />
  );

  return (
    <div className="app-bg relative flex h-full w-full overflow-hidden">
      {/* 桌面侧边栏 */}
      {!isMobile && (
        <aside className="h-full w-[320px] shrink-0 bg-white">
          {sidebarContent}
        </aside>
      )}

      {/* 移动端侧边栏抽屉 */}
      <AnimatedDrawer
        open={isMobile && mobileDrawerVisible}
        onOpenChange={setMobileDrawerVisible}
        side="left"
        widthClass="w-[86vw] max-w-[320px]"
      >
        {sidebarContent}
      </AnimatedDrawer>

      {/* 主内容区 */}
      <main className="flex h-full min-w-0 flex-1 flex-col overflow-hidden">
        <EmailStreamView
          accounts={accounts}
          selectedAccounts={selectedAccounts}
          selectedAccountIds={selectedAccountIds}
          activeAccount={activeAccount}
          isMobile={isMobile}
          onOpenMobileSidebar={() => setMobileDrawerVisible(true)}
        />
      </main>

      {/* 导入弹窗 */}
      <ImportModal
        open={importModalVisible}
        onClose={() => setImportModalVisible(false)}
        onImportSuccess={handleImportSuccess}
      />
    </div>
  );
}

export function App() {
  return (
    <ToastProvider>
      <AppInner />
    </ToastProvider>
  );
}

export default App;
