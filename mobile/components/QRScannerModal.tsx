import { useCallback, useEffect, useRef, useState } from 'react';
import { Modal, Pressable, StyleSheet, View } from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { Button, Text, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import type { QRPayload } from '@/types/attendance.types';
import { parseQRPayload } from '@/lib/qr-payload';
import { radius, spacing } from '@/constants/tokens';

type Props = {
  visible: boolean;
  onClose: () => void;
  onScanned: (payload: QRPayload) => void;
};

export function QRScannerModal({ visible, onClose, onScanned }: Props) {
  const { colors } = useTheme();
  const [permission, requestPermission] = useCameraPermissions();
  const [invalid, setInvalid] = useState(false);
  const handledRef = useRef(false);

  useEffect(() => {
    if (visible) {
      handledRef.current = false;
      setInvalid(false);
    }
  }, [visible]);

  const handleBarcode = useCallback(
    ({ data }: { data: string }) => {
      if (handledRef.current) return;
      const payload = parseQRPayload(data);
      if (!payload) {
        // Lock briefly so the camera doesn't fire setInvalid every frame for
        // the same bad QR; release after 1.5s so the user can re-aim without
        // closing the modal.
        handledRef.current = true;
        setInvalid(true);
        setTimeout(() => {
          handledRef.current = false;
        }, 1500);
        return;
      }
      handledRef.current = true;
      onScanned(payload);
    },
    [onScanned],
  );

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose} statusBarTranslucent>
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        {!permission ? (
          <View style={styles.centered}>
            <Text variant="bodyMedium" style={{ color: colors.onBackground }}>
              Kamera hazirlaniyor...
            </Text>
          </View>
        ) : !permission.granted ? (
          <View style={styles.centered}>
            <FontAwesome name="camera" size={48} color={colors.onSurfaceVariant} />
            <Text variant="titleMedium" style={{ color: colors.onBackground, marginTop: spacing.md }}>
              Kamera izni gerekli
            </Text>
            <Text
              variant="bodyMedium"
              style={{ color: colors.onSurfaceVariant, marginTop: spacing.sm, textAlign: 'center' }}
            >
              QR kodu tarayabilmek icin kamera erisimine ihtiyacimiz var.
            </Text>
            <Button mode="contained" onPress={requestPermission} style={{ marginTop: spacing.lg }}>
              Izin ver
            </Button>
            <Button mode="text" onPress={onClose} style={{ marginTop: spacing.sm }}>
              Iptal
            </Button>
          </View>
        ) : (
          <>
            <CameraView
              style={StyleSheet.absoluteFill}
              facing="back"
              barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
              onBarcodeScanned={handleBarcode}
            />
            <View style={styles.overlay} pointerEvents="box-none">
              <View style={styles.frame} />
              <Text variant="bodyMedium" style={styles.hint}>
                QR kodu cerceveye hizalayin
              </Text>
              {invalid && (
                <Text variant="bodySmall" style={styles.invalid}>
                  Gecersiz QR. Ders QR kodunu taradiginizdan emin olun.
                </Text>
              )}
            </View>
            <Pressable
              onPress={onClose}
              style={styles.closeBtn}
              accessibilityRole="button"
              accessibilityLabel="Kapat"
            >
              <FontAwesome name="close" size={22} color="#fff" />
            </Pressable>
          </>
        )}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing.xl,
  },
  overlay: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
  },
  frame: {
    width: 260,
    height: 260,
    borderRadius: radius.lg,
    borderWidth: 3,
    borderColor: '#fff',
  },
  hint: {
    color: '#fff',
    marginTop: spacing.lg,
    textShadowColor: 'rgba(0,0,0,0.5)',
    textShadowRadius: 4,
  },
  invalid: {
    color: '#ffdada',
    marginTop: spacing.sm,
    paddingHorizontal: spacing.lg,
    textAlign: 'center',
  },
  closeBtn: {
    position: 'absolute',
    top: 48,
    right: spacing.lg,
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(0,0,0,0.5)',
    alignItems: 'center',
    justifyContent: 'center',
  },
});
