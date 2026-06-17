import React, { useState } from 'react';
import { View, Text, Pressable, StyleSheet, ActivityIndicator } from 'react-native';
import { StatusBar } from 'expo-status-bar';

const API_URL = 'http://192.168.1.158:8080/api/v1/hello?name=Mobile';

export default function App() {
  const [response, setResponse] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handlePress = async () => {
    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      const res = await fetch(API_URL);
      const data = await res.json();
      setResponse(data);
    } catch (err: any) {
      setError(err.message || 'Failed to connect');
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>NetMe 💰</Text>
      <Text style={styles.subtitle}>Hello World</Text>

      <Pressable
        style={({ pressed }) => [styles.button, pressed && styles.buttonPressed]}
        onPress={handlePress}
        disabled={loading}
      >
        {loading ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.buttonText}>Tap Me</Text>
        )}
      </Pressable>

      {response && (
        <View style={styles.responseBox}>
          <Text style={styles.responseTitle}>✅ Response:</Text>
          <Text style={styles.responseText}>{response.message}</Text>
        </View>
      )}

      {error && (
        <View style={styles.errorBox}>
          <Text style={styles.errorTitle}>❌ Error:</Text>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}

      <View style={styles.statusBox}>
        <Text style={styles.statusText}>
          {response ? '🟢 Connected' : '🟡 Ready'}
        </Text>
      </View>

      <StatusBar style="auto" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
  },
  title: {
    fontSize: 42,
    fontWeight: '800',
    marginBottom: 8,
    color: '#10b981',
    letterSpacing: 0.5,
  },
  subtitle: {
    fontSize: 16,
    color: '#86efac',
    marginBottom: 50,
    fontWeight: '500',
    letterSpacing: 0.3,
  },
  button: {
    backgroundColor: '#059669',
    paddingHorizontal: 40,
    paddingVertical: 16,
    borderRadius: 16,
    marginBottom: 40,
    minWidth: 160,
    shadowColor: '#10b981',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.3,
    shadowRadius: 12,
    elevation: 12,
  },
  buttonPressed: {
    opacity: 0.85,
    transform: [{ scale: 0.98 }],
  },
  buttonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '700',
    textAlign: 'center',
    letterSpacing: 0.5,
  },
  responseBox: {
    backgroundColor: 'rgba(16, 185, 129, 0.1)',
    borderRadius: 16,
    padding: 20,
    marginBottom: 20,
    width: '100%',
    borderLeftWidth: 4,
    borderLeftColor: '#10b981',
    shadowColor: '#10b981',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.15,
    shadowRadius: 8,
    elevation: 8,
  },
  responseTitle: {
    color: '#10b981',
    fontSize: 15,
    fontWeight: '700',
    marginBottom: 8,
    letterSpacing: 0.3,
  },
  responseText: {
    color: '#d1fae5',
    fontSize: 16,
    fontWeight: '500',
    lineHeight: 24,
  },
  errorBox: {
    backgroundColor: 'rgba(239, 68, 68, 0.1)',
    borderRadius: 16,
    padding: 20,
    marginBottom: 20,
    width: '100%',
    borderLeftWidth: 4,
    borderLeftColor: '#ef4444',
    shadowColor: '#ef4444',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.15,
    shadowRadius: 8,
    elevation: 8,
  },
  errorTitle: {
    color: '#ef4444',
    fontSize: 15,
    fontWeight: '700',
    marginBottom: 8,
    letterSpacing: 0.3,
  },
  errorText: {
    color: '#fecaca',
    fontSize: 15,
    fontWeight: '500',
    lineHeight: 22,
  },
  statusBox: {
    backgroundColor: 'rgba(16, 185, 129, 0.15)',
    borderRadius: 16,
    padding: 16,
    minWidth: 200,
    borderWidth: 2,
    borderColor: '#10b981',
    shadowColor: '#10b981',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 8,
  },
  statusText: {
    textAlign: 'center',
    fontSize: 16,
    fontWeight: '700',
    color: '#10b981',
    letterSpacing: 0.3,
  },
});
