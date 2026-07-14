import { authService } from './authService';
import { MonthlyTotal } from './budgetService';

const api = authService.api;

export interface NetWorth {
  assets: number;
  liabilities: number;
  net_worth: number;
}

export interface TopCategory {
  category_id: string;
  name: string;
  icon: string;
  color: string;
  spent: number;
  pct: number;
}

export interface AnalyticsOverview {
  net_worth: NetWorth;
  monthly_totals: MonthlyTotal[];
  top_categories: TopCategory[];
}

export const analyticsService = {
  getOverview: async (): Promise<AnalyticsOverview> => {
    const { data } = await api.get('/analytics/overview');
    return data;
  },
};
