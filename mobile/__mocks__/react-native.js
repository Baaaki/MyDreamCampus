// Minimal react-native mock for unit tests run in Node environment.
// Real native modules are not available, so we expose only what the
// services/lib layer actually imports.

module.exports = {
  Platform: {
    OS: "ios",
    select: (specifics) => specifics.ios ?? specifics.default,
  },
};
