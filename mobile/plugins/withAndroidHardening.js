// Applies release security hardening to the generated AndroidManifest.xml and
// gradle.properties during `expo prebuild`, since mobile/android/ is generated
// and must not be edited manually.
const { withAndroidManifest, withGradleProperties } = require('expo/config-plugins');

function setGradleProp(props, key, value) {
  const existing = props.find((p) => p.type === 'property' && p.key === key);
  if (existing) {
    existing.value = value;
  } else {
    props.push({ type: 'property', key, value });
  }
}

module.exports = function withAndroidHardening(config) {
  config = withAndroidManifest(config, (cfg) => {
    const manifest = cfg.modResults.manifest;
    const app = manifest.application && manifest.application[0];
    if (app && app.$) {
      app.$['android:allowBackup'] = 'false';
      app.$['android:usesCleartextTraffic'] = 'false';
    }
    return cfg;
  });

  config = withGradleProperties(config, (cfg) => {
    setGradleProp(cfg.modResults, 'android.enableMinifyInReleaseBuilds', 'true');
    setGradleProp(cfg.modResults, 'android.enableShrinkResourcesInReleaseBuilds', 'true');
    return cfg;
  });

  return config;
};
