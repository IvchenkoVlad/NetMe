import React, { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
  Modal,
  TextInput,
  Alert,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Rect, Text as SvgText } from 'react-native-svg';
import { budgetService, BudgetSummary, CategorySummary, MonthlyTotal } from '../services/budgetService';

// ─── Helpers ──────────────────────────────────────────────────────────────────

const fmt = (n: number) =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n);

const monthLabel = (ym: string) => {
  const [y, m] = ym.split('-').map(Number);
  return new Date(y, m - 1).toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
};

const shortMonth = (ym: string) => {
  const [y, m] = ym.split('-').map(Number);
  return new Date(y, m - 1).toLocaleDateString('en-US', { month: 'short' });
};

const addMonths = (ym: string, delta: number) => {
  const [y, m] = ym.split('-').map(Number);
  const d = new Date(y, m - 1 + delta);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
};

const currentMonth = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

// ─── Bar Chart ────────────────────────────────────────────────────────────────

const BarChart = ({ data }: { data: MonthlyTotal[] }) => {
  if (!data.length) return null;
  const W = 320, H = 100;
  const maxVal = Math.max(...data.map(d => d.spending), 1);
  const barW = Math.floor((W - (data.length - 1) * 6) / data.length);
  return (
    <View style={{ alignItems: 'center', marginVertical: 8 }}>
      <Svg width={W} height={H + 20}>
        {data.map((d, i) => {
          const h = Math.max(4, (d.spending / maxVal) * H);
          const x = i * (barW + 6);
          return (
            <React.Fragment key={d.month}>
              <Rect x={x} y={H - h} width={barW} height={h} rx={4} fill="#2dd4a7" opacity={0.85} />
              <SvgText x={x + barW / 2} y={H + 14} fontSize={9} fill="rgba(255,255,255,0.4)" textAnchor="middle">
                {shortMonth(d.month)}
              </SvgText>
            </React.Fragment>
          );
        })}
      </Svg>
    </View>
  );
};

// ─── Progress Bar ─────────────────────────────────────────────────────────────

const ProgressBar = ({ spent, limit, color }: { spent: number; limit: number; color: string }) => {
  const pct = limit > 0 ? Math.min(spent / limit, 1) : 0;
  const over = limit > 0 && spent > limit;
  return (
    <View style={pb.track}>
      <View style={[pb.fill, { width: `${pct * 100}%`, backgroundColor: over ? '#ef4444' : color }]} />
    </View>
  );
};
const pb = StyleSheet.create({
  track: { height: 5, backgroundColor: 'rgba(255,255,255,0.1)', borderRadius: 3, overflow: 'hidden', marginTop: 6 },
  fill: { height: '100%', borderRadius: 3 },
});

// ─── Budget Edit Modal ────────────────────────────────────────────────────────

