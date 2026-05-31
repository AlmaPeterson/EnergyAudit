// ============================================================================
// Authentication Manager
// ============================================================================

class AuthManager {
    static getToken() {
        return localStorage.getItem(TOKEN_KEY);
    }

    static setToken(token) {
        localStorage.setItem(TOKEN_KEY, token);
    }

    static clearToken() {
        localStorage.removeItem(TOKEN_KEY);
    }

    static isAuthenticated() {
        return !!this.getToken();
    }

    static getAuthHeader() {
        const token = this.getToken();
        return token ? { 'Authorization': `Bearer ${token}` } : {};
    }
}
