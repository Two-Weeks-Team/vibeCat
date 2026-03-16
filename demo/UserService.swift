import Foundation

/// Defines the core attributes of a user profile.
struct User {
    /// A unique identifier for the user.
    let id: String
    /// The user's full name.
    let name: String
    /// The user's email address (optional).
    let email: String?
    /// The timestamp of the user's most recent login.
    let lastLoginDate: Date?
}

/// A repository interface for managing `User` data persistence.
protocol UserRepository {
    /// Retrieves a user by their unique identifier.
    /// - Parameter id: The unique identifier of the user to fetch.
    /// - Returns: The found `User`, or `nil` if no user exists with the given ID.
    func find(by id: String) -> User?
    
    /// Persists a given user to the repository.
    /// - Parameter user: The `User` object to save.
    func save(_ user: User)
}

/// A service layer for managing user sessions and retrieving profile data.
///
/// `UserService` acts as a facade over a generic `UserRepository`. It provides an
/// in-memory caching mechanism to optimize repeated data access for users.
class UserService {
    private let repository: UserRepository
    
    /// An in-memory cache mapping user IDs to their respective `User` objects.
    private var sessionCache: [String: User] = [:]

    /// Initializes a new instance of `UserService`.
    /// - Parameter repository: A concrete implementation of `UserRepository` used for data persistence.
    init(repository: UserRepository) {
        self.repository = repository
    }

    // MARK: - Public API

    /// Retrieves user data, checking the session cache first before falling back to the repository.
    ///
    /// - Parameter userId: The unique identifier of the user to retrieve.
    /// - Returns: The user's profile, or `nil` if the user cannot be found.
    func getUserData(userId: String) -> User? {
        if let cached = sessionCache[userId] {
            return cached
        }

        // Safely unwrap the user since `find(by:)` returns an optional
        guard let user = repository.find(by: userId) else {
            return nil
        }

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

    /// Updates the `lastLoginDate` of a specific user to the current time and persists the change.
    ///
    /// - Parameter userId: The unique identifier of the user to update.
    func updateLastLogin(userId: String) {
        guard let user = repository.find(by: userId) else { return }
        
        let updated = User(
            id: user.id,
            name: user.name,
            email: user.email,
            lastLoginDate: Date()
        )
        repository.save(updated)
        sessionCache[userId] = updated
    }

    /// Determines whether a user's session has expired.
    ///
    /// A session is considered expired if more than 3600 seconds (1 hour) have elapsed
    /// since the user's `lastLoginDate`. If the user has no login date, the session
    /// is also treated as expired.
    ///
    /// - Parameter userId: The unique identifier of the user.
    /// - Returns: `true` if the session has expired; otherwise, `false`.
    func isSessionExpired(userId: String) -> Bool {
        guard let user = getUserData(userId: userId),
              let lastLogin = user.lastLoginDate else {
            return true
        }
        
        let elapsed = Date().timeIntervalSince(lastLogin)
        return elapsed > 3600
    }

    /// Retrieves the display name of a given user.
    ///
    /// - Parameter userId: The unique identifier of the user.
    /// - Returns: The user's name, or a default empty string if the user is not found.
    func getDisplayName(userId: String) -> String {
        guard let user = getUserData(userId: userId) else { return "" }
        return user.name
    }
}
