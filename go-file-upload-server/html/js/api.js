// ============================================================================
// API Client
// ============================================================================

class APIClient {
    static async extractError(response) {
        try {
            const error = await response.json();
            return error.message || error.error || JSON.stringify(error);
        } catch (jsonError) {
            try {
                return await response.text();
            } catch (textError) {
                return `HTTP ${response.status}`;
            }
        }
    }

    static async request(endpoint, options = {}) {
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
            ...AuthManager.getAuthHeader(),
        };

        const config = {
            ...options,
            headers,
        };

        try {
            const response = await fetch(`${API_BASE}${endpoint}`, config);
            
            if (response.status === 401) {
                AuthManager.clearToken();
                window.location.reload();
                return;
            }

            if (!response.ok) {
                const errorMessage = await this.extractError(response);
                throw new Error(errorMessage || `HTTP ${response.status}`);
            }

            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('Content-Type') || '';
            if (contentType.includes('application/json')) {
                return await response.json();
            }

            return await response.text();
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }

    // Authentication
    static async signup(firstName, lastName, email, password) {
        return this.request('/signup', {
            method: 'POST',
            body: JSON.stringify({ firstName, lastName, email, password }),
        });
    }

    static async login(email, password) {
        return this.request('/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
    }

    // File Uploads
    static async getUploads() {
        return this.request('/uploads');
    }

    static async uploadFile(file, folderId = null) {
        const formData = new FormData();
        formData.append('file', file);
        if (folderId) {
            formData.append('folderId', folderId);
        }

        // Don't set Content-Type for FormData - browser will set it with boundary
        const response = await fetch(`${API_BASE}/uploads`, {
            method: 'POST',
            body: formData,
            headers: AuthManager.getAuthHeader(),
        });

        if (response.status === 401) {
            AuthManager.clearToken();
            window.location.reload();
            return;
        }

        if (!response.ok) {
            const errorMessage = await APIClient.extractError(response);
            throw new Error(errorMessage || `HTTP ${response.status}`);
        }

        const contentType = response.headers.get('Content-Type') || '';
        if (contentType.includes('application/json')) {
            return await response.json();
        }

        return await response.text();
    }

    static async deleteUpload(id) {
        return this.request(`/uploads/${id}`, {
            method: 'DELETE',
        });
    }

    static async downloadUpload(id) {
        const response = await fetch(`${API_BASE.replace('/api', '')}/uploads/${id}`, {
            method: 'GET',
            headers: AuthManager.getAuthHeader(),
        });

        if (!response.ok) {
            const errorMessage = await this.extractError(response);
            throw new Error(errorMessage || `HTTP ${response.status}`);
        }

        return await response.blob();
    }

    // Sharing
    static async createShare(fileId, granteeEmail, accessLevel) {
        return this.request('/shares', {
            method: 'POST',
            body: JSON.stringify({ fileId, granteeEmail, accessLevel }),
        });
    }

    static async getShares() {
        return this.request('/shares');
    }

    static async deleteShare(id) {
        return this.request(`/shares/${id}`, {
            method: 'DELETE',
        });
    }
}
