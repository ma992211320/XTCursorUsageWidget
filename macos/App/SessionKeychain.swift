import Foundation
import Security

enum SessionKeychainError: LocalizedError {
    case empty
    case saveFailed

    var errorDescription: String? {
        switch self {
        case .empty: return "请粘贴 Cookie"
        case .saveFailed: return "无法写入本机钥匙串"
        }
    }
}

enum SessionKeychain {
    static let service = "cursor-usage.session"
    static let account = "WorkosCursorSessionToken"

    static var hasSession: Bool { read() != nil }

    static func normalize(_ raw: String) -> String {
        var text = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if let range = text.range(of: "WorkosCursorSessionToken=", options: .caseInsensitive) {
            text = String(text[range.upperBound...])
            if let end = text.firstIndex(where: { $0 == ";" || $0.isNewline }) {
                text = String(text[..<end])
            }
        }
        text = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return text.replacingOccurrences(of: "%3A%3A", with: "::")
    }

    static func save(_ pasted: String) throws {
        let value = normalize(pasted)
        guard !value.isEmpty else { throw SessionKeychainError.empty }
        delete()
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: Data(value.utf8),
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        ]
        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else { throw SessionKeychainError.saveFailed }
    }

    static func read() -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else { return nil }
        let value = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
        return (value?.isEmpty == false) ? value : nil
    }

    static func delete() {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
        SecItemDelete(query as CFDictionary)
    }
}
