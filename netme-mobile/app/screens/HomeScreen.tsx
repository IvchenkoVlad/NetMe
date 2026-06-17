import React, { useState } from 'react';
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native';
import { helloAPI } from '../utils/apiClient';

// Simple button component for web/mobile
const Button = ({ onPress, disabled, children }: any) => (
  <Text
    onPress={onPress}
    style={[styles.button, disabled && styles.buttonDisabled]}
  >
    {children}
  </Text>
);

interface HelloResponse {
  message: string;
  backend?: string;
  status?: string;
}

export default function HomeScreen() {
  const [response, setResponse] = useState<HelloResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handlePressHello = async () => {
    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      const data = await helloAPI.getHello('Mobile App');
      setResponse(data);
    } catch (err: any) {
      setError(err.message || 'Failed to connect to backend');
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>NetMe — Hello World</Text>
      <Text style={styles.subtitle}>Test Backend Connection</Text>

      {/* Main Button */}
      <Button onPress={handlePressHello} disabled={loading}>
        {loading ? '⏳ Loading...' : '👇 Click Me — Call Backend'}
      </Button>

      {/* Response Display */}
      {response && (
        <View style={styles.responseBox}>
          <Text style={styles.responseTitle}>✅ Backend Response:</Text>
          <Text style={styles.responseText}>{response.message}</Text>
          {response.backend && (
            <Text style={styles.responseSubtext}>{response.backend}</Text>
          )}
          {response.status && (
            <Text style={styles.responseSubtext}>Status: {response.status}</Text>
          )}
        </View>
      )}

      {/* Error Display */}
      {error && (
        <View style={styles.errorBox}>
          <Text style={styles.errorTitle}>❌ Error:</Text>
          <Text style={styles.errorText}>{error}</Text>
          <Text style={styles.errorHint}>
            Make sure backend is running: go run cmd/server/main.go
          </Text>
        </View>
      )}

      {/* Connection Info */}
      <View style={styles.infoBox}>
        <Text style={styles.infoLabel}>Connection Status:</Text>
        <Text style={styles.infoValue}>
          {response ? '🟢 Connected' : '🟡 Not tested yet'}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f5f5f5',
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#333',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
    marginBottom: 40,
  },
  button: {
    backgroundColor: '#007AFF',
    color: '#fff',
    paddingHorizontal: 24,
    paddingVertical: 16,
    borderRadius: 8,
    minWidth: 200,
    textAlign: 'center',
    marginBottom: 30,
    fontSize: 16,
    fontWeight: '600',
    cursor: 'pointer',
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  responseBox: {
    backgroundColor: '#d4edda',
    borderColor: '#28a745',
    borderWidth: 1,
    borderRadius: 8,
    padding: 16,
    marginBottom: 20,
    width: '100%',
  },
  responseTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#155724',
    marginBottom: 8,
  },
  responseText: {
    fontSize: 16,
    color: '#155724',
    fontWeight: '500',
  },
  responseSubtext: {
    fontSize: 12,
    color: '#155724',
    marginTop: 4,
    opacity: 0.8,
  },
  errorBox: {
    backgroundColor: '#f8d7da',
    borderColor: '#f5c6cb',
    borderWidth: 1,
    borderRadius: 8,
    padding: 16,
    marginBottom: 20,
    width: '100%',
  },
  errorTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#721c24',
    marginBottom: 8,
  },
  errorText: {
    fontSize: 14,
    color: '#721c24',
  },
  errorHint: {
    fontSize: 11,
    color: '#721c24',
    marginTop: 8,
    opacity: 0.8,
    fontStyle: 'italic',
  },
  infoBox: {
    backgroundColor: '#fff',
    borderRadius: 8,
    padding: 12,
    width: '100%',
    alignItems: 'center',
  },
  infoLabel: {
    fontSize: 12,
    color: '#666',
    marginBottom: 4,
  },
  infoValue: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
  },
});
