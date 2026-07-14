import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { plaidService } from '../services/plaidService';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Transaction {
  id: string;
  name: string;
  merchant_name?: string;
  amount: number;
  currency_code: string;
  date: string;
  pending: boolean;
  category?: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const PAGE_SIZE = 50;

const currentMonth = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

const addMonths = (ym: string, delta: number) => {
  const [y, m] = ym.split('-').map(Number);
  const d = new Date(y, m - 1 + delta);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
};

const monthLabel = (ym: string) => {
  const [y, m] = ym.split('-').map(Number);
  return new Date(y, m - 1).toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
};

const fmt = (amount: number, currency = 'USD') =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(Math.abs(amount));

const fmtDate = (dateStr: string) => {
  const [y, m, d] = dateStr.split('T')[0].split('-').map(Number);
  return new Date(y, m - 1, d).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
};

const GLASS = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
} as const;

// ─── Transaction Row ──────────────────────────────────────────────────────────

const TxnRow = ({ txn, onPress }: { txn: Transaction; onPress: () => void }) => {
  const isIncome = txn.amount < 0;
  const label = txn.merchant_name || txn.name;
  const categoryLabel = txn.category
    ? txn.category.replace(/_/g, ' ').toLowerCase()
    : null;

  return (
    <TouchableOpacity style={r.row} onPress={onPress} activeOpacity={0.7}>
      <View style={r.left}>
        <Text style={r.name} numberOfLines={1}>{label}</Text>
        <Text style={r.meta}>
          {fmtDate(txn.date)}
          {categoryLabel ? ` · ${categoryLabel}` : ''}
          {txn.pending ? ' · pending' : ''}
        </Text>
      </View>
      <Text style={[r.amount, isIncome && r.income]}>
        {isIncome ? '+' : '-'}{fmt(txn.amount, txn.currency_code)}
      </Text>
    </TouchableOpacity>
  );
};

const r = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    paddingHorizontal: 16,
  },
  left: { flex: 1, paddingRight: 12 },
  name: { fontSize: 14, fontWeight: '500', color: '#fff' },
  meta: { fontSize: 12, color: 'rgba(255,255,255,0.4)', marginTop: 2, textTransform: 'capitalize' },
  amount: { fontSize: 14, fontWeight: '700', color: '#fca5a5' },
  income: { color: '#4ade80' },
});

// ─── Separator ────────────────────────────────────────────────────────────────

const Separator = () => (
  <View style={{ height: StyleSheet.hairlineWidth, backgroundColor: 'rgba(255,255,255,0.07)', marginHorizontal: 16 }} />
);

// ─── Main Screen ──────────────────────────────────────────────────────────────

export const TransactionsScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();

  const [month, setMonth] = useState(currentMonth());
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const offsetRef = useRef(0);

  const fetchPage = useCallback(async (m: string, offset: number, replace: boolean) => {
    const page = await plaidService.getTransactions(PAGE_SIZE, offset, '', m);
    if (replace) {
      setTxns(page);
    } else {
      setTxns(prev => [...prev, ...page]);
    }
    setHasMore(page.length === PAGE_SIZE);
    offsetRef.current = offset + page.length;
  }, []);

  const reload = useCallback(async (m: string) => {
    offsetRef.current = 0;
    await fetchPage(m, 0, true);
  }, [fetchPage]);

  useEffect(() => {
    setLoading(true);
    setHasMore(true);
    reload(month).finally(() => setLoading(false));
  }, [month, reload]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await reload(month);
    setRefreshing(false);
  }, [month, reload]);

  const onEndReached = useCallback(async () => {
    if (loadingMore || !hasMore) return;
    setLoadingMore(true);
    await fetchPage(month, offsetRef.current, false);
    setLoadingMore(false);
  }, [loadingMore, hasMore, month, fetchPage]);

  const changeMonth = (delta: number) => {
    setTxns([]);
    setMonth(m => addMonths(m, delta));
  };

  return (
    <View style={[s.container, { paddingTop: insets.top }]}>
      {/* Month selector */}
      <View style={s.monthBar}>
        <TouchableOpacity onPress={() => changeMonth(-1)} style={s.arrow}>
          <Text style={s.arrowTxt}>‹</Text>
        </TouchableOpacity>
        <Text style={s.monthLabel}>{monthLabel(month)}</Text>
        <TouchableOpacity
          onPress={() => changeMonth(1)}
          style={s.arrow}
          disabled={month >= currentMonth()}
        >
          <Text style={[s.arrowTxt, month >= currentMonth() && s.arrowDisabled]}>›</Text>
        </TouchableOpacity>
      </View>

      {loading ? (
        <View style={s.center}><ActivityIndicator color="#2dd4a7" /></View>
      ) : (
        <FlatList
          data={txns}
          keyExtractor={item => item.id}
          renderItem={({ item }) => (
            <TxnRow
              txn={item}
              onPress={() => navigation.navigate('TransactionDetail', { transactionId: item.id })}
            />
          )}
          ItemSeparatorComponent={Separator}
          contentContainerStyle={txns.length === 0 ? s.emptyContainer : s.listContent}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor="#2dd4a7" />
          }
          onEndReached={onEndReached}
          onEndReachedThreshold={0.3}
          ListEmptyComponent={
            <View style={s.empty}>
              <Text style={s.emptyIcon}>🧾</Text>
              <Text style={s.emptyTitle}>No transactions</Text>
              <Text style={s.emptySub}>Nothing recorded for {monthLabel(month)}</Text>
            </View>
          }
          ListFooterComponent={
            loadingMore ? (
              <View style={s.footer}><ActivityIndicator color="#2dd4a7" size="small" /></View>
            ) : null
          }
          style={s.list}
        />
      )}
    </View>
  );
};

// ─── Styles ───────────────────────────────────────────────────────────────────

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'transparent' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },

  monthBar: {
    ...GLASS,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginHorizontal: 16,
    marginBottom: 8,
    paddingVertical: 4,
    paddingHorizontal: 8,
  },
  arrow: { padding: 8 },
  arrowTxt: { fontSize: 28, color: '#fff', fontWeight: '300' },
  arrowDisabled: { opacity: 0.25 },
  monthLabel: { fontSize: 16, fontWeight: '700', color: '#fff' },

  list: { flex: 1 },
  listContent: {
    ...GLASS,
    marginHorizontal: 16,
    marginBottom: 40,
    overflow: 'hidden',
  },
  emptyContainer: { flex: 1 },

  empty: { flex: 1, alignItems: 'center', paddingTop: 80 },
  emptyIcon: { fontSize: 44, marginBottom: 12 },
  emptyTitle: { fontSize: 18, fontWeight: '700', color: '#fff', marginBottom: 6 },
  emptySub: { fontSize: 14, color: 'rgba(255,255,255,0.45)' },

  footer: { paddingVertical: 20, alignItems: 'center' },
});
