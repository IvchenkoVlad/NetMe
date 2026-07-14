import React, { useCallback, useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Modal,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { transactionService, Transaction } from '../services/transactionService';
import { budgetService, Category } from '../services/budgetService';

// ─── Constants ────────────────────────────────────────────────────────────────

const GLASS = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
} as const;

const fmt = (amount: number, currency = 'USD') =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(Math.abs(amount));

const fmtDate = (dateStr: string) => {
  if (!dateStr) return '';
  const [y, m, d] = dateStr.split('T')[0].split('-').map(Number);
  if (!y || !m || !d) return dateStr;
  return new Date(y, m - 1, d).toLocaleDateString('en-US', {
    month: 'long', day: 'numeric', year: 'numeric',
  });
};

const normalize = (name: string) => name.toLowerCase().trim();

// ─── Component ────────────────────────────────────────────────────────────────

export const TransactionDetailScreen: React.FC<{ route: any; navigation: any }> = ({
  route,
  navigation,
}) => {
  const insets = useSafeAreaInsets();
  const { transactionId } = route.params as { transactionId: string };

  const [txn, setTxn] = useState<Transaction | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pickerVisible, setPickerVisible] = useState(false);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [t, cats] = await Promise.all([
        transactionService.getTransaction(transactionId),
        budgetService.getCategories(),
      ]);
      setTxn(t);
      setCategories(cats);
    } catch {
      setError('Failed to load transaction.');
    } finally {
      setLoading(false);
    }
  }, [transactionId]);

  useEffect(() => { load(); }, [load]);

  const currentCategory = categories.find(c => c.id === txn?.category_id);
  const merchantDisplay = txn?.merchant_name ?? txn?.name ?? '';

  const onCategorySelect = async (cat: Category) => {
    if (!txn) return;
    setPickerVisible(false);
    setSaving(true);
    try {
      const updated = await transactionService.patchTransaction(txn.id, cat.id);
      setTxn(updated);
      showRulePrompt(cat);
    } catch {
      Alert.alert('Error', 'Could not update category. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const showRulePrompt = (cat: Category) => {
    Alert.alert(
      'Create Rule',
      `Always categorize "${merchantDisplay}" as ${cat.icon} ${cat.name}?`,
      [
        { text: 'No', style: 'cancel' },
        {
          text: 'Yes',
          onPress: () => showApplyToPastPrompt(cat),
        },
      ],
    );
  };

  const showApplyToPastPrompt = (cat: Category) => {
    Alert.alert(
      'Fix Past Transactions',
      `Also update past transactions from "${merchantDisplay}"?`,
      [
        {
          text: 'No, future only',
          onPress: () => saveRule(cat, false),
        },
        {
          text: 'Yes, fix past too',
          onPress: () => saveRule(cat, true),
        },
      ],
    );
  };

  const saveRule = async (cat: Category, applyToPast: boolean) => {
    try {
      const result = await transactionService.createRule(
        normalize(merchantDisplay),
        cat.id,
        applyToPast,
      );
      const msg =
        applyToPast && result.updated_count > 0
          ? `Rule saved — ${result.updated_count} past transaction${result.updated_count === 1 ? '' : 's'} updated.`
          : 'Rule saved.';
      Alert.alert('Done', msg);
    } catch {
      Alert.alert('Error', 'Rule could not be saved. Please try again.');
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { paddingTop: insets.top }]}>
        <ActivityIndicator size="large" color="#2dd4a7" style={{ marginTop: 80 }} />
      </View>
    );
  }

  if (error || !txn) {
    return (
      <View style={[styles.container, { paddingTop: insets.top }]}>
        <TouchableOpacity onPress={() => navigation.goBack()} style={styles.backBtn}>
          <Text style={styles.backBtnText}>← Back</Text>
        </TouchableOpacity>
        <Text style={styles.errorText}>{error ?? 'Transaction not found.'}</Text>
      </View>
    );
  }

  const expenseCategories = categories.filter(c => !c.is_income);
  const incomeCategories = categories.filter(c => c.is_income);

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      {/* Header */}
      <TouchableOpacity onPress={() => navigation.goBack()} style={styles.backBtn}>
        <Text style={styles.backBtnText}>← Back</Text>
      </TouchableOpacity>

      {/* Amount + Merchant */}
      <View style={[GLASS, styles.card]}>
        <Text style={styles.amount}>
          {txn.amount < 0 ? '+' : '-'}{fmt(txn.amount, txn.currency_code)}
        </Text>
        <Text style={styles.merchant}>{merchantDisplay}</Text>
        <Text style={styles.meta}>{fmtDate(txn.date)}</Text>
        {txn.pending && <Text style={styles.badge}>PENDING</Text>}
      </View>

      {/* Category */}
      <TouchableOpacity
        style={[GLASS, styles.card, styles.row]}
        onPress={() => setPickerVisible(true)}
        disabled={saving}
      >
        <Text style={styles.label}>Category</Text>
        <View style={styles.row}>
          {currentCategory ? (
            <Text style={styles.categoryChip}>
              {currentCategory.icon} {currentCategory.name}
            </Text>
          ) : (
            <Text style={styles.categoryChipEmpty}>Tap to categorize</Text>
          )}
          <Text style={styles.chevron}> ›</Text>
        </View>
      </TouchableOpacity>

      {saving && <ActivityIndicator color="#2dd4a7" style={{ marginTop: 12 }} />}

      {/* Category Picker Modal */}
      <Modal
        visible={pickerVisible}
        transparent
        animationType="slide"
        onRequestClose={() => setPickerVisible(false)}
      >
        <TouchableOpacity
          style={styles.backdrop}
          activeOpacity={1}
          onPress={() => setPickerVisible(false)}
        />
        <View style={styles.sheet}>
          <Text style={styles.sheetTitle}>Select Category</Text>
          <FlatList
            data={[
              { title: 'Expenses', data: expenseCategories },
              { title: 'Income', data: incomeCategories },
            ]}
            keyExtractor={item => item.title}
            renderItem={({ item: section }) => (
              <View>
                <Text style={styles.sectionHeader}>{section.title}</Text>
                {section.data.map(cat => (
                  <TouchableOpacity
                    key={cat.id}
                    style={styles.catRow}
                    onPress={() => onCategorySelect(cat)}
                  >
                    <Text style={styles.catIcon}>{cat.icon}</Text>
                    <Text style={styles.catName}>{cat.name}</Text>
                    {txn.category_id === cat.id && (
                      <Text style={styles.checkmark}>✓</Text>
                    )}
                  </TouchableOpacity>
                ))}
              </View>
            )}
          />
        </View>
      </Modal>
    </View>
  );
};

