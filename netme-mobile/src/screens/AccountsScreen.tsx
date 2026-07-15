import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  RefreshControl,
  StyleSheet,
  Alert,
  ActivityIndicator,
  Modal,
  FlatList,
  Animated,
  PanResponder,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useIsFocused, useNavigation } from '@react-navigation/native';
import { plaidService } from '../services/plaidService';
import { PlaidLinkModal } from './PlaidLinkScreen';
import { PlaidConsentModal } from './PlaidConsentModal';
import { Transaction } from '../services/transactionService';
import { fmt, fmtDateLong } from '../utils/format';
import { GLASS } from '../styles/theme';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Account {
  id: string;
  name: string;
  official_name?: string;
  type: string;
  subtype?: string;
  mask?: string;
  current_balance?: number;
  available_balance?: number;
  currency_code: string;
  institution_name: string;
  plaid_item_id: string;
}


interface BankGroup {
  institution_name: string;
  plaid_item_id: string;
  color: string;
  depository: Account[];
  credit: Account[];
  others: Account[];
}

type ViewMode = 'byBank' | 'overview';

// ─── Constants ────────────────────────────────────────────────────────────────

const BANK_COLORS = ['#2dd4a7', '#60a5fa', '#f97316', '#a78bfa', '#f43f5e', '#facc15'];
const TRIAL_BANK_LIMIT = 1;

const isDebt = (type: string) => type === 'credit' || type === 'loan';
const isBalance = (type: string) => type === 'depository';

function groupByBank(accounts: Account[]): BankGroup[] {
  const map = new Map<string, BankGroup>();
  let colorIdx = 0;
  for (const a of accounts) {
    if (!map.has(a.plaid_item_id)) {
      map.set(a.plaid_item_id, {
        institution_name: a.institution_name,
        plaid_item_id: a.plaid_item_id,
        color: BANK_COLORS[colorIdx++ % BANK_COLORS.length],
        depository: [],
        credit: [],
        others: [],
      });
    }
    const g = map.get(a.plaid_item_id)!;
    if (a.type === 'depository') g.depository.push(a);
    else if (a.type === 'credit' || a.type === 'loan') g.credit.push(a);
    else g.others.push(a);
  }
  return Array.from(map.values());
}

// ─── Transaction Modal ────────────────────────────────────────────────────────

