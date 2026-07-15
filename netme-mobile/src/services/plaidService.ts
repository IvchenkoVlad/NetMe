import { api } from './api';

export const plaidService = {
  createLinkToken: async (): Promise<string> => {
    const { data } = await api.post('/plaid/link-token');
    return data.link_token;
  },

  exchangeToken: async (publicToken: string, institutionId?: string, institutionName?: string) => {
    const { data } = await api.post('/plaid/exchange', {
      public_token: publicToken,
      institution_id: institutionId || '',
      institution_name: institutionName || '',
    });
    return data;
  },

  syncTransactions: async () => {
    const { data } = await api.post('/plaid/sync');
    return data;
  },

  getAccounts: async () => {
    const { data } = await api.get('/accounts');
    return data.accounts || [];
  },

  getTransactions: async (limit = 50, offset = 0, accountId = '', month = '') => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (accountId) params.set('account_id', accountId);
    if (month) params.set('month', month);
    const { data } = await api.get(`/transactions?${params}`);
    return data.transactions || [];
  },

  getItems: async () => {
    const { data } = await api.get('/plaid/items');
    return data.items || [];
  },
};