const EditBudgetModal: React.FC<{
  category: CategorySummary | null;
  month: string;
  onSave: (amount: number) => void;
  onClose: () => void;
}> = ({ category, month, onSave, onClose }) => {
  const [value, setValue] = useState('');
  useEffect(() => {
    if (category) setValue(category.budget_limit > 0 ? String(category.budget_limit) : '');
  }, [category]);

  if (!category) return null;
  return (
    <Modal visible animationType="fade" transparent>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={em.overlay}>
        <View style={em.card}>
          <Text style={em.icon}>{category.icon}</Text>
          <Text style={em.title}>{category.name}</Text>
          <Text style={em.sub}>Monthly budget for {monthLabel(month)}</Text>
          <TextInput
            style={em.input}
            value={value}
            onChangeText={setValue}
            keyboardType="decimal-pad"
            placeholder="0"
            placeholderTextColor="#94a3b8"
            autoFocus
          />
          <View style={em.row}>
            <TouchableOpacity style={em.cancel} onPress={onClose}>
              <Text style={em.cancelTxt}>Cancel</Text>
            </TouchableOpacity>
            <TouchableOpacity style={em.save} onPress={() => {
              const n = parseFloat(value);
              if (isNaN(n) || n < 0) { Alert.alert('Invalid amount'); return; }
              onSave(n);
            }}>
              <Text style={em.saveTxt}>Save</Text>
            </TouchableOpacity>
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
};

const em = StyleSheet.create({
  overlay: { flex: 1, backgroundColor: 'rgba(0,0,0,0.6)', justifyContent: 'center', alignItems: 'center' },
  card: { backgroundColor: '#1e293b', borderRadius: 20, padding: 24, width: 300, alignItems: 'center', borderWidth: 1, borderColor: 'rgba(255,255,255,0.1)' },
  icon: { fontSize: 36, marginBottom: 8 },
  title: { fontSize: 18, fontWeight: '700', color: '#fff', marginBottom: 4 },
  sub: { fontSize: 13, color: 'rgba(255,255,255,0.5)', marginBottom: 16 },
  input: { width: '100%', borderWidth: 1, borderColor: 'rgba(255,255,255,0.15)', borderRadius: 10, padding: 12, fontSize: 24, fontWeight: '700', color: '#fff', textAlign: 'center', marginBottom: 16, backgroundColor: 'rgba(255,255,255,0.06)' },
  row: { flexDirection: 'row', gap: 12, width: '100%' },
  cancel: { flex: 1, padding: 12, borderRadius: 10, borderWidth: 1, borderColor: 'rgba(255,255,255,0.12)', alignItems: 'center' },
  cancelTxt: { color: 'rgba(255,255,255,0.6)', fontWeight: '600' },
  save: { flex: 1, padding: 12, borderRadius: 10, backgroundColor: '#2dd4a7', alignItems: 'center' },
  saveTxt: { color: '#0f172a', fontWeight: '700' },
});

// ─── Category Row ─────────────────────────────────────────────────────────────

const CategoryRow = ({ cat, onPress }: { cat: CategorySummary; onPress: () => void }) => {
  const over = cat.budget_limit > 0 && cat.spent > cat.budget_limit;
  return (
    <TouchableOpacity style={cr.row} onPress={onPress} activeOpacity={0.75}>
      <View style={[cr.iconBox, { backgroundColor: cat.color + '25' }]}>
        <Text style={cr.icon}>{cat.icon}</Text>
      </View>
      <View style={cr.mid}>
        <View style={cr.topRow}>
          <Text style={cr.name}>{cat.name}</Text>
          <Text style={[cr.spent, over && cr.over]}>{fmt(cat.spent)}</Text>
        </View>
        {cat.budget_limit > 0 ? (
          <>
            <ProgressBar spent={cat.spent} limit={cat.budget_limit} color={cat.color} />
            <Text style={cr.limit}>{fmt(cat.spent)} of {fmt(cat.budget_limit)}</Text>
          </>
        ) : (
          <Text style={cr.nobudget}>Tap to set budget</Text>
        )}
      </View>
    </TouchableOpacity>
  );
};

const cr = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 10 },
  iconBox: { width: 40, height: 40, borderRadius: 12, justifyContent: 'center', alignItems: 'center' },
  icon: { fontSize: 20 },
  mid: { flex: 1 },
  topRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  name: { fontSize: 14, fontWeight: '600', color: '#fff' },
  spent: { fontSize: 14, fontWeight: '700', color: '#fff' },
  over: { color: '#fca5a5' },
  limit: { fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 3 },
  nobudget: { fontSize: 11, color: 'rgba(255,255,255,0.35)', marginTop: 4 },
});

// ─── Main Screen ──────────────────────────────────────────────────────────────

