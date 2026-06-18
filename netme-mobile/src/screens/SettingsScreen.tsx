import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useAuth } from '../context/AuthContext';
import { authService } from '../services/authService';

export default function SettingsScreen() {
  const { user, clearAuth } = useAuth();
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDeleteAccount = () => {
    Alert.alert(
      'Delete Account',
      'This will permanently delete your account and all your data. This cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: confirmDelete,
        },
      ]
    );
  };

  const confirmDelete = async () => {
    try {
      setIsDeleting(true);
      await authService.deleteAccount();
      await clearAuth();
    } catch {
      Alert.alert('Error', 'Failed to delete account. Please try again.');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.email}>{user?.email}</Text>

      {isDeleting ? (
        <ActivityIndicator style={styles.button} />
      ) : (
        <TouchableOpacity style={styles.deleteButton} onPress={handleDeleteAccount}>
          <Text style={styles.deleteButtonText}>Delete Account</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 24,
    backgroundColor: '#fff',
  },
  email: {
    fontSize: 16,
    color: '#333',
    marginBottom: 32,
  },
  button: {
    marginTop: 16,
  },
  deleteButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#e53e3e',
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
    marginTop: 16,
  },
  deleteButtonText: {
    color: '#e53e3e',
    fontSize: 16,
    fontWeight: '600',
  },
});
