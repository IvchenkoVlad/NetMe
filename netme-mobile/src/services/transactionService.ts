import { api } from './api';

export interface Transaction {
  id: string;
  user_id: string;
  account_id: string;
  plaid_transaction_id: string;
  amount: number;
  currency_code: string;
  name: string;
  merchant_name?: string;
  date: string;
  authorized_date?: string;
  category?: string;
  category_detailed?: string;
  payment_channel?: string;
  pending: boolean;
  category_id?: string;
}

export interface CategoryRule {
  id: string;
  normalized_merchant: string;
  category_id: string;
  category?: {
    id: string;
    name: string;
    icon: string;
    color: string;
    is_income: boolean;
  };
  created_at: string;
}

export const transactionService = {
  getTransaction: async (id: string): Promise<Transaction> => {
    const { data } = await api.get(`/transactions/${id}`);
    return data.transaction;
  },

  patchTransaction: async (id: string, categoryId: string): Promise<Transaction> => {
    const { data } = await api.patch(`/transactions/${id}`, { category_id: categoryId });
    return data.transaction;
  },

  createRule: async (
    normalizedMerchant: string,
    categoryId: string,
    applyToPast: boolean,
  ): Promise<{ rule: CategoryRule; updated_count: number }> => {
    const { data } = await api.post('/rules', {
      normalized_merchant: normalizedMerchant,
      category_id: categoryId,
      apply_to_past: applyToPast,
    });
    return data;
  },

  listRules: async (): Promise<CategoryRule[]> => {
    const { data } = await api.get('/rules');
    return data.rules ?? [];
  },
};
