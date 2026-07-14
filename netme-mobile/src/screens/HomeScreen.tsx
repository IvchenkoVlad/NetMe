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
import { analyticsService, AnalyticsOverview } from '../services/analyticsService';
import { Transaction } from '../services/transactionService';
import { fmt, fmtDate, currentMonth } from '../utils/format';
import { GLASS } from '../styles/theme';

export const HomeScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [summary, setSummary] = useState<BudgetSummary | null>(null);
  const [analytics, setAnalytics] = useState<AnalyticsOverview | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);

  const load = useCallback(async () => {
    try {
      const [s, a, txns] = await Promise.all([
        budgetService.getSummary(currentMonth()),
        analyticsService.getOverview(),
        plaidService.getTransactions(5),
      ]);
      setSummary(s);
      setAnalytics(a);
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
  const overBudget = summary?.categories.filter(
    c => !c.is_income && c.budget_limit > 0 && c.spent > c.budget_limit
  ) ?? [];

  const nw = analytics?.net_worth;
  const topCats = analytics?.top_categories ?? [];

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
          {/* Net Worth */}
          {nw != null && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Net worth</Text>
              <Text style={[s.netWorthValue, nw.net_worth < 0 && s.overColor]}>
                {fmt(nw.net_worth)}
              </Text>
              <View style={s.nwRow}>
                <View style={s.nwItem}>
                  <Text style={s.nwLabel}>Assets</Text>
                  <Text style={[s.nwValue, s.incomeColor]}>{fmt(nw.assets)}</Text>
                </View>
                <View style={s.nwDivider} />
                <View style={s.nwItem}>
                  <Text style={s.nwLabel}>Liabilities</Text>
                  <Text style={[s.nwValue, s.overColor]}>{fmt(nw.liabilities)}</Text>
                </View>
              </View>
            </View>
          )}

          {/* This Month */}
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

          {/* Top Spending Categories */}
          {topCats.length > 0 && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Top spending</Text>
              {topCats.map(cat => (
                <View key={cat.category_id} style={s.catRow}>
                  <View style={[s.catIcon, { backgroundColor: cat.color + '25' }]}>
                    <Text style={s.catEmoji}>{cat.icon}</Text>
                  </View>
                  <View style={s.catMid}>
                    <View style={s.catLabelRow}>
                      <Text style={s.catName}>{cat.name}</Text>
                      <Text style={s.catSpent}>{fmt(cat.spent)}</Text>
                    </View>
                    <View style={s.barTrack}>
                      <View style={[s.barFill, { width: `${Math.min(cat.pct, 100)}%`, backgroundColor: cat.color }]} />
                    </View>
                  </View>
                </View>
              ))}
            </View>
          )}

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
                    <Text style={s.txnDate}>{fmtDate(txn.date)}</Text>
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

  // Net Worth card
  netWorthValue: { fontSize: 36, fontWeight: '800', color: '#fff', marginBottom: 12 },
  nwRow: { flexDirection: 'row', alignItems: 'center' },
  nwItem: { flex: 1, alignItems: 'center' },
  nwDivider: { width: 1, height: 32, backgroundColor: 'rgba(255,255,255,0.15)' },
  nwLabel: { fontSize: 11, color: 'rgba(255,255,255,0.5)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  nwValue: { fontSize: 16, fontWeight: '700', color: '#fff' },

  // This month card
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

  // Top categories
  catRow: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 6 },
  catIcon: { width: 36, height: 36, borderRadius: 10, justifyContent: 'center', alignItems: 'center' },
  catEmoji: { fontSize: 18 },
  catMid: { flex: 1 },
  catLabelRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 },
  catName: { fontSize: 13, fontWeight: '500', color: '#fff' },
  catSpent: { fontSize: 13, fontWeight: '700', color: '#fff' },
  barTrack: { height: 4, backgroundColor: 'rgba(255,255,255,0.1)', borderRadius: 2, overflow: 'hidden' },
  barFill: { height: '100%', borderRadius: 2 },

  // Over budget
  alertRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: 6, gap: 10 },
  alertIcon: { fontSize: 20 },
  alertName: { flex: 1, fontSize: 14, fontWeight: '500', color: '#fff' },
  alertOver: { fontSize: 13, color: '#fca5a5', fontWeight: '600' },

  // Transactions
  txnRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: 8, gap: 12 },
  txnLeft: { flex: 1 },
  txnName: { fontSize: 14, fontWeight: '500', color: '#fff' },
  txnDate: { fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 },
  txnAmount: { fontSize: 14, fontWeight: '700', color: '#fff' },

  incomeColor: { color: '#4ade80' },
  overColor: { color: '#fca5a5' },
  emptyText: { fontSize: 14, color: 'rgba(255,255,255,0.4)', textAlign: 'center', paddingVertical: 8 },
});