export const BudgetScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
  const [month, setMonth] = useState(currentMonth());
  const [summary, setSummary] = useState<BudgetSummary | null>(null);
  const [history, setHistory] = useState<MonthlyTotal[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [editing, setEditing] = useState<CategorySummary | null>(null);

  const load = useCallback(async (m: string) => {
    try {
      const [s, h] = await Promise.all([budgetService.getSummary(m), budgetService.getHistory(6)]);
      setSummary(s);
      setHistory(h);
    } catch (e: any) { console.error('budget load:', e.message); }
  }, []);

  useEffect(() => { setLoading(true); load(month).finally(() => setLoading(false)); }, [month, load]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await load(month);
    setRefreshing(false);
  }, [month, load]);

  const handleSaveBudget = async (amount: number) => {
    if (!editing) return;
    try {
      await budgetService.setBudget(editing.id, month, amount);
      setEditing(null);
      await load(month);
    } catch (e: any) { Alert.alert('Error', e.message); }
  };

  const allSpending = summary?.categories.filter(c => !c.is_income) ?? [];
  const incomeCats  = summary?.categories.filter(c => c.is_income && c.spent > 0) ?? [];

  return (
    <View style={[s.container, { paddingTop: insets.top }]}>
      {loading ? (
        <View style={s.center}><ActivityIndicator color="#2dd4a7" /></View>
      ) : (
        <ScrollView
          style={s.scroll}
          contentContainerStyle={s.scrollContent}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor="#2dd4a7" />}
        >
          {/* Hero — month nav + totals */}
          <View style={s.hero}>
            <View style={s.monthRow}>
              <TouchableOpacity onPress={() => setMonth(m => addMonths(m, -1))} style={s.arrow}>
                <Text style={s.arrowTxt}>‹</Text>
              </TouchableOpacity>
              <Text style={s.monthLabel}>{monthLabel(month)}</Text>
              <TouchableOpacity onPress={() => setMonth(m => addMonths(m, 1))} style={s.arrow} disabled={month >= currentMonth()}>
                <Text style={[s.arrowTxt, month >= currentMonth() && s.arrowDisabled]}>›</Text>
              </TouchableOpacity>
            </View>
            <View style={s.totalsRow}>
              <View style={s.total}>
                <Text style={s.totalLabel}>Income</Text>
                <Text style={[s.totalValue, s.incomeVal]}>{fmt(summary?.income ?? 0)}</Text>
              </View>
              <View style={s.totalDivider} />
              <View style={s.total}>
                <Text style={s.totalLabel}>Spending</Text>
                <Text style={s.totalValue}>{fmt(summary?.spending ?? 0)}</Text>
              </View>
            </View>
          </View>

          {/* Bar chart */}
          {history.length > 1 && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Monthly Spending</Text>
              <BarChart data={history} />
            </View>
          )}

          {/* Spending categories */}
          {allSpending.length > 0 && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Spending by Category</Text>
              {allSpending.map(cat => <CategoryRow key={cat.id} cat={cat} onPress={() => setEditing(cat)} />)}
            </View>
          )}

          {/* Income */}
          {incomeCats.length > 0 && (
            <View style={s.card}>
              <Text style={s.cardTitle}>Income</Text>
              {incomeCats.map(cat => (
                <View key={cat.id} style={cr.row}>
                  <View style={[cr.iconBox, { backgroundColor: cat.color + '25' }]}>
                    <Text style={cr.icon}>{cat.icon}</Text>
                  </View>
                  <View style={cr.mid}>
                    <View style={cr.topRow}>
                      <Text style={cr.name}>{cat.name}</Text>
                      <Text style={[cr.spent, { color: '#4ade80' }]}>{fmt(cat.spent)}</Text>
                    </View>
                  </View>
                </View>
              ))}
            </View>
          )}

          {!summary || (allSpending.length === 0 && incomeCats.length === 0) ? (
            <View style={s.empty}>
              <Text style={s.emptyIcon}>📊</Text>
              <Text style={s.emptyTitle}>No data for this month</Text>
              <Text style={s.emptySub}>Sync your accounts to see spending</Text>
            </View>
          ) : null}
        </ScrollView>
      )}

      <EditBudgetModal category={editing} month={month} onSave={handleSaveBudget} onClose={() => setEditing(null)} />
    </View>
  );
};

// ─── Styles ───────────────────────────────────────────────────────────────────

const GLASS = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
} as const;

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'transparent' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  scroll: { flex: 1 },
  scrollContent: { padding: 16, gap: 14, paddingBottom: 40 },

  // Hero
  hero: { ...GLASS, padding: 20 },
  monthRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 },
  arrow: { padding: 8 },
  arrowTxt: { fontSize: 28, color: '#fff', fontWeight: '300' },
  arrowDisabled: { opacity: 0.25 },
  monthLabel: { fontSize: 17, fontWeight: '700', color: '#fff' },
  totalsRow: { flexDirection: 'row', alignItems: 'center' },
  total: { flex: 1, alignItems: 'center' },
  totalDivider: { width: 1, height: 36, backgroundColor: 'rgba(255,255,255,0.15)' },
  totalLabel: { fontSize: 11, color: 'rgba(255,255,255,0.5)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  totalValue: { fontSize: 22, fontWeight: '700', color: '#fff' },
  incomeVal: { color: '#4ade80' },

  // Content cards
  card: { ...GLASS, padding: 16 },
  cardTitle: { fontSize: 12, fontWeight: '700', color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: 0.6, marginBottom: 12 },

  // Empty state
  empty: { alignItems: 'center', paddingTop: 48 },
  emptyIcon: { fontSize: 44, marginBottom: 12 },
  emptyTitle: { fontSize: 18, fontWeight: '700', color: '#fff', marginBottom: 6 },
  emptySub: { fontSize: 14, color: 'rgba(255,255,255,0.45)' },
});
