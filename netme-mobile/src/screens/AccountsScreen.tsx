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
  SafeAreaView,
  FlatList,
  Animated,
  PanResponder,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { plaidService } from '../services/plaidService';
import { PlaidLinkModal } from './PlaidLinkScreen';

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

interface Transaction {
  id: string;
  name: string;
  merchant_name?: string;
  amount: number;
  currency_code: string;
  date: string;
  category?: string;
  payment_channel?: string;
  pending: boolean;
}

interface BankGroup {
  institution_name: string;
  plaid_item_id: string;
  depository: Account[];
  credit: Account[];
  others: Account[];
}

const fmt = (amount: number, currency = 'USD') =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(Math.abs(amount));

// Plaid dates are DATE-only ("2026-03-04") but lib/pq may serialize as "2026-03-04T00:00:00Z".
// Strip the time part and parse manually to avoid UTC-midnight shifting the day in local tz.
const fmtDate = (dateStr: string) => {
  if (!dateStr) return '';
  const datePart = dateStr.split('T')[0]; // "2026-03-04"
  const parts = datePart.split('-');
  if (parts.length !== 3) return dateStr;
  const [y, m, d] = parts.map(Number);
  if (isNaN(y) || isNaN(m) || isNaN(d)) return dateStr;
  return new Date(y, m - 1, d).toLocaleDateString('en-US', {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  });
};

const isDebt = (type: string) => type === 'credit' || type === 'loan';
const isBalance = (type: string) => type === 'depository';

