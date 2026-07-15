import React from 'react';
import { COLORS } from '../styles/theme';
import {
  Modal,
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Linking,
  SafeAreaView,
} from 'react-native';

interface Props {
  visible: boolean;
  onAccept: () => void;
  onDecline: () => void;
}

// Required by Plaid developer agreement: must show disclosure before opening Plaid Link.
// https://plaid.com/legal/#end-user-privacy-policy
export const PlaidConsentModal: React.FC<Props> = ({ visible, onAccept, onDecline }) => (
  <Modal visible={visible} animationType="slide" presentationStyle="pageSheet" transparent={false}>
    <SafeAreaView style={s.container}>
      <View style={s.content}>
        <View style={s.iconRow}>
          <View style={s.appBadge}><Text style={s.appBadgeText}>N</Text></View>
          <View style={s.connector} />
          <View style={s.plaidBadge}><Text style={s.plaidBadgeText}>P</Text></View>
        </View>

        <Text style={s.title}>Connect your bank</Text>

        <Text style={s.body}>
          <Text style={s.bold}>NetMe</Text> uses{' '}
          <Text style={s.bold}>Plaid</Text> to securely link your bank accounts. Plaid
          will access your account and transaction data on our behalf.
        </Text>

        <Text style={s.body}>
          By tapping <Text style={s.bold}>Continue</Text> you agree to the secure sharing
          of your financial data and acknowledge you have read:
        </Text>

        <View style={s.links}>
          <TouchableOpacity onPress={() => Linking.openURL('https://plaid.com/legal/#end-user-privacy-policy')}>
            <Text style={s.link}>Plaid's End User Privacy Policy →</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => Linking.openURL('https://getnetme.app/privacy')}>
            <Text style={s.link}>NetMe Privacy Policy →</Text>
          </TouchableOpacity>
        </View>
      </View>

      <View style={s.actions}>
        <TouchableOpacity style={s.btnPrimary} onPress={onAccept}>
          <Text style={s.btnPrimaryText}>Continue</Text>
        </TouchableOpacity>
        <TouchableOpacity style={s.btnSecondary} onPress={onDecline}>
          <Text style={s.btnSecondaryText}>Cancel</Text>
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  </Modal>
);

const s = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#fff' },
  content: { flex: 1, padding: 28, justifyContent: 'center' },

  iconRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 28,
  },
  appBadge: {
    width: 52, height: 52, borderRadius: 14,
    backgroundColor: COLORS.bg,
    justifyContent: 'center', alignItems: 'center',
  },
  appBadgeText: { color: COLORS.teal, fontSize: 22, fontWeight: '800' },
  connector: { width: 28, height: 2, backgroundColor: '#e2e8f0', marginHorizontal: 4 },
  plaidBadge: {
    width: 52, height: 52, borderRadius: 14,
    backgroundColor: '#0a2540',
    justifyContent: 'center', alignItems: 'center',
  },
  plaidBadgeText: { color: '#fff', fontSize: 22, fontWeight: '800' },

  title: { fontSize: 22, fontWeight: '700', color: '#1e293b', marginBottom: 16, textAlign: 'center' },
  body: { fontSize: 15, color: '#475569', lineHeight: 22, marginBottom: 14 },
  bold: { fontWeight: '600', color: '#1e293b' },

  links: { marginTop: 8, gap: 10 },
  link: { fontSize: 14, color: COLORS.teal, fontWeight: '500' },

  actions: { padding: 24, gap: 10 },
  btnPrimary: {
    backgroundColor: '#0a2540',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
  },
  btnPrimaryText: { color: '#fff', fontSize: 16, fontWeight: '700' },
  btnSecondary: {
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: 'center',
  },
  btnSecondaryText: { color: '#64748b', fontSize: 15 },
});
