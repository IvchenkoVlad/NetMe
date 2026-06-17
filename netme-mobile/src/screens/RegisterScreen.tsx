import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Image,
} from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuth } from '../context/AuthContext';

export const RegisterScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string; confirm?: string }>({});
  const { register } = useAuth();
  const insets = useSafeAreaInsets();

  const validate = () => {
    const e: { email?: string; password?: string; confirm?: string } = {};
    if (!email || !/\S+@\S+\.\S+/.test(email)) e.email = 'Enter a valid email';
    if (!password || password.length < 8) e.password = 'Min 8 characters';
    if (password !== confirm) e.confirm = 'Passwords do not match';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleRegister = async () => {
    if (!validate()) return;
    try {
      setLoading(true);
      await register(email, password);
    } catch (err: any) {
      Alert.alert('Registration Failed', err.response?.data?.message || 'Could not create account');
    } finally {
      setLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.root}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <LinearGradient
        colors={['#2dd4a7', '#1e3a5f', '#0f172a']}
        locations={[0, 0.45, 1]}
        start={{ x: 0, y: 0 }}
        end={{ x: 1, y: 1 }}
        style={StyleSheet.absoluteFill}
      />

      <ScrollView
        contentContainerStyle={[styles.scroll, { paddingTop: insets.top + 12, paddingBottom: insets.bottom + 24 }]}
        keyboardShouldPersistTaps="handled"
        bounces={false}
        showsVerticalScrollIndicator={false}
      >
        {/* Hero — centered logo + brand */}
        <View style={styles.hero}>
          <Image
            source={require('../../assets/logo.png')}
            style={styles.logoImage}
            resizeMode="contain"
          />
          <Text style={styles.brandName}>NetMe</Text>
          <Text style={styles.tagline}>Your money.{'\n'}All in one place.</Text>
        </View>

        {/* Card */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Create Account</Text>

          <View style={styles.field}>
            <Text style={styles.label}>Email</Text>
            <TextInput
              style={[styles.input, errors.email && styles.inputError]}
              placeholder="you@example.com"
              placeholderTextColor="#94a3b8"
              keyboardType="email-address"
              autoCapitalize="none"
              autoCorrect={false}
              value={email}
              onChangeText={setEmail}
              editable={!loading}
            />
            {errors.email && <Text style={styles.errorText}>{errors.email}</Text>}
          </View>

          <View style={styles.field}>
            <Text style={styles.label}>Password</Text>
            <TextInput
              style={[styles.input, errors.password && styles.inputError]}
              placeholder="Min 8 characters"
              placeholderTextColor="#94a3b8"
              secureTextEntry
              value={password}
              onChangeText={setPassword}
              editable={!loading}
            />
            {errors.password && <Text style={styles.errorText}>{errors.password}</Text>}
          </View>

          <View style={styles.field}>
            <Text style={styles.label}>Confirm Password</Text>
            <TextInput
              style={[styles.input, errors.confirm && styles.inputError]}
              placeholder="Repeat password"
              placeholderTextColor="#94a3b8"
              secureTextEntry
              value={confirm}
              onChangeText={setConfirm}
              editable={!loading}
            />
            {errors.confirm && <Text style={styles.errorText}>{errors.confirm}</Text>}
          </View>

          <TouchableOpacity
            style={styles.buttonWrapper}
            onPress={handleRegister}
            disabled={loading}
            activeOpacity={0.85}
          >
            <LinearGradient
              colors={['#4a90e2', '#2dd4a7']}
              start={{ x: 0, y: 0 }}
              end={{ x: 1, y: 0 }}
              style={styles.button}
            >
              {loading
                ? <ActivityIndicator color="#fff" />
                : <Text style={styles.buttonText}>Create Account</Text>
              }
            </LinearGradient>
          </TouchableOpacity>

          <TouchableOpacity onPress={() => navigation.navigate('Login')} disabled={loading}>
            <Text style={styles.switchText}>
              Already have an account?{'  '}
              <Text style={styles.switchLink}>Sign In</Text>
            </Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  scroll: {
    flexGrow: 1,
    paddingHorizontal: 20,
  },
  hero: {
    alignItems: 'center',
    paddingTop: 32,
    paddingBottom: 36,
  },
  logoImage: {
    width: 180,
    height: 180,
    borderRadius: 40,
    marginBottom: 16,
  },
  brandName: {
    color: '#fff',
    fontSize: 22,
    fontWeight: '700',
    letterSpacing: -0.5,
    marginBottom: 5,
  },
  tagline: {
    color: 'rgba(255,255,255,0.5)',
    fontSize: 13,
    lineHeight: 20,
    textAlign: 'center',
  },
  card: {
    backgroundColor: '#f8fafc',
    borderRadius: 28,
    paddingHorizontal: 24,
    paddingTop: 28,
    paddingBottom: 28,
  },
  cardTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#0f172a',
    marginBottom: 22,
  },
  field: {
    marginBottom: 16,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
    color: '#475569',
    marginBottom: 7,
    letterSpacing: 0.2,
  },
  input: {
    backgroundColor: '#fff',
    borderWidth: 1.5,
    borderColor: '#e2e8f0',
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 13,
    fontSize: 15,
    color: '#0f172a',
  },
  inputError: {
    borderColor: '#f43f5e',
    backgroundColor: '#fff1f2',
  },
  errorText: {
    color: '#f43f5e',
    fontSize: 12,
    marginTop: 5,
  },
  buttonWrapper: {
    borderRadius: 14,
    overflow: 'hidden',
    marginTop: 10,
    marginBottom: 18,
  },
  button: {
    paddingVertical: 16,
    alignItems: 'center',
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
    letterSpacing: 0.3,
  },
  switchText: {
    textAlign: 'center',
    color: '#64748b',
    fontSize: 14,
  },
  switchLink: {
    color: '#4a90e2',
    fontWeight: '600',
  },
});
