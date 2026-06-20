import { authService } from './authService';

const api = authService.api;

export interface Category {
  id: string;
  name: string;
  icon: string;
  color: string;
  is_income: boolean;
  sort_order: number;
  plaid_primary_categories: string[];
}

export interface CategorySummary extends Category {
  spent: number;
  budget_limit: number;
  transaction_count: number;
}

export interface BudgetSummary {
  month: string;
  income: number;
  spending: number;
  categories: CategorySummary[];
}

export interface MonthlyTotal {
  month: string;
  spending: number;
  income: number;
}

export const budgetService = {
  getSummary: async (month: string): Promise<BudgetSummary> => {
    const { data } = await api.get(`/budget/summary?month=${month}`);
    return data;
  },

  getHistory: async (months = 6): Promise<MonthlyTotal[]> => {
    const { data } = await api.get(`/budget/history?months=${months}`);
    return data.history || [];
  },

  getCategories: async (): Promise<Category[]> => {
    const { data } = await api.get('/categories');
    return data.categories || [];
  },

  createCategory: async (name: string, icon: string, color: string, isIncome: boolean, plaid: string[]): Promise<Category> => {
    const { data } = await api.post('/categories', { name, icon, color, is_income: isIncome, plaid_primary_categories: plaid });
    return data;
  },

  updateCategory: async (id: string, name: string, icon: string, color: string, plaid: string[]): Promise<Category> => {
    const { data } = await api.put(`/categories/${id}`, { name, icon, color, plaid_primary_categories: plaid });
    return data;
  },

  deleteCategory: async (id: string): Promise<void> => {
    await api.delete(`/categories/${id}`);
  },

  setBudget: async (categoryId: string, month: string, amount: number): Promise<void> => {
    await api.put(`/budget/${categoryId}?month=${month}`, { amount });
  },
};
