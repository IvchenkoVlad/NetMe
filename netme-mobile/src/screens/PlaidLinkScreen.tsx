import React, { useEffect, useRef, useState } from 'react';
import {
  Modal,
  View,
  Text,
  TouchableOpacity,
  ActivityIndicator,
  StyleSheet,
  SafeAreaView,
} from 'react-native';
import WebView from 'react-native-webview';
import { plaidService } from '../services/plaidService';
import { COLORS } from '../styles/theme';

const API_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/v1';

interface Props {
  visible: boolean;
  onSuccess: (publicToken: string, institutionId?: string, institutionName?: string) => void;
  onClose: () => void;
}

export const PlaidLinkModal: React.FC<Props> = ({ visible, onSuccess, onClose }) => {
  const [linkToken, setLinkToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const webViewRef = useRef<WebView>(null);

  useEffect(() => {
    if (visible) {
      setLinkToken(null);
      setLoading(true);
      setError(null);
      plaidService
        .createLinkToken()
        .then(setLinkToken)
        .catch((e) => setError(e.message))
        .finally(() => setLoading(false));
    }
  }, [visible]);

  const handleMessage = (event: { nativeEvent: { data: string } }) => {
    const raw = event.nativeEvent.data;
    console.log('[Plaid] message received:', raw);
    try {
      const msg = JSON.parse(raw);
      if (msg.event === 'success') {
        console.log('[Plaid] success, public_token:', msg.public_token);
        onSuccess(msg.public_token, msg.institution_id, msg.institution_name);
      } else if (msg.event === 'exit') {
        console.log('[Plaid] exit');
        onClose();
      }
    } catch (e) {
      console.log('[Plaid] non-JSON:', raw);
    }
  };

  const handleShouldStartLoad = (req: { url: string }) => {
    const url = req.url;
    console.log('[Plaid] navigation:', url);
    if (url.startsWith('plaidlink://callback')) {
      try {
        const qs = url.split('?data=')[1] || '';
        const msg = JSON.parse(decodeURIComponent(qs));
        if (msg.event === 'success') {
          onSuccess(msg.public_token, msg.institution_id, msg.institution_name);
        } else {
          onClose();
        }
      } catch {
        onClose();
      }
      return false;
    }
    return true;
  };

  const pageUrl = linkToken ? `${API_URL}/plaid/link-page?token=${linkToken}` : null;

  return (
    <Modal visible={visible} animationType="slide" presentationStyle="pageSheet">
      <SafeAreaView style={styles.container}>
        <View style={styles.header}>
          <Text style={styles.headerTitle}>Connect Bank</Text>
          <TouchableOpacity onPress={onClose} style={styles.closeButton}>
            <Text style={styles.closeText}>Cancel</Text>
          </TouchableOpacity>
        </View>

        {(loading || !pageUrl) && !error && (
          <View style={styles.center}>
            <ActivityIndicator size="large" color={COLORS.teal} />
            <Text style={styles.loadingText}>Preparing secure connection…</Text>
          </View>
        )}

        {error && (
          <View style={styles.center}>
            <Text style={styles.errorText}>{error}</Text>
            <TouchableOpacity style={styles.retryButton} onPress={onClose}>
              <Text style={styles.retryText}>Close</Text>
            </TouchableOpacity>
          </View>
        )}

        {pageUrl && !loading && !error && (
          <WebView
            ref={webViewRef}
            source={{ uri: pageUrl }}
            onMessage={handleMessage}
            onShouldStartLoadWithRequest={handleShouldStartLoad}
            style={styles.webview}
            javaScriptEnabled
            domStorageEnabled
            originWhitelist={['http://*', 'https://*', 'plaidlink://*']}
            onError={(e) => {
              console.log('[Plaid] WebView error:', e.nativeEvent);
              setError('Failed to load Plaid. Check your connection.');
            }}
          />
        )}
      </SafeAreaView>
    </Modal>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#fff' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  headerTitle: { fontSize: 17, fontWeight: '600', color: COLORS.navy },
  closeButton: { padding: 4 },
  closeText: { fontSize: 16, color: COLORS.teal, fontWeight: '500' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24 },
  loadingText: { marginTop: 12, color: '#666', fontSize: 15 },
  errorText: { color: '#e53e3e', fontSize: 15, textAlign: 'center', marginBottom: 16 },
  retryButton: {
    backgroundColor: COLORS.teal,
    paddingHorizontal: 24,
    paddingVertical: 10,
    borderRadius: 8,
  },
  retryText: { color: '#fff', fontWeight: '600' },
  webview: { flex: 1 },
});
