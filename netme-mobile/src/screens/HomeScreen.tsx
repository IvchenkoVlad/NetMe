import React, { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { budgetService, BudgetSummary } from '../services/budgetService';
import { plaidService } from '../services/plaidService';

interface Account {
  id: string;
  name: string;
  current_balance?: number;
}

interface Transaction {
  id: string;
  name: string;
  merchant_name?: string;
  amount: number;
  currency_code: string;
  date: string;
  pending: boolean;
}

const currentMonth = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

const fmt = (n: number) =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n);

const GLASS = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
} as const;

export const HomeScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [summary, setSummary] = useState<BudgetSummary | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);

  const load = useCallback(async () => {
    try {
      const [s, accts, txns] = await Promise.all([
        budgetService.getSummary(currentMonth()),
        plaidService.getAccounts(),
        plaidService.getTransactions(5),
      ]);
      setSummary(s);
      setAccounts(accts);
      setTransactions(txns);
    } catch (e: any) {
      console.error('home load:', e.message);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  }, [load]);

  const net = (summary?.income ?? 0) - (summary?.spending ?? 0);
  const totalBalance = accounts.reduce((sum, a) => sum + (a.current_balance ?? 0), 0);
  const overBudget = summary?.categories.filter(
    c => !c.is_income && c.budget_limit > 0 && c.spent > c.budget_limit
  ) ?? [];

  return (
    <View style={[s.container, { paddingTop: insets.top }]}>
      {loading ? (
        <View style={s.center}><ActivityIndicator color="#2dd4a7" /></View>
      ) : (
        <ScrollView
          style={s.scroll}
          contentContainerStyle={s.content}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor="#2dd4a7" />}
        >
          {/* Net Position */}
          <View style={s.card}>
            <Text style={s.cardTitle}>This month</Text>
            <View style={s.netRow}>
              <View style={s.netItem}>
                <Text style={s.netLabel}>Income</Text>
                <Text style={[s.netValue, s.incomeColor]}>{fmt(summary?.income ?? 0)}</Text>
              </View>
              <View style={s.netDivider} />
              <View style={s.netItem}>
                <Text style={s.netLabel}>Spending</Text>
                <Text style={s.netValue}>{fmt(summary?.spending ?? 0)}</Text>
              </View>
            </View>
            <View style={s.savedRow}>
              <Text style={s.savedLabel}>{net >= 0 ? 'Saved' : 'Over by'}</Text>
              <Text style={[s.savedValue, net < 0 && s.overColor]}>{fmt(Math.abs(net))}</Text>
            </View>
          </View>

          {/* Total Balance */}
          <View style={s.card}>
            <Text style={s.cardTitle}>Total balance</Text>
            {accounts.length === 0 ? (
              <Text style={s.emptyText}>Connect an account to see your balance</Text>
            ) : (
              <Text style={s.balanceValue}>{fmt(totalBalance)}</Text>
            )}
          </View>

          {/* Over-budget alerts */}
          {overBudget.length > 0 && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Over budget</Text>
              {overBudget.map(cat => (
                <View key={cat.id} style={s.alertRow}>
                  <Text style={s.alertIcon}>{cat.icon}</Text>
                  <Text style={s.alertName}>{cat.name}</Text>
                  <Text style={s.alertOver}>over by {fmt(cat.spent - cat.budget_limit)}</Text>
                </View>
              ))}
            </View>
          )}

          {/* Recent Transactions */}
          <View style={s.card}>
            <Text style={s.cardTitle}>Recent transactions</Text>
            {transactions.length === 0 ? (
              <Text style={s.emptyText}>No transactions yet</Text>
            ) : (
              transactions.map(txn => (
                <TouchableOpacity
                  key={txn.id}
                  style={s.txnRow}
                  onPress={() => navigation.navigate('TransactionDetail', { transactionId: txn.id })}
                  activeOpacity={0.7}
                >
                  <View style={s.txnLeft}>
                    <Text style={s.txnName} numberOfLines={1}>{txn.merchant_name || txn.name}</Text>
                    <Text style={s.txnDate}>{txn.date}</Text>
                  </View>
                  <Text style={[s.txnAmount, txn.amount < 0 && s.incomeColor]}>
                    {txn.amount < 0 ? '+' : ''}{fmt(Math.abs(txn.amount))}
                  </Text>
                </TouchableOpacity>
              ))
            )}
          </View>
        </ScrollView>
      )}
    </View>
  );
};

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'transparent' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  scroll: { flex: 1 },
  content: { padding: 16, gap: 14, paddingBottom: 40 },

  card: { ...GLASS, padding: 16 },
  cardTitle: {
    fontSize: 12, fontWeight: '700', color: 'rgba(255,255,255,0.4)',
    textTransform: 'uppercase', letterSpacing: 0.6, marginBottom: 12,
  },

  netRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 12 },
  netItem: { flex: 1, alignItems: 'center' },
  netDivider: { width: 1, height: 36, backgroundColor: 'rgba(255,255,255,0.15)' },
  netLabel: {
    fontSize: 11, color: 'rgba(255,255,255,0.5)',
    textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2,
  },
  netValue: { fontSize: 22, fontWeight: '700', color: '#fff' },
  savedRow: {
    flexDirection: 'row', justifyContent: 'center', alignItems: 'center', gap: 6,
    paddingTop: 10, borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: 'rgba(255,255,255,0.1)',
  },
  savedLabel: { fontSize: 13, color: 'rgba(255,255,255,0.5)' },
  savedValue: { fontSize: 15, fontWeight: '700', color: '#4ade80' },

  balanceValue: { fontSize: 32, fontWeight: '700', color: '#fff' },

  alertRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: 6, gap: 10 },
  alertIcon: { fontSize: 20 },
  alertName: { flex: 1, fontSize: 14, fontWeight: '500', color: '#fff' },
  alertOver: { fontSize: 13, color: '#fca5a5', fontWeight: '600' },

  txnRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: 8, gap: 12 },
  txnLeft: { flex: 1 },
  txnName: { fontSize: 14, fontWeight: '500', color: '#fff' },
  txnDate: { fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 },
  txnAmount: { fontSize: 14, fontWeight: '700', color: '#fff' },

  incomeColor: { color: '#4ade80' },
  overColor: { color: '#fca5a5' },
  emptyText: { fontSize: 14, color: 'rgba(255,255,255,0.4)', textAlign: 'center', paddingVertical: 8 },
});
