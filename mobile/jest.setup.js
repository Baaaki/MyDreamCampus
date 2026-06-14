// Mock expo-secure-store with an in-memory store so service-layer tests
// can exercise auth/refresh flows without hitting native code.
jest.mock('expo-secure-store', () => {
  const store = new Map();
  return {
    getItemAsync: jest.fn(async (key) => store.get(key) ?? null),
    setItemAsync: jest.fn(async (key, value) => {
      store.set(key, value);
    }),
    deleteItemAsync: jest.fn(async (key) => {
      store.delete(key);
    }),
    __resetStore: () => store.clear(),
  };
});

// Silence the noisy Reanimated logger import side effect for unit tests.
jest.mock('react-native/Libraries/Animated/NativeAnimatedHelper', () => ({}), {
  virtual: true,
});