const AccountTransactionsModal: React.FC<{ account: Account | null; onClose: () => void; focusVersion: number }> = ({ account, onClose, focusVersion }) => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const translateY = useRef(new Animated.Value(0)).current;
  const navigation = useNavigation<any>();

  const panResponder = useRef(PanResponder.create({
    onStartShouldSetPanResponder: () => true,
    onMoveShouldSetPanResponder: (_, { dy }) => dy > 8,
    onPanResponderMove: (_, { dy }) => { if (dy > 0) translateY.setValue(dy); },
    onPanResponderRelease: (_, { dy }) => {
      if (dy > 120) { onClose(); translateY.setValue(0); }
      else Animated.spring(translateY, { toValue: 0, useNativeDriver: true }).start();
    },
  })).current;

  useEffect(() => {
    if (!account) return;
    translateY.setValue(0);
    setLoading(true);
    plaidService.getTransactions(100, 0, account.id).then(setTransactions).catch(() => {}).finally(() => setLoading(false));
  }, [account, focusVersion]);

  if (!account) return null;

  return (
    <Modal visible animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <Animated.View style={[t.container, { transform: [{ translateY }] }]}>
        <View style={t.dragArea} {...panResponder.panHandlers}>
          <View style={t.dragHandle} />
          <View style={t.header}>
            <View>
              <Text style={t.headerTitle}>{account.name}</Text>
              <Text style={t.headerSub}>{account.subtype || account.type}{account.mask ? ` ••${account.mask}` : ''}</Text>
            </View>
            <TouchableOpacity onPress={onClose}><Text style={t.closeTxt}>Done</Text></TouchableOpacity>
          </View>
        </View>
        <View style={t.balanceRow}>
          {account.current_balance != null && (
            <View style={t.balanceItem}>
              <Text style={t.balanceLabel}>Current</Text>
              <Text style={t.balanceValue}>{fmt(account.current_balance, account.currency_code)}</Text>
            </View>
          )}
          {account.available_balance != null && (
            <View style={t.balanceItem}>
              <Text style={t.balanceLabel}>Available</Text>
              <Text style={t.balanceValue}>{fmt(account.available_balance, account.currency_code)}</Text>
            </View>
          )}
        </View>
        {loading ? (
          <View style={t.center}><ActivityIndicator color="#2dd4a7" /></View>
        ) : transactions.length === 0 ? (
          <View style={t.center}><Text style={t.empty}>No transactions yet</Text></View>
        ) : (
          <FlatList
            data={transactions}
            keyExtractor={item => item.id}
            renderItem={({ item }) => (
              <TouchableOpacity
                style={t.row}
                activeOpacity={0.7}
                onPress={() => {
                  onClose();
                  navigation.navigate('TransactionDetail', { transactionId: item.id });
                }}
              >
                <View style={t.left}>
                  <Text style={t.name} numberOfLines={1}>{item.merchant_name || item.name}</Text>
                  <Text style={t.meta}>
                    {fmtDateLong(item.date)}{item.category ? ` · ${item.category.replace(/_/g, ' ').toLowerCase()}` : ''}
                    {item.pending ? ' · pending' : ''}
                  </Text>
                </View>
                <Text style={[t.amount, item.amount < 0 && t.positive]}>
                  {item.amount < 0 ? '+' : '-'}{fmt(item.amount, item.currency_code)}
                </Text>
              </TouchableOpacity>
            )}
            ItemSeparatorComponent={() => <View style={t.sep} />}
            contentContainerStyle={{ paddingBottom: 40 }}
            bounces={false}
          />
        )}
      </Animated.View>
    </Modal>
  );
};

// ─── Account Card ─────────────────────────────────────────────────────────────

const AccountCard = ({
  account,
  bankColor,
  showBank = false,
  onPress,
}: {
  account: Account;
  bankColor?: string;
  showBank?: boolean;
  onPress: () => void;
}) => {
  const debt = isDebt(account.type);
  return (
    <TouchableOpacity
      style={[s.card, bankColor && { borderLeftColor: bankColor, borderLeftWidth: 3 }]}
      onPress={onPress}
      activeOpacity={0.7}
    >
      <View style={s.cardLeft}>
        <View style={[s.dot, { backgroundColor: debt ? '#f97316' : '#2dd4a7' }]} />
        <View style={{ flex: 1 }}>
          <Text style={s.cardName} numberOfLines={1}>{account.name}</Text>
          <Text style={s.cardSub} numberOfLines={1}>
            {showBank ? `${account.institution_name} · ` : ''}
            {account.subtype || account.type}{account.mask ? ` ••${account.mask}` : ''}
          </Text>
        </View>
      </View>
      <View style={s.cardRight}>
        <Text style={[s.cardBalance, debt && s.cardDebt]}>
          {debt ? '-' : ''}{fmt(account.current_balance ?? 0, account.currency_code)}
        </Text>
        {account.available_balance != null && !debt && (
          <Text style={s.cardAvail}>{fmt(account.available_balance)} avail</Text>
        )}
      </View>
    </TouchableOpacity>
  );
};

// ─── By Bank View ─────────────────────────────────────────────────────────────

