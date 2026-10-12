#!/usr/bin/env bash
# Builds the signed release APK locally, without an EAS account.
# Output: mobile/dist/mydreamcampus.apk (gitignored), published on GitHub
# Releases under that fixed name so releases/latest/download/... stays stable.
# Signing setup: DEPLOY.md, "Mobil Yayın".
set -euo pipefail

cd "$(dirname "$0")/.."

API_URL="${API_URL:-https://mydreamcampus.madebybaki.com/api}"
GRADLE_PROPS="${GRADLE_USER_HOME:-$HOME/.gradle}/gradle.properties"
OUT_DIR="dist"
OUT_APK="$OUT_DIR/mydreamcampus.apk"

# A debug-signed APK installs fine, but later releases signed with another key
# cannot update it, so refuse to build one for publishing.
store_file="$(sed -n 's/^MDC_UPLOAD_STORE_FILE=//p' "$GRADLE_PROPS" 2>/dev/null | tail -1)"
if [[ -z "$store_file" ]]; then
  echo "error: MDC_UPLOAD_* signing properties are missing from $GRADLE_PROPS" >&2
  exit 1
fi
if [[ ! -f "$store_file" ]]; then
  echo "error: keystore not found: $store_file" >&2
  exit 1
fi

# prebuild rewrites the android/ios scripts in package.json to `expo run:*`,
# which would break the Expo Go dev workflow.
backup="$(mktemp)"
cp package.json "$backup"
trap 'cp "$backup" package.json; rm -f "$backup"' EXIT

# --clean drops android/build too, so a stale JS bundle with another API URL
# can never be reused by Gradle's up-to-date check.
CI=1 ./node_modules/.bin/expo prebuild -p android --clean --no-install

# Shell env wins over mobile/.env, which usually points at a LAN address.
export EXPO_PUBLIC_API_URL="$API_URL"
# x86/x86_64 only serve emulators; dropping them roughly halves the native
# compile time and the APK size for real phones.
(cd android && ./gradlew assembleRelease --no-daemon \
  -PreactNativeArchitectures=armeabi-v7a,arm64-v8a)

mkdir -p "$OUT_DIR"
cp android/app/build/outputs/apk/release/app-release.apk "$OUT_APK"

if ! unzip -p "$OUT_APK" assets/index.android.bundle | grep -aqF "$API_URL"; then
  echo "error: $API_URL is not in the bundled JS; check EXPO_PUBLIC_API_URL" >&2
  exit 1
fi

version="$(node -p "const a=require('./app.json').expo; a.version + ' (' + a.android.versionCode + ')'")"
echo
echo "APK:     $OUT_APK"
echo "Version: $version"
echo "API:     $API_URL"
sha256sum "$OUT_APK"