function groupByBank(accounts: Account[]): BankGroup[] {
  const map = new Map<string, BankGroup>();
  for (const a of accounts) {
    if (!map.has(a.plaid_item_id)) {
      map.set(a.plaid_item_id, {
        institution_name: a.institution_name,
        plaid_item_id: a.plaid_item_id,
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

const TransactionRow = ({ txn }: { txn: Transaction }) => (
  <View style={txnStyles.row}>
    <View style={txnStyles.left}>
      <Text style={txnStyles.name} numberOfLines={1}>
        {txn.merchant_name || txn.name}
      </Text>
      <Text style={txnStyles.meta}>
        {fmtDate(txn.date)}{txn.category ? ` · ${txn.category.replace(/_/g, ' ').toLowerCase()}` : ''}
        {txn.pending ? ' · pending' : ''}
      </Text>
    </View>
    <Text style={[txnStyles.amount, txn.amount < 0 && txnStyles.positive]}>
      {txn.amount < 0 ? '+' : '-'}{fmt(txn.amount, txn.currency_code)}
    </Text>
  </View>
);

interface TxnModalProps {
  account: Account | null;
  onClose: () => void;
}

const AccountTransactionsModal: React.FC<TxnModalProps> = ({ account, onClose }) => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const translateY = useRef(new Animated.Value(0)).current;

  const panResponder = useRef(
    PanResponder.create({
      onStartShouldSetPanResponder: () => true,
      onMoveShouldSetPanResponder: (_, { dy }) => dy > 8,
      onPanResponderMove: (_, { dy }) => {
        if (dy > 0) translateY.setValue(dy);
      },
      onPanResponderRelease: (_, { dy }) => {
        if (dy > 120) {
          onClose();
          translateY.setValue(0);
        } else {
          Animated.spring(translateY, { toValue: 0, useNativeDriver: true }).start();
        }
      },
    })
  ).current;

  useEffect(() => {
    if (!account) return;
    translateY.setValue(0);
    setLoading(true);
    plaidService
      .getTransactions(100, 0, account.id)
      .then(setTransactions)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [account]);

  if (!account) return null;

  return (
    <Modal visible animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <Animated.View style={[txnStyles.container, { transform: [{ translateY }] }]}>
        {/* Drag handle */}
        <View style={txnStyles.dragArea} {...panResponder.panHandlers}>
          <View style={txnStyles.dragHandle} />
          <View style={txnStyles.header}>
            <View>
              <Text style={txnStyles.headerTitle}>{account.name}</Text>
              <Text style={txnStyles.headerSub}>
                {account.subtype || account.type}{account.mask ? ` ••${account.mask}` : ''}
              </Text>
            </View>
            <TouchableOpacity onPress={onClose} style={txnStyles.closeBtn}>
              <Text style={txnStyles.closeTxt}>Done</Text>
            </TouchableOpacity>
          </View>
        </View>

        <View style={txnStyles.balanceRow}>
          <View style={txnStyles.balanceItem}>
            <Text style={txnStyles.balanceLabel}>Current</Text>
            <Text style={txnStyles.balanceValue}>
              {account.current_balance != null ? fmt(account.current_balance, account.currency_code) : '—'}
            </Text>
          </View>
          {account.available_balance != null && (
            <View style={txnStyles.balanceItem}>
              <Text style={txnStyles.balanceLabel}>Available</Text>
              <Text style={txnStyles.balanceValue}>
                {fmt(account.available_balance, account.currency_code)}
              </Text>
            </View>
          )}
        </View>

        {loading ? (
          <View style={txnStyles.center}>
            <ActivityIndicator color="#2dd4a7" />
          </View>
        ) : transactions.length === 0 ? (
          <View style={txnStyles.center}>
            <Text style={txnStyles.empty}>No transactions yet</Text>
          </View>
        ) : (
          <FlatList
            data={transactions}
            keyExtractor={(t) => t.id}
            renderItem={({ item }) => <TransactionRow txn={item} />}
            ItemSeparatorComponent={() => <View style={txnStyles.sep} />}
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
  onPress,
}: {
  account: Account;
  onPress: () => void;
}) => {
  const debt = isDebt(account.type);
  const balance = account.current_balance ?? 0;
  return (
    <TouchableOpacity style={styles.card} onPress={onPress} activeOpacity={0.7}>
      <View style={styles.cardLeft}>
        <View style={[styles.dot, { backgroundColor: debt ? '#f97316' : '#2dd4a7' }]} />
        <View style={{ flex: 1 }}>
          <Text style={styles.cardName} numberOfLines={1}>{account.name}</Text>
          <Text style={styles.cardSub} numberOfLines={1}>
            {account.subtype || account.type}{account.mask ? ` ••${account.mask}` : ''}
          </Text>
        </View>
      </View>
      <View style={styles.cardRight}>
        <Text style={[styles.cardBalance, debt && styles.cardDebt]}>
          {debt ? '-' : ''}{fmt(balance, account.currency_code)}
        </Text>
        {account.available_balance != null && !debt && (
          <Text style={styles.cardAvail}>{fmt(account.available_balance)} avail</Text>
        )}
      </View>
    </TouchableOpacity>
  );
};

// ─── Section Header ───────────────────────────────────────────────────────────

const SectionHeader = ({ title, count }: { title: string; count: number }) =>
  count === 0 ? null : (
    <Text style={styles.sectionLabel}>{title}</Text>
  );

// ─── Main Screen ──────────────────────────────────────────────────────────────

export const AccountsScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [showPlaid, setShowPlaid] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<Account | null>(null);

  const loadAccounts = useCallback(async () => {
    try {
      const data = await plaidService.getAccounts();
      setAccounts(data);
    } catch (e: any) {
      console.error('load accounts:', e);
    }
  }, []);

  useEffect(() => {
    loadAccounts().finally(() => setLoading(false));
  }, [loadAccounts]);

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
    } finally {
      setSyncing(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await plaidService.syncTransactions();
      await loadAccounts();
      Alert.alert('Synced', `${result.transactions_added ?? 0} new transactions added.`);
    } catch (e: any) {
      Alert.alert('Error', e.message || 'Sync failed.');
    } finally {
      setSyncing(false);
    }
  };

  const totalBalance = accounts
    .filter((a) => isBalance(a.type))
    .reduce((sum, a) => sum + (a.current_balance ?? 0), 0);

  const totalDebt = accounts
    .filter((a) => isDebt(a.type))
    .reduce((sum, a) => sum + Math.abs(a.current_balance ?? 0), 0);

  const banks = groupByBank(accounts);

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2dd4a7" />
      </View>
    );
  }

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      {/* Header */}
      <LinearGradient
        colors={['#2dd4a7', '#1e3a5f']}
        start={{ x: 0, y: 0 }}
        end={{ x: 1, y: 1 }}
        style={styles.header}
      >
        <View style={styles.totalsRow}>
          <View style={styles.totalItem}>
            <Text style={styles.totalLabel}>Balance</Text>
            <Text style={styles.totalValue}>{fmt(totalBalance)}</Text>
          </View>
          <View style={styles.totalDivider} />
          <View style={styles.totalItem}>
            <Text style={styles.totalLabel}>Debt</Text>
            <Text style={[styles.totalValue, styles.debtValue]}>{fmt(totalDebt)}</Text>
          </View>
        </View>

        <View style={styles.headerActions}>
          <TouchableOpacity style={styles.headerBtn} onPress={() => setShowPlaid(true)} disabled={syncing}>
            <Text style={styles.headerBtnText}>+ Connect Bank</Text>
          </TouchableOpacity>
          {accounts.length > 0 && (
            <TouchableOpacity style={[styles.headerBtn, styles.headerBtnOutline]} onPress={handleSync} disabled={syncing}>
              {syncing
                ? <ActivityIndicator size="small" color="#fff" />
                : <Text style={styles.headerBtnText}>Sync</Text>}
            </TouchableOpacity>
          )}
        </View>
      </LinearGradient>

      {/* Account List */}
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor="#2dd4a7" />}
      >
        {accounts.length === 0 ? (
          <View style={styles.empty}>
            <Text style={styles.emptyIcon}>🏦</Text>
            <Text style={styles.emptyTitle}>No accounts yet</Text>
            <Text style={styles.emptySubtitle}>Connect your first bank account to get started</Text>
            <TouchableOpacity style={styles.connectBtn} onPress={() => setShowPlaid(true)}>
              <Text style={styles.connectBtnText}>Connect Bank</Text>
            </TouchableOpacity>
          </View>
        ) : (
          banks.map((bank) => (
            <View key={bank.plaid_item_id} style={styles.bankSection}>
              <Text style={styles.bankName}>{bank.institution_name}</Text>

              <SectionHeader title="Checking & Savings" count={bank.depository.length} />
              {bank.depository.map((a) => (
                <AccountCard key={a.id} account={a} onPress={() => setSelectedAccount(a)} />
              ))}

              <SectionHeader title="Credit Cards & Loans" count={bank.credit.length} />
              {bank.credit.map((a) => (
                <AccountCard key={a.id} account={a} onPress={() => setSelectedAccount(a)} />
              ))}

              <SectionHeader title="Other Accounts" count={bank.others.length} />
              {bank.others.map((a) => (
                <AccountCard key={a.id} account={a} onPress={() => setSelectedAccount(a)} />
              ))}
            </View>
          ))
        )}
      </ScrollView>

      <PlaidLinkModal
        visible={showPlaid}
        onSuccess={handlePlaidSuccess}
        onClose={() => setShowPlaid(false)}
      />

      {selectedAccount && (
        <AccountTransactionsModal
          account={selectedAccount}
          onClose={() => setSelectedAccount(null)}
        />
      )}
    </View>
  );
};

// ─── Styles ───────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#f1f5f9' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },

  header: { paddingHorizontal: 20, paddingTop: 16, paddingBottom: 20 },
  totalsRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 16 },
  totalItem: { flex: 1, alignItems: 'center' },
  totalDivider: { width: 1, height: 40, backgroundColor: 'rgba(255,255,255,0.3)' },
  totalLabel: { color: 'rgba(255,255,255,0.75)', fontSize: 12, marginBottom: 4, textTransform: 'uppercase', letterSpacing: 0.5 },
  totalValue: { color: '#fff', fontSize: 26, fontWeight: '700' },
  debtValue: { color: '#fca5a5' },
  headerActions: { flexDirection: 'row', gap: 10 },
  headerBtn: { backgroundColor: 'rgba(255,255,255,0.25)', borderRadius: 20, paddingHorizontal: 16, paddingVertical: 8 },
  headerBtnOutline: { backgroundColor: 'rgba(255,255,255,0.15)' },
  headerBtnText: { color: '#fff', fontWeight: '600', fontSize: 14 },

  scroll: { flex: 1 },
  scrollContent: { padding: 16, paddingBottom: 32, gap: 20 },

  bankSection: { gap: 6 },
  bankName: { fontSize: 16, fontWeight: '700', color: '#0f172a', marginBottom: 4, paddingLeft: 2 },
  sectionLabel: { fontSize: 11, fontWeight: '600', color: '#94a3b8', textTransform: 'uppercase', letterSpacing: 0.8, marginTop: 8, marginBottom: 4, paddingLeft: 2 },

  card: {
    backgroundColor: '#fff',
    borderRadius: 14,
    padding: 14,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 6,
    elevation: 2,
  },
  cardLeft: { flexDirection: 'row', alignItems: 'center', gap: 10, flex: 1 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  cardName: { fontSize: 14, fontWeight: '600', color: '#1e3a5f' },
  cardSub: { fontSize: 12, color: '#64748b', marginTop: 1, textTransform: 'capitalize' },
  cardRight: { alignItems: 'flex-end' },
  cardBalance: { fontSize: 15, fontWeight: '700', color: '#1e3a5f' },
  cardDebt: { color: '#ef4444' },
  cardAvail: { fontSize: 11, color: '#64748b', marginTop: 2 },

  empty: { alignItems: 'center', paddingTop: 60, paddingHorizontal: 32 },
  emptyIcon: { fontSize: 48, marginBottom: 12 },
  emptyTitle: { fontSize: 20, fontWeight: '700', color: '#1e3a5f', marginBottom: 8 },
  emptySubtitle: { fontSize: 14, color: '#64748b', textAlign: 'center', marginBottom: 24 },
  connectBtn: { backgroundColor: '#2dd4a7', borderRadius: 12, paddingHorizontal: 28, paddingVertical: 14 },
  connectBtnText: { color: '#fff', fontWeight: '700', fontSize: 16 },
});

