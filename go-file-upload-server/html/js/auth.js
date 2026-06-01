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

    static parseJwt(token) {
        try {
            const payload = token.split('.')[1];
            const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
            const padded = base64.padEnd(base64.length + (4 - (base64.length % 4)) % 4, '=');
            const decoded = atob(padded);
            const json = decodeURIComponent(Array.from(decoded, (c) => '%'+('00'+c.charCodeAt(0).toString(16)).slice(-2)).join(''));
            return JSON.parse(json);
        } catch {
            return null;
        }
    }

    static getUserEmail() {
        const token = this.getToken();
        if (!token) return null;
        const payload = this.parseJwt(token);
        return payload?.email || null;
    }

    static getUserId() {
        const token = this.getToken();
        if (!token) return null;
        const payload = this.parseJwt(token);
        return payload?.sub || null;
    }
}
