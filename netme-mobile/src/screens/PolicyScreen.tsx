import React from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  SafeAreaView,
  Linking,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';

type PolicyType = 'privacy' | 'terms';

type PolicyRouteParams = { type: PolicyType };

const CONTENT: Record<PolicyType, { title: string; url: string; body: string }> = {
  privacy: {
    title: 'Privacy Policy',
    url: 'https://getnetme.app/privacy',
    body: `Last updated: July 2026

NetMe ("we", "our", "us") is committed to protecting your personal information.

WHAT WE COLLECT
• Account information: email address, authentication provider
• Financial data: bank account names, balances, and transaction history retrieved via Plaid
• Usage data: app interactions for debugging

HOW WE USE YOUR DATA
• To provide the NetMe personal finance service
• To sync your transactions and show budget summaries
• We do not sell your data to third parties

PLAID
We use Plaid Technologies, Inc. to connect to your financial institution. Your banking credentials are entered directly into Plaid's interface and are never transmitted to or stored by NetMe. Plaid's privacy policy is available at plaid.com/legal.

DATA RETENTION
We retain your data for as long as your account is active. Diagnostic logs are purged after 90 days. You may delete your account at any time from Settings, which permanently removes all associated data.

YOUR RIGHTS
Depending on your location, you may have rights under CCPA (California) or GDPR (EU/UK) including: access, correction, deletion, and portability of your data. Contact us at privacy@getnetme.app.

CONTACT
privacy@getnetme.app`,
  },
  terms: {
    title: 'Terms of Service',
    url: 'https://getnetme.app/terms',
    body: `Last updated: July 2026

By using NetMe you agree to these Terms of Service.

SERVICE DESCRIPTION
NetMe is a personal finance application that aggregates your bank account and transaction data via Plaid to help you track spending and budgets.

ELIGIBILITY
You must be 18 years or older and a resident of a country where the service is available.

YOUR ACCOUNT
You are responsible for maintaining the confidentiality of your login credentials. You agree to notify us immediately of any unauthorised access.

ACCEPTABLE USE
You agree not to: reverse-engineer the service, attempt to access other users' data, use the service for unlawful purposes, or interfere with service availability.

FINANCIAL INFORMATION
NetMe provides budgeting tools for informational purposes only. Nothing in the app constitutes financial, investment, or legal advice.

LIMITATION OF LIABILITY
NetMe is provided "as is". We are not liable for any indirect, incidental, or consequential damages arising from your use of the service.

TERMINATION
We may suspend or terminate your account for violations of these Terms. You may delete your account at any time from Settings.

CONTACT
support@getnetme.app`,
  },
};

export const PolicyScreen: React.FC = () => {
  const navigation = useNavigation();
  const route = useRoute<RouteProp<{ Policy: PolicyRouteParams }, 'Policy'>>();
  const type: PolicyType = route.params?.type ?? 'privacy';
  const { title, url, body } = CONTENT[type];

  return (
    <SafeAreaView style={s.container}>
      <View style={s.header}>
        <TouchableOpacity onPress={() => navigation.goBack()} style={s.back}>
          <Text style={s.backText}>‹ Back</Text>
        </TouchableOpacity>
        <Text style={s.title}>{title}</Text>
        <TouchableOpacity onPress={() => Linking.openURL(url)} style={s.webLink}>
          <Text style={s.webLinkText}>Web ↗</Text>
        </TouchableOpacity>
      </View>
      <ScrollView style={s.scroll} contentContainerStyle={s.scrollContent}>
        <Text style={s.body}>{body}</Text>
      </ScrollView>
    </SafeAreaView>
  );
};

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#fff' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  back: { minWidth: 60 },
  backText: { fontSize: 17, color: '#2dd4a7' },
  title: { flex: 1, fontSize: 17, fontWeight: '600', color: '#1e293b', textAlign: 'center' },
  webLink: { minWidth: 60, alignItems: 'flex-end' },
  webLinkText: { fontSize: 14, color: '#2dd4a7' },
  scroll: { flex: 1 },
  scrollContent: { padding: 20, paddingBottom: 48 },
  body: { fontSize: 14, color: '#374151', lineHeight: 22 },
});