// ─── Styles ──────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f172a', padding: 16 },
  backBtn: { marginBottom: 16 },
  backBtnText: { color: '#2dd4a7', fontSize: 16 },
  card: { padding: 20, marginBottom: 12 },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  amount: { fontSize: 36, fontWeight: '700', color: '#f1f5f9', marginBottom: 4 },
  merchant: { fontSize: 18, color: '#cbd5e1', marginBottom: 4 },
  meta: { fontSize: 14, color: '#64748b' },
  badge: {
    marginTop: 8, alignSelf: 'flex-start',
    backgroundColor: 'rgba(251,191,36,0.2)', color: '#fbbf24',
    paddingHorizontal: 8, paddingVertical: 2, borderRadius: 6, fontSize: 11, fontWeight: '600',
  },
  label: { fontSize: 14, color: '#94a3b8' },
  categoryChip: { fontSize: 15, color: '#f1f5f9' },
  categoryChipEmpty: { fontSize: 15, color: '#64748b' },
  chevron: { fontSize: 20, color: '#64748b' },
  errorText: { color: '#f87171', fontSize: 16, textAlign: 'center', marginTop: 40 },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.5)' },
  sheet: {
    backgroundColor: '#1e293b', borderTopLeftRadius: 20, borderTopRightRadius: 20,
    paddingHorizontal: 16, paddingTop: 16, paddingBottom: 40, maxHeight: '75%',
  },
  sheetTitle: { fontSize: 18, fontWeight: '600', color: '#f1f5f9', marginBottom: 12 },
  sectionHeader: { fontSize: 12, color: '#64748b', marginTop: 12, marginBottom: 4, textTransform: 'uppercase' },
  catRow: {
    flexDirection: 'row', alignItems: 'center',
    paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.05)',
  },
  catIcon: { fontSize: 20, width: 32 },
  catName: { flex: 1, fontSize: 15, color: '#f1f5f9' },
  checkmark: { color: '#2dd4a7', fontSize: 16, fontWeight: '700' },
});
