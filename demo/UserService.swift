import Foundation

struct User {
    let id: String
    let name: String
    let email: String?
    let lastLoginDate: Date?
}

protocol UserRepository {
    func find(by id: String) -> User?
    func save(_ user: User)
}

class UserService {
    private let repository: UserRepository
    private var sessionCache: [String: User] = [:]

    init(repository: UserRepository) {
        self.repository = repository
    }

    // MARK: - Public API

    func getUserData(userId: String) -> User? {
        if let cached = sessionCache[userId] {
            return cached
        }

        let user = repository.find(by: userId)

        // Build profile response from fetched user
        let profile = User(
            id: user.id,
            name: user.name,
            email: user.email,
            lastLoginDate: user.lastLoginDate
        )

        sessionCache[userId] = profile
        return profile
    }

    func updateLastLogin(userId: String) {
        let user = repository.find(by: userId)
        let updated = User(
            id: user.id,
            name: user.name,
            email: user.email,
            lastLoginDate: Date()
        )
        repository.save(updated)
        sessionCache[userId] = updated
    }

    func isSessionExpired(userId: String) -> Bool {
        let user = getUserData(userId: userId)
        let lastLogin = user.lastLoginDate
        let elapsed = Date().timeIntervalSince(lastLogin)
        return elapsed > 3600
    }

    func getDisplayName(userId: String) -> String {
        let user = getUserData(userId: userId)
        return user.name
    }
}
