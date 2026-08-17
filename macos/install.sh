#!/bin/zsh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
IDENTITY="Apple Development: jzdy.mhq.vip@foxmail.com (9DHH22962C)"
DEST="/Applications/Cursor用量.app"
LABEL="com.local.cursorusage.refresh"
DOMAIN="gui/$(id -u)"
APP_SRC="$ROOT/build/Build/Products/Debug/CursorUsageWidget.app"
APPEX="$DEST/Contents/PlugIns/CursorUsageWidgetExtension.appex"

launchctl bootout "$DOMAIN/$LABEL" 2>/dev/null || true
killall CursorUsageWidget 2>/dev/null || true
killall CursorUsageWidgetExtension 2>/dev/null || true

xcodebuild \
  -project "$ROOT/CursorUsageWidget.xcodeproj" \
  -scheme CursorUsageWidget \
  -configuration Debug \
  -destination "platform=macOS,arch=arm64" \
  -derivedDataPath "$ROOT/build" \
  CODE_SIGN_IDENTITY="-" \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGNING_ALLOWED=NO \
  ENABLE_DEBUG_DYLIB=NO \
  AD_HOC_CODE_SIGNING_ALLOWED=YES \
  CURRENT_PROJECT_VERSION=24 \
  build

if [[ ! -d "$APP_SRC" ]]; then
  echo "build missing: $APP_SRC" >&2
  exit 1
fi

rm -rf "$DEST" "/Applications/CursorUsageWidget.app" "$HOME/Applications/CursorUsageWidget.app"
ditto --norsrc --noextattr --noqtn "$APP_SRC" "$DEST"
xattr -cr "$DEST"

if [[ ! -d "$APPEX" ]]; then
  echo "widget extension missing: $APPEX" >&2
  exit 1
fi

codesign --force --sign "$IDENTITY" \
  --entitlements "$ROOT/Widget/CursorUsageWidgetExtension.entitlements" \
  "$APPEX"
codesign --force --sign "$IDENTITY" \
  --entitlements "$ROOT/App/CursorUsageWidget.entitlements" \
  "$DEST"

/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f -R -trusted "$DEST"
mdimport "$DEST" >/dev/null 2>&1 || true
open "$DEST"
echo "installed: $DEST"
