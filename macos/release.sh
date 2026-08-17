#!/bin/zsh
set -euo pipefail

# Developer ID 签名并打 DMG，供 GitHub Release。
# 不写 LaunchAgent、不碰钥匙串里的 Cookie。
# 公证：先在本机执行
#   xcrun notarytool store-credentials "cursor-usage-notary" --apple-id "你的Apple ID" --team-id 66V43K97HH
# 再运行：NOTARY_PROFILE=cursor-usage-notary ./release.sh

ROOT="$(cd "$(dirname "$0")" && pwd)"
IDENTITY="Developer ID Application: haiqing ma (66V43K97HH)"
VERSION="1.0"
BUILD="22"
DIST="$ROOT/dist"
STAGE="$DIST/dmg-root"
APP_NAME="Cursor用量.app"
APP_SRC="$ROOT/build/Build/Products/Release/CursorUsageWidget.app"
SIGN_ROOT="$(mktemp -d /tmp/cursor-usage-sign.XXXXXX)"
APP_DST="$SIGN_ROOT/$APP_NAME"
APPEX_REL="Contents/PlugIns/CursorUsageWidgetExtension.appex"
DMG="$DIST/CursorUsageWidget-$VERSION.dmg"

if ! security find-identity -v -p codesigning | grep -F -q "$IDENTITY"; then
  echo "missing $IDENTITY" >&2
  exit 1
fi

xcodebuild \
  -project "$ROOT/CursorUsageWidget.xcodeproj" \
  -scheme CursorUsageWidget \
  -configuration Release \
  -destination "platform=macOS,arch=arm64" \
  -derivedDataPath "$ROOT/build" \
  CODE_SIGN_IDENTITY="-" \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGNING_ALLOWED=NO \
  ENABLE_DEBUG_DYLIB=NO \
  AD_HOC_CODE_SIGNING_ALLOWED=YES \
  CURRENT_PROJECT_VERSION="$BUILD" \
  MARKETING_VERSION="$VERSION" \
  build

if [[ ! -d "$APP_SRC" ]]; then
  echo "build missing: $APP_SRC" >&2
  exit 1
fi

rm -rf "$DIST"
mkdir -p "$DIST"
ditto --norsrc --noextattr --noqtn "$APP_SRC" "$APP_DST"
xattr -cr "$APP_DST" 2>/dev/null || true
dot_clean -m "$APP_DST" 2>/dev/null || true

if [[ ! -d "$APP_DST/$APPEX_REL" ]]; then
  echo "widget extension missing" >&2
  exit 1
fi

sign_bundle() {
  local dest="$1"
  local entitlements="$2"
  local attempt
  for attempt in 1 2 3 4 5; do
    xattr -cr "$dest" 2>/dev/null || true
    if codesign --force --options runtime --timestamp \
      --sign "$IDENTITY" \
      --entitlements "$entitlements" \
      "$dest"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

# 先签 appex，再签 app。不要 --deep。
sign_bundle "$APP_DST/$APPEX_REL" "$ROOT/Widget/CursorUsageWidgetExtension.entitlements"
sign_bundle "$APP_DST" "$ROOT/App/CursorUsageWidget.entitlements"

codesign --verify --strict "$APP_DST/$APPEX_REL"
codesign --verify --strict "$APP_DST"
ditto --norsrc --noextattr --noqtn "$APP_DST" "$DIST/$APP_NAME"

mkdir -p "$STAGE"
ditto --norsrc --noextattr --noqtn "$APP_DST" "$STAGE/$APP_NAME"
ln -s /Applications "$STAGE/Applications"

hdiutil create \
  -volname "Cursor 用量" \
  -srcfolder "$STAGE" \
  -ov -format UDZO \
  "$DMG"
rm -rf "$STAGE"

codesign --force --timestamp --sign "$IDENTITY" "$DMG"
codesign --verify --strict "$DMG"

shasum -a 256 "$DMG" | tee "$DMG.sha256"
echo "signed dmg: $DMG"

if [[ -n "${NOTARY_PROFILE:-}" ]]; then
  xcrun notarytool submit "$DMG" --keychain-profile "$NOTARY_PROFILE" --wait
  xcrun stapler staple "$DMG"
  xcrun stapler validate "$DMG"
  echo "notarized: $DMG"
else
  echo "not notarized. store credentials, then:"
  echo "  NOTARY_PROFILE=cursor-usage-notary $ROOT/release.sh"
fi
rm -rf "$SIGN_ROOT"