const ByBankView = ({
  banks,
  onSelectAccount,
}: {
  banks: BankGroup[];
  onSelectAccount: (a: Account) => void;
}) => {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  const toggle = (id: string) =>
    setCollapsed(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  return (
    <>
      {banks.map(bank => {
        const isCollapsed = collapsed.has(bank.plaid_item_id);
        const total = [...bank.depository, ...bank.credit, ...bank.others]
          .reduce((sum, a) => sum + (isDebt(a.type) ? -(a.current_balance ?? 0) : (a.current_balance ?? 0)), 0);

        return (
          <View key={bank.plaid_item_id} style={s.bankBlock}>
            <TouchableOpacity style={s.bankHeader} onPress={() => toggle(bank.plaid_item_id)} activeOpacity={0.7}>
              <View style={[s.bankColorDot, { backgroundColor: bank.color }]} />
              <Text style={s.bankName}>{bank.institution_name}</Text>
              <Text style={s.bankTotal}>{fmt(total)}</Text>
              <Text style={s.bankChevron}>{isCollapsed ? '›' : '⌄'}</Text>
            </TouchableOpacity>

            {!isCollapsed && (
              <View style={s.bankCards}>
                {bank.depository.length > 0 && (
                  <>
                    <Text style={s.sectionLabel}>Checking & Savings</Text>
                    {bank.depository.map(a => <AccountCard key={a.id} account={a} onPress={() => onSelectAccount(a)} />)}
                  </>
                )}
                {bank.credit.length > 0 && (
                  <>
                    <Text style={s.sectionLabel}>Credit & Loans</Text>
                    {bank.credit.map(a => <AccountCard key={a.id} account={a} onPress={() => onSelectAccount(a)} />)}
                  </>
                )}
                {bank.others.length > 0 && (
                  <>
                    <Text style={s.sectionLabel}>Other</Text>
                    {bank.others.map(a => <AccountCard key={a.id} account={a} onPress={() => onSelectAccount(a)} />)}
                  </>
                )}
              </View>
            )}
          </View>
        );
      })}
    </>
  );
};

// ─── Overview View ────────────────────────────────────────────────────────────

const OverviewView = ({
  banks,
  onSelectAccount,
}: {
  banks: BankGroup[];
  onSelectAccount: (a: Account) => void;
}) => {
  const bankColorMap = new Map(banks.map(b => [b.plaid_item_id, b.color]));

  const allDepository = banks.flatMap(b => b.depository);
  const allCredit     = banks.flatMap(b => [...b.credit, ...b.others]);

  return (
    <>
      {allDepository.length > 0 && (
        <View style={s.overviewSection}>
          <Text style={s.sectionLabel}>Cash & Savings</Text>
          {allDepository.map(a => (
            <AccountCard
              key={a.id}
              account={a}
              bankColor={bankColorMap.get(a.plaid_item_id)}
              showBank
              onPress={() => onSelectAccount(a)}
            />
          ))}
        </View>
      )}
      {allCredit.length > 0 && (
        <View style={s.overviewSection}>
          <Text style={s.sectionLabel}>Credit & Loans</Text>
          {allCredit.map(a => (
            <AccountCard
              key={a.id}
              account={a}
              bankColor={bankColorMap.get(a.plaid_item_id)}
              showBank
              onPress={() => onSelectAccount(a)}
            />
          ))}
        </View>
      )}
    </>
  );
};

// ─── FAB ─────────────────────────────────────────────────────────────────────

const FAB = ({
  banks,
  onConnectBank,
  onSync,
  syncing,
  insetBottom,
}: {
  banks: BankGroup[];
  onConnectBank: () => void;
  onSync: () => void;
  syncing: boolean;
  insetBottom: number;
}) => {
  const [open, setOpen] = useState(false);
  const rotate = useRef(new Animated.Value(0)).current;

  const toggle = () => {
    Animated.spring(rotate, { toValue: open ? 0 : 1, useNativeDriver: true }).start();
    setOpen(o => !o);
  };

  const rotation = rotate.interpolate({ inputRange: [0, 1], outputRange: ['0deg', '45deg'] });

  const handleConnect = () => {
    setOpen(false);
    if (banks.length >= TRIAL_BANK_LIMIT) {
      Alert.alert('Upgrade to Premium', 'Connect unlimited banks with NetMe Premium. Free plan includes 1 bank connection.', [
        { text: 'Maybe Later', style: 'cancel' },
        { text: 'Upgrade', onPress: () => {} },
      ]);
      return;
    }
    onConnectBank();
  };

  const handleSync = () => { setOpen(false); onSync(); };

  return (
    <View style={[s.fabContainer, { bottom: Math.max(insetBottom, 16) + 16 }]}>
      {open && (
        <View style={s.fabOptions}>
          <TouchableOpacity style={s.fabOption} onPress={handleSync} disabled={syncing}>
            {syncing
              ? <ActivityIndicator size="small" color="#fff" />
              : <Text style={s.fabOptionText}>↻  Sync</Text>}
          </TouchableOpacity>
          <TouchableOpacity style={s.fabOption} onPress={handleConnect}>
            <Text style={s.fabOptionText}>🏦  Connect Bank</Text>
          </TouchableOpacity>
        </View>
      )}
      <TouchableOpacity style={s.fabMain} onPress={toggle} activeOpacity={0.85}>
        <Animated.Text style={[s.fabIcon, { transform: [{ rotate: rotation }] }]}>+</Animated.Text>
      </TouchableOpacity>
    </View>
  );
};

// ─── Main Screen ──────────────────────────────────────────────────────────────

export const AccountsScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const isFocused = useIsFocused();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [showConsent, setShowConsent] = useState(false);
  const [showPlaid, setShowPlaid] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<Account | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('byBank');
  const [focusVersion, setFocusVersion] = useState(0);

  const loadAccounts = useCallback(async () => {
    try { setAccounts(await plaidService.getAccounts()); } catch (e: any) { console.error(e); }
  }, []);

  useEffect(() => { loadAccounts().finally(() => setLoading(false)); }, [loadAccounts]);

  useEffect(() => { if (isFocused) setFocusVersion(v => v + 1); }, [isFocused]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await loadAccounts();
    setRefreshing(false);
  }, [loadAccounts]);

  const handlePlaidSuccess = async (publicToken: string, instId?: string, instName?: string) => {
    setShowPlaid(false);
    setSyncing(true);
    try {
      await plaidService.exchangeToken(publicToken, instId, instName);
      await plaidService.syncTransactions();
      await loadAccounts();
      Alert.alert('Connected!', 'Your bank account has been linked successfully.');
    } catch (e: any) {
      Alert.alert('Error', e.message || 'Failed to connect account.');
    } finally { setSyncing(false); }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await plaidService.syncTransactions();
      await loadAccounts();
      Alert.alert('Synced', `${result.transactions_added ?? 0} new transactions added.`);
    } catch (e: any) {
      Alert.alert('Error', e.message || 'Sync failed.');
    } finally { setSyncing(false); }
  };

  const totalBalance = accounts.filter(a => isBalance(a.type)).reduce((sum, a) => sum + (a.current_balance ?? 0), 0);
  const totalDebt    = accounts.filter(a => isDebt(a.type)).reduce((sum, a) => sum + Math.abs(a.current_balance ?? 0), 0);
  const banks = groupByBank(accounts);

  if (loading) {
    return <View style={s.center}><ActivityIndicator size="large" color="#2dd4a7" /></View>;
  }

  return (
    <View style={[s.container, { paddingTop: insets.top }]}>
      <ScrollView
        style={s.scroll}
        contentContainerStyle={s.scrollContent}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor="#2dd4a7" />}
      >
        {/* Hero */}
        <View style={s.hero}>
          <View style={s.totalsRow}>
            <View style={s.totalItem}>
              <Text style={s.totalLabel}>Balance</Text>
              <Text style={s.totalValue} adjustsFontSizeToFit numberOfLines={1} minimumFontScale={0.5}>
                {fmt(totalBalance)}
              </Text>
            </View>
            <View style={s.totalDivider} />
            <View style={s.totalItem}>
              <Text style={s.totalLabel}>Debt</Text>
              <Text style={[s.totalValue, s.debtValue]} adjustsFontSizeToFit numberOfLines={1} minimumFontScale={0.5}>
                {fmt(totalDebt)}
              </Text>
            </View>
          </View>
        </View>

        {accounts.length === 0 ? (
          <View style={s.empty}>
            <Text style={s.emptyIcon}>🏦</Text>
            <Text style={s.emptyTitle}>No accounts yet</Text>
            <Text style={s.emptySubtitle}>Tap + to connect your first bank account</Text>
          </View>
        ) : (
          <>
            {/* View toggle */}
            <View style={s.toggle}>
              <TouchableOpacity style={[s.toggleBtn, viewMode === 'byBank' && s.toggleBtnActive]} onPress={() => setViewMode('byBank')}>
                <Text style={[s.toggleText, viewMode === 'byBank' && s.toggleTextActive]}>By Bank</Text>
              </TouchableOpacity>
              <TouchableOpacity style={[s.toggleBtn, viewMode === 'overview' && s.toggleBtnActive]} onPress={() => setViewMode('overview')}>
                <Text style={[s.toggleText, viewMode === 'overview' && s.toggleTextActive]}>Overview</Text>
              </TouchableOpacity>
            </View>

            {viewMode === 'byBank'
              ? <ByBankView banks={banks} onSelectAccount={setSelectedAccount} />
              : <OverviewView banks={banks} onSelectAccount={setSelectedAccount} />
            }
          </>
        )}

        <View style={{ height: 100 }} />
      </ScrollView>

      <FAB
        banks={banks}
        onConnectBank={() => setShowConsent(true)}
        onSync={handleSync}
        syncing={syncing}
        insetBottom={insets.bottom}
      />

      <PlaidConsentModal
        visible={showConsent}
        onAccept={() => { setShowConsent(false); setShowPlaid(true); }}
        onDecline={() => setShowConsent(false)}
      />
      <PlaidLinkModal visible={showPlaid} onSuccess={handlePlaidSuccess} onClose={() => setShowPlaid(false)} />
      {selectedAccount && <AccountTransactionsModal account={selectedAccount} onClose={() => setSelectedAccount(null)} focusVersion={focusVersion} />}
    </View>
  );
};