const txnStyles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#fff' },
  dragArea: {
    paddingTop: 10,
    borderBottomWidth: 1,
    borderBottomColor: '#f1f5f9',
  },
  dragHandle: {
    width: 36,
    height: 4,
    borderRadius: 2,
    backgroundColor: '#cbd5e1',
    alignSelf: 'center',
    marginBottom: 8,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    paddingHorizontal: 16,
    paddingBottom: 14,
  },
  headerTitle: { fontSize: 17, fontWeight: '700', color: '#1e3a5f' },
  headerSub: { fontSize: 13, color: '#64748b', marginTop: 2, textTransform: 'capitalize' },
  closeBtn: { paddingLeft: 12 },
  closeTxt: { fontSize: 16, color: '#2dd4a7', fontWeight: '600' },

  balanceRow: {
    flexDirection: 'row',
    backgroundColor: '#f8fafc',
    paddingVertical: 14,
    paddingHorizontal: 20,
    gap: 32,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },
  balanceItem: {},
  balanceLabel: { fontSize: 11, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  balanceValue: { fontSize: 20, fontWeight: '700', color: '#1e3a5f' },

  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  empty: { color: '#94a3b8', fontSize: 15 },

  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  left: { flex: 1, paddingRight: 12 },
  name: { fontSize: 14, fontWeight: '500', color: '#1e3a5f' },
  meta: { fontSize: 12, color: '#94a3b8', marginTop: 2, textTransform: 'capitalize' },
  amount: { fontSize: 14, fontWeight: '600', color: '#ef4444' },
  positive: { color: '#16a34a' },
  sep: { height: 1, backgroundColor: '#f1f5f9', marginHorizontal: 16 },
});
