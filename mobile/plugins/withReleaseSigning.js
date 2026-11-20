// Release APKs are built locally (no EAS), and `expo prebuild --clean`
// regenerates android/, so the signing setup has to live in a config plugin.
// The keystore and its passwords stay out of this public repo: Gradle reads
// MDC_UPLOAD_* from ~/.gradle/gradle.properties. Without them release builds
// fall back to the debug key, which scripts/build-apk.sh refuses to publish.
const { withAppBuildGradle } = require('expo/config-plugins');

const MARKER = 'MDC_UPLOAD_STORE_FILE';

const RELEASE_SIGNING_CONFIG = `
        release {
            def storeFileProp = findProperty('MDC_UPLOAD_STORE_FILE')
            def storePasswordProp = findProperty('MDC_UPLOAD_STORE_PASSWORD')
            def keyAliasProp = findProperty('MDC_UPLOAD_KEY_ALIAS')
            def keyPasswordProp = findProperty('MDC_UPLOAD_KEY_PASSWORD')
            if (storeFileProp && storePasswordProp && keyAliasProp && keyPasswordProp) {
                storeFile file(storeFileProp)
                storePassword storePasswordProp
                keyAlias keyAliasProp
                keyPassword keyPasswordProp
            }
        }`;

function applyReleaseSigning(contents) {
  if (contents.includes(MARKER)) return contents;

  const withConfig = contents.replace(
    /(signingConfigs \{\n)/,
    `$1${RELEASE_SIGNING_CONFIG.slice(1)}\n`,
  );
  const withBuildType = withConfig.replace(
    /(release \{\n(?:\s*\/\/.*\n)*\s*)signingConfig signingConfigs\.debug/,
    [
      "$1if (!findProperty('MDC_UPLOAD_STORE_FILE') || !findProperty('MDC_UPLOAD_STORE_PASSWORD') || !findProperty('MDC_UPLOAD_KEY_ALIAS') || !findProperty('MDC_UPLOAD_KEY_PASSWORD')) {",
      "                if (gradle.startParameter.taskNames.any { it.toLowerCase().contains('release') }) {",
      "                    throw new GradleException('Missing MDC_UPLOAD_* signing properties for release build')",
      "                }",
      "            }",
      "            signingConfig signingConfigs.release",
    ].join('\n'),
  );

  // Fail the prebuild instead of silently shipping a debug-signed APK when
  // the Expo template changes shape.
  if (withConfig === contents || withBuildType === withConfig) {
    throw new Error('withReleaseSigning: android/app/build.gradle layout not recognised');
  }
  return withBuildType;
}

module.exports = function withReleaseSigning(config) {
  return withAppBuildGradle(config, (cfg) => {
    if (cfg.modResults.language !== 'groovy') {
      throw new Error('withReleaseSigning: only groovy build.gradle is supported');
    }
    cfg.modResults.contents = applyReleaseSigning(cfg.modResults.contents);
    return cfg;
  });
};