// ─── Styles ───────────────────────────────────────────────────────────────────

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'transparent' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  scroll: { flex: 1 },
  scrollContent: { padding: 16, paddingBottom: 40, gap: 12 },

  // Hero
  hero: { ...GLASS, padding: 20 },
  totalsRow: { flexDirection: 'row', alignItems: 'center' },
  totalItem: { flex: 1, alignItems: 'center' },
  totalDivider: { width: 1, height: 40, backgroundColor: 'rgba(255,255,255,0.15)' },
  totalLabel: { color: 'rgba(255,255,255,0.5)', fontSize: 12, marginBottom: 4, textTransform: 'uppercase', letterSpacing: 0.5 },
  totalValue: { color: '#fff', fontSize: 26, fontWeight: '700' },
  debtValue: { color: '#fca5a5' },

  // View toggle
  toggle: { flexDirection: 'row', ...GLASS, padding: 4, gap: 4 },
  toggleBtn: { flex: 1, paddingVertical: 8, borderRadius: 12, alignItems: 'center' },
  toggleBtnActive: { backgroundColor: 'rgba(45,212,167,0.2)', borderWidth: 1, borderColor: 'rgba(45,212,167,0.35)' },
  toggleText: { color: 'rgba(255,255,255,0.4)', fontSize: 14, fontWeight: '500' },
  toggleTextActive: { color: '#2dd4a7', fontWeight: '600' },

  // By Bank
  bankBlock: { gap: 0 },
  bankHeader: {
    ...GLASS,
    flexDirection: 'row',
    alignItems: 'center',
    padding: 14,
    gap: 10,
  },
  bankColorDot: { width: 10, height: 10, borderRadius: 5 },
  bankName: { flex: 1, fontSize: 15, fontWeight: '700', color: '#fff' },
  bankTotal: { fontSize: 14, fontWeight: '600', color: 'rgba(255,255,255,0.6)' },
  bankChevron: { fontSize: 18, color: 'rgba(255,255,255,0.4)', marginLeft: 4 },
  bankCards: { gap: 6, paddingTop: 6 },

  // Overview
  overviewSection: { gap: 6 },

  // Shared
  sectionLabel: { fontSize: 11, fontWeight: '600', color: 'rgba(255,255,255,0.35)', textTransform: 'uppercase', letterSpacing: 0.8, paddingLeft: 2, marginTop: 4 },
  card: {
    ...GLASS,
    padding: 14,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    overflow: 'hidden',
  },
  cardLeft: { flexDirection: 'row', alignItems: 'center', gap: 10, flex: 1 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  cardName: { fontSize: 14, fontWeight: '600', color: '#fff' },
  cardSub: { fontSize: 12, color: 'rgba(255,255,255,0.4)', marginTop: 1, textTransform: 'capitalize' },
  cardRight: { alignItems: 'flex-end' },
  cardBalance: { fontSize: 15, fontWeight: '700', color: '#fff' },
  cardDebt: { color: '#fca5a5' },
  cardAvail: { fontSize: 11, color: 'rgba(255,255,255,0.35)', marginTop: 2 },

  // Empty
  empty: { alignItems: 'center', paddingTop: 60, paddingHorizontal: 32 },
  emptyIcon: { fontSize: 48, marginBottom: 12 },
  emptyTitle: { fontSize: 20, fontWeight: '700', color: '#fff', marginBottom: 8 },
  emptySubtitle: { fontSize: 14, color: 'rgba(255,255,255,0.5)', textAlign: 'center' },

  // FAB
  fabContainer: { position: 'absolute', right: 20, alignItems: 'flex-end', gap: 10 },
  fabOptions: { gap: 8, alignItems: 'flex-end' },
  fabOption: {
    backgroundColor: 'rgba(15,30,60,0.95)',
    borderRadius: 20,
    paddingVertical: 10,
    paddingHorizontal: 18,
    borderWidth: 1,
    borderColor: 'rgba(45,212,167,0.3)',
  },
  fabOptionText: { color: '#fff', fontSize: 14, fontWeight: '600' },
  fabMain: {
    width: 52,
    height: 52,
    borderRadius: 26,
    backgroundColor: '#2dd4a7',
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#2dd4a7',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 8,
  },
  fabIcon: { color: '#0f172a', fontSize: 28, fontWeight: '300', lineHeight: 32 },
});

// Transaction modal (system sheet — stays light)
const t = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#fff' },
  dragArea: { paddingTop: 10, borderBottomWidth: 1, borderBottomColor: '#f1f5f9' },
  dragHandle: { width: 36, height: 4, borderRadius: 2, backgroundColor: '#cbd5e1', alignSelf: 'center', marginBottom: 8 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-start', paddingHorizontal: 16, paddingBottom: 14 },
  headerTitle: { fontSize: 17, fontWeight: '700', color: '#1e3a5f' },
  headerSub: { fontSize: 13, color: '#64748b', marginTop: 2, textTransform: 'capitalize' },
  closeTxt: { fontSize: 16, color: '#2dd4a7', fontWeight: '600' },
  balanceRow: { flexDirection: 'row', backgroundColor: '#f8fafc', paddingVertical: 14, paddingHorizontal: 20, gap: 32, borderBottomWidth: 1, borderBottomColor: '#e2e8f0' },
  balanceItem: {},
  balanceLabel: { fontSize: 11, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  balanceValue: { fontSize: 20, fontWeight: '700', color: '#1e3a5f' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  empty: { color: '#94a3b8', fontSize: 15 },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-start', paddingHorizontal: 16, paddingVertical: 12 },
  left: { flex: 1, paddingRight: 12 },
  name: { fontSize: 14, fontWeight: '500', color: '#1e3a5f' },
  meta: { fontSize: 12, color: '#94a3b8', marginTop: 2, textTransform: 'capitalize' },
  amount: { fontSize: 14, fontWeight: '600', color: '#ef4444' },
  positive: { color: '#16a34a' },
  sep: { height: 1, backgroundColor: '#f1f5f9', marginHorizontal: 16 },
});
